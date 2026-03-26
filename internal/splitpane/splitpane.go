package splitpane

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Multiplexer int

const (
	None Multiplexer = iota
	Cmux
	Tmux
	Zellij
	Wezterm
	Kitty
	Ghostty
	ITerm2
)

func (m Multiplexer) String() string {
	switch m {
	case Cmux:
		return "cmux"
	case Tmux:
		return "tmux"
	case Zellij:
		return "zellij"
	case Wezterm:
		return "wezterm"
	case Kitty:
		return "kitty"
	case Ghostty:
		return "ghostty"
	case ITerm2:
		return "iTerm2"
	default:
		return "none"
	}
}

// Detect returns the terminal multiplexer in use, checking
// session-based multiplexers first (they run inside terminal emulators).
func Detect() Multiplexer {
	if os.Getenv("CMUX_WORKSPACE_ID") != "" {
		return Cmux
	}
	if os.Getenv("TMUX") != "" {
		return Tmux
	}
	if os.Getenv("ZELLIJ") != "" {
		return Zellij
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return Wezterm
	}
	if os.Getenv("KITTY_PID") != "" {
		return Kitty
	}
	if os.Getenv("TERM_PROGRAM") == "ghostty" {
		return Ghostty
	}
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return ITerm2
	}
	return None
}

// Launch opens koll in a split pane using the detected multiplexer.
func Launch(path string) error {
	mux := Detect()

	bin, err := findBinary()
	if err != nil {
		return fmt.Errorf("cannot find koll binary: %w", err)
	}

	switch mux {
	case Cmux:
		return launchCmux(bin, path)
	case Tmux:
		return launchTmux(bin, path)
	case Zellij:
		return launchZellij(bin, path)
	case Wezterm:
		return launchWezterm(bin, path)
	case Kitty:
		return launchKitty(bin, path)
	case Ghostty:
		return launchGhostty(bin, path)
	case ITerm2:
		return launchITerm2(bin, path)
	default:
		return fmt.Errorf("no supported terminal multiplexer detected\n\nSupported: cmux, tmux, zellij, wezterm, kitty, ghostty, iTerm2\n\nRun 'koll' directly in a split pane instead")
	}
}

func findBinary() (string, error) {
	// Try os.Executable first
	exe, err := os.Executable()
	if err == nil && !strings.Contains(exe, "go-build") {
		return exe, nil
	}
	// Fall back to PATH lookup
	p, err := exec.LookPath("koll")
	if err == nil {
		return p, nil
	}
	return "", fmt.Errorf("koll not found in PATH; install it first")
}

func launchCmux(bin, path string) error {
	// new-split returns "OK surface:N workspace:N"
	out, err := exec.Command("cmux", "new-split", "right").Output()
	if err != nil {
		return fmt.Errorf("cmux new-split failed: %w", err)
	}

	// Parse surface ID from output like "OK surface:63 workspace:6"
	newSurface := ""
	for _, field := range strings.Fields(string(out)) {
		if strings.HasPrefix(field, "surface:") {
			newSurface = field
			break
		}
	}
	if newSurface == "" {
		return fmt.Errorf("cmux new-split did not return a surface ID: %s", string(out))
	}

	time.Sleep(300 * time.Millisecond)

	cmd := fmt.Sprintf("%s %s\n", bin, path)
	if err := exec.Command("cmux", "send", "--surface", newSurface, cmd).Run(); err != nil {
		return fmt.Errorf("cmux send to %s failed: %w", newSurface, err)
	}
	return nil
}

func launchTmux(bin, path string) error {
	return exec.Command("tmux", "split-window", "-h", bin, path).Run()
}

func launchZellij(bin, path string) error {
	return exec.Command("zellij", "run", "--direction", "right", "--", bin, path).Run()
}

func launchWezterm(bin, path string) error {
	return exec.Command("wezterm", "cli", "split-pane", "--right", "--", bin, path).Run()
}

func launchKitty(bin, path string) error {
	return exec.Command("kitty", "@", "launch", "--location=vsplit", bin, path).Run()
}

func launchGhostty(bin, path string) error {
	// Ghostty uses its CLI for split creation
	return exec.Command("ghostty", "+new-split", "--direction=right", "--command", fmt.Sprintf("%s %s", bin, path)).Run()
}

func launchITerm2(bin, path string) error {
	script := fmt.Sprintf(`
tell application "iTerm2"
	tell current session of current window
		set newSession to (split vertically with default profile)
		tell newSession
			write text "%s %s"
		end tell
	end tell
end tell`, bin, path)
	return exec.Command("osascript", "-e", script).Run()
}
