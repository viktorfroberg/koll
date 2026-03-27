package ui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#88C0D0")) // nord blue

	headerDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Faint(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")) // bright white

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	fileDirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // dim the directory part

	statusModified = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B")) // nord yellow

	statusAdded = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")) // nord green

	statusDeleted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF616A")) // nord red

	statusRenamed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B48EAD")) // nord purple

	statusUntracked = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	stagedIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")) // nord green

	diffAdded = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")) // nord green

	diffRemoved = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF616A")) // nord red

	diffHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5E81AC")). // nord dark blue
			Faint(true)

	diffContext = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	updateNotice = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B")) // nord yellow

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Faint(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88C0D0")).
			Bold(true)

	helpDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	flashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")) // green flash for "copied"

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF616A")).
			Faint(true)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Faint(true)

	diffIndent = "    "
)
