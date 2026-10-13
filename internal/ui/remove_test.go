package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// removeSnap holds a discovered PR, the saved PR octo/tools#42 and a second
// discovered PR, sorted as Fetch sorts them.
func removeSnap() domain.Snapshot {
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "Api change", Author: "octo", Auto: true,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIFailing, UpdatedAt: addUpdated},
		{ID: "PR_42", Ref: toolsRef, Title: "Tools change", Author: "sam", Manual: true, Watched: toolsRef,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: addUpdated},
		{ID: "PR_3", Ref: domain.Ref{Repo: "zeta/web", Number: 3}, Title: "Web change", Author: "octo", Auto: true,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: addUpdated},
	}}
	domain.Sort(snap.PRs)
	return snap
}

// selectRef moves the cursor down with j until ref is selected.
func selectRef(t *testing.T, m tea.Model, ref domain.Ref) tea.Model {
	t.Helper()
	for range 10 {
		if selectedRef(m) == ref {
			return m
		}
		m = press(m, "j")
	}
	t.Fatalf("could not select %v; selected %v", ref, selectedRef(m))
	return m
}

// rowBelow returns the ref of the row under ref in the model's rows.
func rowBelow(t *testing.T, m tea.Model, ref domain.Ref) domain.Ref {
	t.Helper()
	rows := m.(Model).rows
	i := indexOf(rows, ref)
	if i < 0 || i+1 >= len(rows) {
		t.Fatalf("no row below %v in %v", ref, rows)
	}
	return rows[i+1].Ref
}

func TestRemoveSaved(t *testing.T) {
	w := &memWatch{removed: true}
	m := addModel(t, &addFetcher{}, w, removeSnap())
	m = selectRef(t, m, toolsRef)
	below := rowBelow(t, m, toolsRef)

	m, cmd := m.Update(key("d"))
	if cmd != nil {
		t.Errorf("d returned a Cmd; asking sends nothing")
	}
	if got, want := lastLine(m), "remove octo/tools#42 from watch list? y/n"; got != want {
		t.Errorf("last line = %q, want %q", got, want)
	}
	if len(w.removes) != 0 {
		t.Fatalf("Remove called before y: %v", w.removes)
	}

	m, cmd = m.Update(key("y"))
	if cmd == nil {
		t.Fatal("y returned no Cmd")
	}
	m = drainUI(m, cmd)

	if len(w.removes) != 1 || w.removes[0] != toolsRef {
		t.Errorf("Remove calls = %v, want [%v]", w.removes, toolsRef)
	}
	if rowsHave(m, toolsRef) || countRef(m, toolsRef) != 0 {
		t.Errorf("octo/tools#42 is still listed:\n%s", plain(m))
	}
	if got, want := flashOf(m), "removed octo/tools#42"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if !strings.Contains(lastLine(m), "removed octo/tools#42") {
		t.Errorf("last line = %q, want it to hold 'removed octo/tools#42'", lastLine(m))
	}
	if got := selectedRef(m); got != below {
		t.Errorf("selected = %v, want the row below, %v", got, below)
	}
}
