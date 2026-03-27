package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viktorfroberg/koll/internal/splitpane"
	"github.com/viktorfroberg/koll/internal/ui"
	"github.com/viktorfroberg/koll/internal/updater"
)

var version = "dev"

func main() {
	split := flag.Bool("split", false, "Open in a split pane (auto-detects terminal multiplexer)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	update := flag.Bool("update", false, "Update koll to the latest version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "koll - real-time git diff viewer\n\n")
		fmt.Fprintf(os.Stderr, "Usage: koll [flags] [path]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("koll %s\n", version)
		return
	}

	if *update {
		if err := updater.Update(version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	path := flag.Arg(0)
	if path == "" {
		path = "."
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	path = absPath

	// --split launches koll in a new pane, no git validation needed here
	// (the spawned koll instance will validate)
	if *split {
		if err := splitpane.Launch(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validate it's a git repo
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s is not a git repository\n", path)
		os.Exit(1)
	}
	repoPath := strings.TrimSpace(string(out))

	updateCh := updater.CheckAsync(version)
	m := ui.NewModel(repoPath, updateCh, version)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
