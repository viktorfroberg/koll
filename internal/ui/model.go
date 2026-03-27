package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viktorfroberg/koll/internal/git"
	"github.com/viktorfroberg/koll/internal/types"
)

type scrollTickMsg struct{}
type flashClearMsg struct{}

type refreshMsg struct{}
type diffLoadedMsg struct {
	path    string
	lines   []types.DiffLine
	diffErr string
}
type updateAvailableMsg struct {
	version string
}

type Model struct {
	repoPath        string
	repoName        string
	files           []types.FileChange
	cursor          int
	offset          int     // current scroll offset (rendered)
	targetOffset    int     // where we're scrolling to
	scrolling       bool    // animation in progress
	width           int
	height          int
	filter          types.FilterMode
	watcher         *git.Watcher
	err             error
	updateChan      <-chan string
	updateAvailable string
	version         string
	firstLoad       bool
	showHelp        bool
	flashMsg        string
}

func NewModel(repoPath string, updateChan <-chan string, version string) Model {
	return Model{
		repoPath:   repoPath,
		repoName:   filepath.Base(repoPath),
		updateChan: updateChan,
		firstLoad:  true,
		version:    version,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadChanges(),
		m.startWatcher(),
		m.checkForUpdate(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Help overlay intercepts all keys
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			m.showHelp = true
			return m, nil

		case key.Matches(msg, keys.Yank):
			visible := m.visibleFiles()
			if m.cursor >= 0 && m.cursor < len(visible) {
				path := visible[m.cursor].Path
				copyToClipboard(path)
				m.flashMsg = "copied: " + path
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return flashClearMsg{}
				})
			}
			return m, nil

		case key.Matches(msg, keys.ScrollUp):
			return m, m.smoothScroll(-1)

		case key.Matches(msg, keys.ScrollDown):
			return m, m.smoothScroll(1)

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
				return m, m.ensureVisibleSmooth()
			}
			return m, nil

		case key.Matches(msg, keys.Down):
			visible := m.visibleFiles()
			if m.cursor < len(visible)-1 {
				m.cursor++
				return m, m.ensureVisibleSmooth()
			}
			return m, nil

		case key.Matches(msg, keys.Toggle):
			visible := m.visibleFiles()
			if m.cursor >= 0 && m.cursor < len(visible) {
				path := visible[m.cursor].Path
				for i := range m.files {
					if m.files[i].Path == path {
						m.files[i].Expanded = !m.files[i].Expanded
						if m.files[i].Expanded && !m.files[i].DiffLoaded {
							return m, m.loadDiff(m.files[i].Path, m.files[i].Staged && !m.files[i].Unstaged)
						}
						break
					}
				}
			}
			return m, nil

		case key.Matches(msg, keys.All):
			// Expand all visible files
			visible := m.visibleFiles()
			var cmds []tea.Cmd
			for _, vf := range visible {
				for i := range m.files {
					if m.files[i].Path == vf.Path {
						m.files[i].Expanded = true
						if !m.files[i].DiffLoaded {
							cmds = append(cmds, m.loadDiff(m.files[i].Path, m.files[i].Staged && !m.files[i].Unstaged))
						}
						break
					}
				}
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.Collapse):
			for i := range m.files {
				m.files[i].Expanded = false
			}
			return m, nil

		case key.Matches(msg, keys.Filter):
			m.filter = (m.filter + 1) % 3
			m.cursor = 0
			m.offset = 0
			return m, nil

		case key.Matches(msg, keys.Refresh):
			return m, m.loadChanges()

		case key.Matches(msg, keys.PageDown):
			return m, m.smoothScroll(m.availHeight())

		case key.Matches(msg, keys.PageUp):
			return m, m.smoothScroll(-m.availHeight())

		case key.Matches(msg, keys.HalfDown):
			return m, m.smoothScroll(m.availHeight() / 2)

		case key.Matches(msg, keys.HalfUp):
			return m, m.smoothScroll(-m.availHeight() / 2)

		case key.Matches(msg, keys.Top):
			m.cursor = 0
			m.targetOffset = 0
			return m, m.smoothScroll(0)

		case key.Matches(msg, keys.Bottom):
			visible := m.visibleFiles()
			if len(visible) > 0 {
				m.cursor = len(visible) - 1
			}
			m.targetOffset = m.totalLines() - m.availHeight()
			return m, m.smoothScroll(0)
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			return m, m.smoothScroll(-3)
		case tea.MouseButtonWheelDown:
			return m, m.smoothScroll(3)
		}

	case scrollTickMsg:
		if m.offset == m.targetOffset {
			m.scrolling = false
			return m, nil
		}
		// Move toward target — ease with larger steps when far away
		diff := m.targetOffset - m.offset
		step := diff / 3
		if step == 0 {
			if diff > 0 {
				step = 1
			} else {
				step = -1
			}
		}
		m.offset += step
		m.clampOffset()
		return m, scrollTick()

	case refreshMsg:
		return m, tea.Batch(m.loadChanges(), m.waitForChange())

	case []types.FileChange:
		expandedMap := make(map[string]bool)
		diffMap := make(map[string][]types.DiffLine)
		for _, f := range m.files {
			expandedMap[f.Path] = f.Expanded
			if f.DiffLoaded {
				diffMap[f.Path] = f.DiffLines
			}
		}
		m.files = msg
		for i := range m.files {
			if m.firstLoad {
				// Auto-expand all files on first load
				m.files[i].Expanded = true
			} else if exp, ok := expandedMap[m.files[i].Path]; ok {
				m.files[i].Expanded = exp
			}
			if lines, ok := diffMap[m.files[i].Path]; ok {
				m.files[i].DiffLines = lines
				m.files[i].DiffLoaded = true
				m.files[i].Additions, m.files[i].Deletions = countChanges(lines)
			}
		}
		m.firstLoad = false
		// Load diffs for expanded files
		var cmds []tea.Cmd
		for _, f := range m.files {
			if f.Expanded {
				cmds = append(cmds, m.loadDiff(f.Path, f.Staged && !f.Unstaged))
			}
		}
		visible := m.visibleFiles()
		if m.cursor >= len(visible) && len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
		return m, tea.Batch(cmds...)

	case diffLoadedMsg:
		for i := range m.files {
			if m.files[i].Path == msg.path {
				m.files[i].DiffLines = msg.lines
				m.files[i].DiffLoaded = true
				m.files[i].DiffError = msg.diffErr
				m.files[i].Additions, m.files[i].Deletions = countChanges(msg.lines)
				break
			}
		}
		return m, nil

	case flashClearMsg:
		m.flashMsg = ""
		return m, nil

	case watcherStartedMsg:
		m.watcher = msg.watcher
		return m, m.waitForChange()

	case updateAvailableMsg:
		m.updateAvailable = msg.version
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return m.renderHelp()
	}

	var b strings.Builder

	// Header
	header := headerStyle.Render("koll") + " " + headerDim.Render(m.repoName)
	b.WriteString(" " + header)
	b.WriteString("\n")

	availHeight := m.availHeight()

	// Build all content lines
	visible := m.visibleFiles()
	var lines []string

	for idx, f := range visible {
		isSelected := idx == m.cursor

		// File line
		fileLine := m.renderFileLine(f, isSelected)
		lines = append(lines, fileLine)

		// Loading indicator
		if f.Expanded && !f.DiffLoaded && f.DiffError == "" {
			lines = append(lines, diffIndent+loadingStyle.Render("loading..."))
		}

		// Error indicator
		if f.Expanded && f.DiffError != "" {
			lines = append(lines, diffIndent+errorStyle.Render("error: "+f.DiffError))
		}

		// Diff lines if expanded
		if f.Expanded && f.DiffLoaded {
			for _, dl := range f.DiffLines {
				diffLine := m.renderDiffLine(dl)
				lines = append(lines, diffLine)
			}
			// Blank line between files for readability
			if idx < len(visible)-1 {
				lines = append(lines, "")
			}
		}
	}

	if len(lines) == 0 {
		lines = append(lines, "  No changes")
	}

	// Apply scroll offset
	start := m.offset
	if start >= len(lines) {
		start = 0
	}
	end := start + availHeight
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Pad remaining space
	rendered := end - start
	for i := rendered; i < availHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	totalAdd, totalDel := 0, 0
	for _, f := range m.files {
		totalAdd += f.Additions
		totalDel += f.Deletions
	}

	filterStr := ""
	if m.filter != types.FilterAll {
		filterStr = fmt.Sprintf(" [%s]", m.filter)
	}

	// Flash message or normal footer
	if m.flashMsg != "" {
		b.WriteString(flashStyle.Render(" " + m.flashMsg))
	} else {
		footer := footerStyle.Render(fmt.Sprintf(" %d files", len(visible))) +
			filterStr +
			footerStyle.Render(" · ") +
			diffAdded.Render(fmt.Sprintf("+%d", totalAdd)) +
			diffRemoved.Render(fmt.Sprintf(" -%d", totalDel)) +
			footerStyle.Render("  ?·help")
		b.WriteString(footer)
	}

	// Version / update line
	b.WriteString("\n")
	if m.updateAvailable != "" {
		b.WriteString(updateNotice.Render(fmt.Sprintf(" %s available · run: koll --update", m.updateAvailable)))
	} else {
		b.WriteString(footerStyle.Render(fmt.Sprintf(" koll %s", m.version)))
	}

	return b.String()
}

