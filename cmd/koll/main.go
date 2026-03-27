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

func printHelp() {
	fmt.Print(`koll - real-time git diff viewer

Usage:
  koll [path]           watch a repo (default: current directory)
  koll --split [path]   open in a split pane next to your editor
  koll help             show this help
  koll version          print version
  koll update           update to latest release

Flags:
  --split     open in a split pane (auto-detects terminal)
  --version   print version and exit
  --update    update koll to the latest version

Keybindings:
  j/k          jump between files
  ↑/↓          scroll line by line
  enter/l      toggle file diff
  a            expand all
  c            collapse all
  s            cycle filter: all → unstaged → staged
  y            copy file path to clipboard
  ?            show all keybindings
  q            quit
`)
}

func main() {
	split := flag.Bool("split", false, "Open in a split pane (auto-detects terminal multiplexer)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	update := flag.Bool("update", false, "Update koll to the latest version")
	flag.Usage = printHelp
	flag.Parse()

	// Handle "koll help", "koll version", "koll update" as subcommands
	if arg := flag.Arg(0); arg == "help" || arg == "-h" {
		printHelp()
		return
	} else if arg == "version" {
		*showVersion = true
	} else if arg == "update" {
		*update = true
	}

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

	// Find git repo (walks up from path)
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not a git repository: %s\n\n", path)
		fmt.Fprintf(os.Stderr, "  koll <path>       watch a git repo\n")
		fmt.Fprintf(os.Stderr, "  koll --split      open in a split pane\n")
		fmt.Fprintf(os.Stderr, "  koll help         show all commands\n")
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
