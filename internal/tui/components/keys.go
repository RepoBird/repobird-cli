package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Home        key.Binding
	End         key.Binding
	Enter       key.Binding
	Back        key.Binding
	Quit        key.Binding
	Help        key.Binding
	Refresh     key.Binding
	New         key.Binding
	Delete      key.Binding
	Search      key.Binding
	NextMatch   key.Binding
	PrevMatch   key.Binding
	CommandMode key.Binding
	VisualMode  key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "go back"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "go forward"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+u", "pgup"),
		key.WithHelp("ctrl+u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "pgdown"),
		key.WithHelp("ctrl+d", "page down"),
	),
	Home: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "go to top"),
	),
	End: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "go to bottom"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/view"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new run"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete/cancel"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	NextMatch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	),
	PrevMatch: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "previous match"),
	),
	CommandMode: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "command mode"),
	),
	VisualMode: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "visual mode"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Enter, k.Back, k.Refresh, k.New},
		{k.Delete, k.Search, k.CommandMode, k.Help},
		{k.Quit},
	}
}

type InputMode int

const (
	NormalMode InputMode = iota
	InsertMode
	CommandMode
	SearchMode
	VisualMode
)

func HandleVimNavigation(msg tea.KeyMsg, currentIndex, maxIndex int) (int, bool) {
	switch msg.String() {
	case "j", "down":
		if currentIndex < maxIndex-1 {
			return currentIndex + 1, true
		}
	case "k", "up":
		if currentIndex > 0 {
			return currentIndex - 1, true
		}
	case "g", "home":
		return 0, true
	case "G", "end":
		return maxIndex - 1, true
	case "ctrl+d":
		newIndex := currentIndex + 10
		if newIndex >= maxIndex {
			newIndex = maxIndex - 1
		}
		return newIndex, true
	case "ctrl+u":
		newIndex := currentIndex - 10
		if newIndex < 0 {
			newIndex = 0
		}
		return newIndex, true
	}
	return currentIndex, false
}
