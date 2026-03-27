package git

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var ignoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
}

type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	repoPath     string
	Events       chan struct{}
	done         chan struct{}
	mu           sync.Mutex
	debounceTimer *time.Timer
}

func NewWatcher(repoPath string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsw,
		repoPath:  repoPath,
		Events:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	// Walk directory tree and add all directories
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		// Skip symlinks to avoid infinite loops
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		base := filepath.Base(path)
		if ignoreDirs[base] && path != repoPath {
			return filepath.SkipDir
		}
		return fsw.Add(path)
	})
	if err != nil {
		fsw.Close()
		return nil, err
	}

	// Also watch .git/index for staging changes
	gitIndex := filepath.Join(repoPath, ".git", "index")
	if _, err := os.Stat(gitIndex); err == nil {
		fsw.Add(filepath.Dir(gitIndex))
	}

	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Skip irrelevant events
			rel, _ := filepath.Rel(w.repoPath, event.Name)
			parts := strings.Split(rel, string(filepath.Separator))

			// Allow .git/index changes through, skip everything else in .git
			isGitIndex := len(parts) >= 2 && parts[0] == ".git" && parts[1] == "index"
			isGitDir := len(parts) >= 1 && parts[0] == ".git"
			if isGitDir && !isGitIndex {
				continue
			}

			// Skip ignored dirs
			skip := false
			for _, part := range parts {
				if ignoreDirs[part] {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			// Add new directories to watch
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					base := filepath.Base(event.Name)
					if !ignoreDirs[base] {
						w.fsWatcher.Add(event.Name)
					}
				}
			}

			// Debounce: reset timer on each event
			w.mu.Lock()
			if w.debounceTimer != nil {
				w.debounceTimer.Stop()
			}
			w.debounceTimer = time.AfterFunc(300*time.Millisecond, func() {
				select {
				case w.Events <- struct{}{}:
				default:
				}
			})
			w.mu.Unlock()

		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

		case <-w.done:
			return
		}
	}
}

func (w *Watcher) Close() {
	w.mu.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.mu.Unlock()
	close(w.done)
	w.fsWatcher.Close()
}
