package session

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines keyboard shortcuts for the session cycling TUI.
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Restart key.Binding
	Stop    key.Binding
	Start   key.Binding
	Force   key.Binding
}

// ShortHelp returns a short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns a full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Restart, k.Stop, k.Start},
		{k.Force, k.Help, k.Quit},
	}
}

// DefaultKeyMap returns the default keyboard shortcuts.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart session"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop session"),
		),
		Start: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "start session"),
		),
		Force: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "force stop"),
		),
	}
}