func (m Model) renderHelp() string {
	var b strings.Builder

	title := headerStyle.Render("koll") + " " + headerDim.Render("keybindings")
	b.WriteString(" " + title + "\n\n")

	bindings := []struct{ key, desc string }{
		{"j / k", "jump between files"},
		{"↑ / ↓", "scroll line by line"},
		{"enter / l", "toggle file diff"},
		{"a", "expand all files"},
		{"c", "collapse all files"},
		{"s", "cycle filter: all → unstaged → staged"},
		{"y", "copy file path to clipboard"},
		{"ctrl+d / ctrl+u", "half page scroll"},
		{"pgdn / pgup", "full page scroll"},
		{"g / G", "jump to top / bottom"},
		{"r", "force refresh"},
		{"q / ctrl+c", "quit"},
	}

	for _, bind := range bindings {
		line := "  " + helpKeyStyle.Render(fmt.Sprintf("%-18s", bind.key)) + helpStyle.Render(bind.desc)
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpDimStyle.Render("  press any key to close"))

	return b.String()
}

func (m Model) renderFileLine(f types.FileChange, isSelected bool) string {
	cursor := "  "
	if isSelected {
		cursor = selectedStyle.Render("> ")
	}

	// Status indicator with color
	var statusStyle lipgloss.Style
	switch f.Status {
	case types.Modified:
		statusStyle = statusModified
	case types.Added:
		statusStyle = statusAdded
	case types.Deleted:
		statusStyle = statusDeleted
	case types.Renamed:
		statusStyle = statusRenamed
	case types.Untracked:
		statusStyle = statusUntracked
	default:
		statusStyle = fileStyle
	}

	staged := " "
	if f.Staged {
		staged = stagedIndicator.Render("S")
	}
	status := staged + statusStyle.Render(f.Status.String())

	// Expand indicator
	expandIndicator := " "
	if f.Expanded {
		expandIndicator = "▾"
	} else {
		expandIndicator = "▸"
	}

	// Split path into dir + filename for dimmed directory
	dir, file := splitPath(f.Path)
	var pathStr string
	if dir != "" {
		if isSelected {
			pathStr = selectedStyle.Render(dir) + selectedStyle.Render(file)
		} else {
			pathStr = fileDirStyle.Render(dir) + fileStyle.Render(file)
		}
	} else {
		if isSelected {
			pathStr = selectedStyle.Render(file)
		} else {
			pathStr = fileStyle.Render(file)
		}
	}

	// Change stats if loaded
	stats := ""
	if f.DiffLoaded && (f.Additions > 0 || f.Deletions > 0) {
		stats = " " + diffAdded.Render(fmt.Sprintf("+%d", f.Additions)) +
			diffRemoved.Render(fmt.Sprintf(" -%d", f.Deletions))
	}

	return fmt.Sprintf("%s%s %s %s%s", cursor, status, expandIndicator, pathStr, stats)
}

