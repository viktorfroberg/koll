package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Toggle     key.Binding
	All        key.Binding
	Collapse   key.Binding
	Filter     key.Binding
	Refresh    key.Binding
	Quit       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	HalfUp     key.Binding
	HalfDown   key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Yank       key.Binding
	Help       key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k"),
	),
	Down: key.NewBinding(
		key.WithKeys("j"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", "l", "right"),
	),
	All: key.NewBinding(
		key.WithKeys("a"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("c"),
	),
	Filter: key.NewBinding(
		key.WithKeys("s"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
	),
	HalfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
	),
	Yank: key.NewBinding(
		key.WithKeys("y"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
	),
}
