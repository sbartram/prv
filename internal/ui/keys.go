package ui

// The bubbles key package is imported as bkey: the tests already name a helper
// key, and a package-level name cannot share a file-scope import name.
import bkey "charm.land/bubbles/v2/key"

// keyMap is the one place that names the keys the list handles (D-17). Update
// matches key presses against it; the footer hints and the help overlay render
// from it.
type keyMap struct {
	Up, Down, Move, Refresh, Quit bkey.Binding

	// Help opens the full-screen help overlay (D-17); Open opens the selected
	// PR in the browser (NAV-02).
	Help, Open bkey.Binding

	// Enter opens the detail view of the selected PR (DETL-01); Back leaves it
	// (DETL-03). Scroll is help text only: the detail scrolls on Up and Down.
	Enter, Back, Scroll bkey.Binding

	// Expand lists the passing checks that the detail view folds (D-10).
	Expand bkey.Binding

	// State visibility (D-10). States is help text only; nothing matches on it.
	ToggleOpen, ToggleMerged, ToggleClosed, States bkey.Binding

	// PR-number filter (D-11). Filter opens the prompt; Clear drops a kept
	// number filter in list mode.
	Filter, Clear bkey.Binding

	// Add opens the add-PR prompt (D-01); the list and empty screens only. It is
	// not in the footer hints or the help yet.
	Add bkey.Binding

	// Remove asks to take the selected PR off the watch list (D-06); the list
	// screen only, and only rows marked +. It is not in the footer hints or the
	// help yet.
	Remove bkey.Binding
}

// promptKeys are the keys the number prompt handles while it is open. Every
// other key is ignored, apart from ctrl+c.
type promptKeys struct {
	Accept, Cancel, Delete bkey.Binding

	// Yes is the one key that accepts the remove confirm line.
	Yes bkey.Binding
}

func defaultPromptKeys() promptKeys {
	return promptKeys{
		Accept: bkey.NewBinding(bkey.WithKeys("enter")),
		Cancel: bkey.NewBinding(bkey.WithKeys("esc")),
		Delete: bkey.NewBinding(bkey.WithKeys("backspace")),
		Yes:    bkey.NewBinding(bkey.WithKeys("y")),
	}
}

func defaultKeys() keyMap {
	return keyMap{
		Up:      bkey.NewBinding(bkey.WithKeys("k", "up"), bkey.WithHelp("k/↑", "up")),
		Down:    bkey.NewBinding(bkey.WithKeys("j", "down"), bkey.WithHelp("j/↓", "down")),
		Move:    bkey.NewBinding(bkey.WithKeys("j", "k"), bkey.WithHelp("j/k", "move")), // help text only; nothing matches on it
		Refresh: bkey.NewBinding(bkey.WithKeys("r"), bkey.WithHelp("r", "refresh")),
		Quit:    bkey.NewBinding(bkey.WithKeys("q", "ctrl+c"), bkey.WithHelp("q", "quit")),
		Help:    bkey.NewBinding(bkey.WithKeys("?"), bkey.WithHelp("?", "help")),
		Open:    bkey.NewBinding(bkey.WithKeys("o"), bkey.WithHelp("o", "open")),

		Enter:  bkey.NewBinding(bkey.WithKeys("enter"), bkey.WithHelp("enter", "details")),
		Back:   bkey.NewBinding(bkey.WithKeys("esc"), bkey.WithHelp("esc", "back")),
		Scroll: bkey.NewBinding(bkey.WithKeys("j", "k"), bkey.WithHelp("j/k", "scroll")), // help text only; nothing matches on it

		Expand: bkey.NewBinding(bkey.WithKeys("e"), bkey.WithHelp("e", "expand")),

		ToggleOpen:   bkey.NewBinding(bkey.WithKeys("1"), bkey.WithHelp("1", "open")),
		ToggleMerged: bkey.NewBinding(bkey.WithKeys("2"), bkey.WithHelp("2", "merged")),
		ToggleClosed: bkey.NewBinding(bkey.WithKeys("3"), bkey.WithHelp("3", "closed")),
		States:       bkey.NewBinding(bkey.WithKeys("1", "2", "3"), bkey.WithHelp("1/2/3", "states")),

		Filter: bkey.NewBinding(bkey.WithKeys("/"), bkey.WithHelp("/", "PR #")),
		Clear:  bkey.NewBinding(bkey.WithKeys("esc"), bkey.WithHelp("esc", "clear #")),

		Add:    bkey.NewBinding(bkey.WithKeys("a"), bkey.WithHelp("a", "add")),
		Remove: bkey.NewBinding(bkey.WithKeys("d"), bkey.WithHelp("d", "remove")),
	}
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []bkey.Binding {
	return []bkey.Binding{k.Move, k.Refresh, k.Enter, k.Open, k.Filter, k.States, k.Help, k.Quit}
}

// DetailShortHelp is the footer hint list while the detail view is open (D-09).
func (k keyMap) DetailShortHelp() []bkey.Binding {
	return []bkey.Binding{k.Back, k.Open, k.Refresh, k.Scroll, k.Expand, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]bkey.Binding {
	return [][]bkey.Binding{{k.Up, k.Down, k.Refresh, k.Open}, {k.Filter, k.Clear, k.ToggleOpen, k.ToggleMerged, k.ToggleClosed}, {k.Enter, k.Back, k.Scroll, k.Expand}, {k.Help, k.Quit}}
}