func splitPath(path string) (string, string) {
	return filepath.Split(path)
}

func copyToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}

func (m Model) renderDiffLine(dl types.DiffLine) string {
	content := dl.Content
	maxWidth := m.width - len(diffIndent) - 1
	if maxWidth > 0 && len(content) > maxWidth {
		content = content[:maxWidth]
	}

	switch dl.Type {
	case types.LineAdded:
		return diffIndent + diffAdded.Render(content)
	case types.LineRemoved:
		return diffIndent + diffRemoved.Render(content)
	case types.Header:
		return diffIndent + diffHeader.Render(content)
	default:
		return diffIndent + diffContext.Render(content)
	}
}

func (m Model) visibleFiles() []types.FileChange {
	if m.filter == types.FilterAll {
		return m.files
	}
	var result []types.FileChange
	for _, f := range m.files {
		if m.filter == types.FilterStaged && f.Staged {
			result = append(result, f)
		} else if m.filter == types.FilterUnstaged && f.Unstaged {
			result = append(result, f)
		}
	}
	return result
}

func (m *Model) cursorLineNum() int {
	visible := m.visibleFiles()
	lineNum := 0
	for idx, f := range visible {
		if idx == m.cursor {
			break
		}
		lineNum++ // file line
		if f.Expanded && f.DiffLoaded {
			lineNum += len(f.DiffLines)
			if idx < len(visible)-1 {
				lineNum++ // blank separator
			}
		}
	}
	return lineNum
}

