package ui

// The bubbles key package is imported as bkey: the tests already name a helper
// key, and a package-level name cannot share a file-scope import name.
import bkey "charm.land/bubbles/v2/key"

// keyMap is the one place that names the keys the list handles (D-17). Update
// matches key presses against it; the footer hints and the help overlay render
// from it.
type keyMap struct {
	Up, Down, Move, Refresh, Quit bkey.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:      bkey.NewBinding(bkey.WithKeys("k", "up"), bkey.WithHelp("k/↑", "up")),
		Down:    bkey.NewBinding(bkey.WithKeys("j", "down"), bkey.WithHelp("j/↓", "down")),
		Move:    bkey.NewBinding(bkey.WithKeys("j", "k"), bkey.WithHelp("j/k", "move")), // help text only; nothing matches on it
		Refresh: bkey.NewBinding(bkey.WithKeys("r"), bkey.WithHelp("r", "refresh")),
		Quit:    bkey.NewBinding(bkey.WithKeys("q", "ctrl+c"), bkey.WithHelp("q", "quit")),
	}
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []bkey.Binding {
	return []bkey.Binding{k.Move, k.Refresh, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]bkey.Binding {
	return [][]bkey.Binding{{k.Up, k.Down, k.Refresh}, {k.Quit}}
}
