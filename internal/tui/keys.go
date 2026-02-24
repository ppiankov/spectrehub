package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit        key.Binding
	Search      key.Binding
	FilterTool  key.Binding
	Sort        key.Binding
	Copy        key.Binding
	ClearFilter key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	FilterTool: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "filter tool"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "cycle sort"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear"),
	),
}