func (m *Model) ensureVisibleSmooth() tea.Cmd {
	lineNum := m.cursorLineNum()
	avail := m.availHeight()

	newTarget := m.targetOffset
	if lineNum < newTarget {
		newTarget = lineNum
	} else if lineNum >= newTarget+avail {
		newTarget = lineNum - avail + 1
	}

	if newTarget != m.targetOffset {
		m.targetOffset = newTarget
		if !m.scrolling {
			m.scrolling = true
			return scrollTick()
		}
	}
	return nil
}

func (m Model) loadChanges() tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetChanges(m.repoPath)
		if err != nil {
			return []types.FileChange{}
		}
		return files
	}
}

func (m Model) loadDiff(path string, staged bool) tea.Cmd {
	repoPath := m.repoPath
	return func() tea.Msg {
		lines, err := git.GetFileDiff(repoPath, path, staged)
		if err != nil {
			return diffLoadedMsg{path: path, lines: nil, diffErr: err.Error()}
		}
		return diffLoadedMsg{path: path, lines: lines}
	}
}

func (m Model) startWatcher() tea.Cmd {
	return func() tea.Msg {
		w, err := git.NewWatcher(m.repoPath)
		if err != nil {
			return nil
		}
		// Store watcher - we need a way to pass it back
		// Use a special message for this
		return watcherStartedMsg{watcher: w}
	}
}

type watcherStartedMsg struct {
	watcher *git.Watcher
}

func (m Model) waitForChange() tea.Cmd {
	if m.watcher == nil {
		return nil
	}
	w := m.watcher
	return func() tea.Msg {
		<-w.Events
		return refreshMsg{}
	}
}

func (m Model) checkForUpdate() tea.Cmd {
	if m.updateChan == nil {
		return nil
	}
	ch := m.updateChan
	return func() tea.Msg {
		v, ok := <-ch
		if !ok || v == "" {
			return nil
		}
		return updateAvailableMsg{version: v}
	}
}

func (m Model) availHeight() int {
	// header + footer stats + version line
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) totalLines() int {
	total := 0
	visible := m.visibleFiles()
	for idx, f := range visible {
		total++ // file line
		if f.Expanded && f.DiffLoaded {
			total += len(f.DiffLines)
			if idx < len(visible)-1 {
				total++ // blank separator
			}
		}
	}
	if total == 0 {
		total = 1
	}
	return total
}

func (m *Model) clampOffset() {
	if m.offset < 0 {
		m.offset = 0
	}
	maxOffset := m.totalLines() - m.availHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m *Model) smoothScroll(delta int) tea.Cmd {
	m.targetOffset += delta
	// Clamp target
	maxOffset := m.totalLines() - m.availHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.targetOffset < 0 {
		m.targetOffset = 0
	}
	if m.targetOffset > maxOffset {
		m.targetOffset = maxOffset
	}
	if !m.scrolling {
		m.scrolling = true
		return scrollTick()
	}
	return nil
}

func scrollTick() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return scrollTickMsg{}
	})
}

func countChanges(lines []types.DiffLine) (int, int) {
	add, del := 0, 0
	for _, l := range lines {
		switch l.Type {
		case types.LineAdded:
			add++
		case types.LineRemoved:
			del++
		}
	}
	return add, del
}
