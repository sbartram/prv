package ui

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// selPR builds the PR a/b#n.
func selPR(n int) domain.PR {
	return domain.PR{
		ID: fmt.Sprintf("PR_%d", n), Ref: domain.Ref{Repo: "a/b", Number: n},
		Title: fmt.Sprintf("change %d", n), Author: "octo",
	}
}

func selSnap(nums ...int) domain.Snapshot {
	s := domain.Snapshot{Viewer: "octo"}
	for _, n := range nums {
		s.PRs = append(s.PRs, selPR(n))
	}
	return s
}

// selModel builds a sized model that has loaded the PRs a/b#nums.
func selModel(w, h int, nums ...int) tea.Model {
	var m tea.Model = New(context.Background(), &fakeFetcher{}, Options{ClosedDays: 14})
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(result(m, selSnap(nums...), nil))
	return m
}

// refreshTo delivers a successful refresh result.
func refreshTo(m tea.Model, nums ...int) tea.Model {
	m, _ = m.Update(result(m, selSnap(nums...), nil))
	return m
}

func press(m tea.Model, keys ...string) tea.Model {
	for _, k := range keys {
		m, _ = m.Update(key(k))
	}
	return m
}

// markedRef is the ref on the row that starts with ">", or "" when none does.
func markedRef(t *testing.T, m tea.Model) string {
	t.Helper()
	var found []string
	for _, l := range strings.Split(plain(m), "\n") {
		if strings.HasPrefix(l, ">") {
			f := strings.Fields(l)
			if len(f) < 2 {
				t.Fatalf("marked line has no ref: %q", l)
			}
			found = append(found, f[1])
		}
	}
	if len(found) > 1 {
		t.Fatalf("%d rows are marked: %v", len(found), found)
	}
	if len(found) == 0 {
		return ""
	}
	return found[0]
}

func TestSelectionFollowsRef(t *testing.T) {
	m := selModel(120, 40, 1, 2, 3, 4)
	m = press(m, "j", "j")
	if got := markedRef(t, m); got != "a/b#3" {
		t.Fatalf("setup: marked = %q, want a/b#3", got)
	}
	m = refreshTo(m, 3, 1, 2, 4)
	if got := markedRef(t, m); got != "a/b#3" || m.(Model).cursor != 0 {
		t.Errorf("after C moved to the top: marked %q at cursor %d, want a/b#3 at 0", got, m.(Model).cursor)
	}
	m = refreshTo(m, 1, 3, 2, 4)
	if got := markedRef(t, m); got != "a/b#3" || m.(Model).cursor != 1 {
		t.Errorf("after C moved to row 2: marked %q at cursor %d, want a/b#3 at 1", got, m.(Model).cursor)
	}
	// The selected PR and its neighbour swap places.
	m = refreshTo(m, 3, 1, 2, 4)
	if got := markedRef(t, m); got != "a/b#3" {
		t.Errorf("after a swap with its neighbour: marked %q, want a/b#3", got)
	}
}

func TestSelectionNearestNeighbour(t *testing.T) {
	tests := []struct {
		name   string
		moves  []string
		next   []int
		marked string
	}{
		{"middle leaves: the one below", []string{"j", "j"}, []int{1, 2, 4, 5}, "a/b#4"},
		{"last leaves: the one above", []string{"j", "j", "j", "j"}, []int{1, 2, 3, 4}, "a/b#4"},
		{"first leaves: the one below", nil, []int{2, 3, 4, 5}, "a/b#2"},
		{"below also gone: the nearer one above", []string{"j", "j"}, []int{1, 2, 5}, "a/b#2"},
		{"equal distance: below wins", []string{"j", "j"}, []int{1, 5}, "a/b#5"},
		{"nothing survives: same row index, clamped", []string{"j", "j"}, []int{8, 9}, "a/b#9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := selModel(120, 40, 1, 2, 3, 4, 5)
			m = press(m, tt.moves...)
			m = refreshTo(m, tt.next...)
			if got := markedRef(t, m); got != tt.marked {
				t.Errorf("marked = %q, want %q", got, tt.marked)
			}
		})
	}
}

func TestSelectionSurvivesEmpty(t *testing.T) {
	m := selModel(120, 40, 1, 2, 3)
	m = press(m, "j")
	m = refreshTo(m)
	if got := m.(Model).cursor; got != 0 {
		t.Errorf("cursor on the empty screen = %d, want 0", got)
	}
	if !strings.Contains(plain(m), "No open PRs") {
		t.Fatalf("setup: no empty screen:\n%s", plain(m))
	}
	m = refreshTo(m, 1, 2, 3)
	if got := markedRef(t, m); got != "a/b#2" {
		t.Errorf("marked after the PRs came back = %q, want a/b#2", got)
	}
}

func TestSelectionFirstLoadAndSingleRow(t *testing.T) {
	m := selModel(120, 40, 1, 2)
	if got := markedRef(t, m); got != "a/b#1" {
		t.Errorf("first load: marked = %q, want a/b#1", got)
	}
	one := selModel(120, 40, 1)
	one = refreshTo(one, 1)
	if got := markedRef(t, one); got != "a/b#1" {
		t.Errorf("one-row list refreshed with itself: marked = %q, want a/b#1", got)
	}
}

func TestSelectionAgedOut(t *testing.T) {
	snap := selSnap(1, 2, 3)
	snap.PRs[1].State = domain.StateMerged
	var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{ClosedDays: 14})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(result(m, snap, nil))
	m = press(m, "j")
	if got := markedRef(t, m); got != "a/b#2" {
		t.Fatalf("setup: marked = %q, want the merged a/b#2", got)
	}
	// The merged PR ages out of the closed window.
	m = refreshTo(m, 1, 3)
	if got := markedRef(t, m); got != "a/b#3" {
		t.Errorf("marked after the merged PR left = %q, want its neighbour a/b#3", got)
	}
}

func TestSelectionScrollsIntoView(t *testing.T) {
	nums := make([]int, 30)
	for i := range nums {
		nums[i] = i + 1
	}
	m := selModel(120, 24, nums...)
	for i := 0; i < 24; i++ {
		m = press(m, "j")
	}
	if got := markedRef(t, m); got != "a/b#25" {
		t.Fatalf("setup: marked = %q, want a/b#25", got)
	}
	moved := []int{1, 25}
	for _, n := range nums {
		if n != 1 && n != 25 {
			moved = append(moved, n)
		}
	}
	m = refreshTo(m, moved...)
	if got := markedRef(t, m); got != "a/b#25" {
		t.Errorf("after the move to row 2: marked = %q, want a/b#25 on screen", got)
	}
	ls := strings.Split(plain(m), "\n")
	if len(ls) != 24 {
		t.Errorf("view has %d lines, want 24", len(ls))
	}
	for i, l := range ls {
		if strings.TrimSpace(l) == "" {
			t.Errorf("line %d is blank:\n%s", i, plain(m))
		}
	}
}

func TestSelectionInvariant(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	pool := make([]int, 12)
	for i := range pool {
		pool[i] = i + 1
	}
	m := selModel(100, 10, pool...)
	for step := 0; step < 200; step++ {
		switch rng.IntN(3) {
		case 0:
			sub := append([]int(nil), pool...)
			rng.Shuffle(len(sub), func(i, j int) { sub[i], sub[j] = sub[j], sub[i] })
			m = refreshTo(m, sub[:rng.IntN(len(sub)+1)]...)
		case 1:
			m = press(m, "j")
		default:
			m = press(m, "k")
		}
		mm := m.(Model)
		if len(mm.rows) == 0 {
			if mm.cursor != 0 || mm.offset != 0 {
				t.Fatalf("step %d: no rows but cursor %d offset %d", step, mm.cursor, mm.offset)
			}
			continue
		}
		if mm.cursor < 0 || mm.cursor >= len(mm.rows) {
			t.Fatalf("step %d: cursor %d out of range for %d rows", step, mm.cursor, len(mm.rows))
		}
		if i := indexOf(mm.rows, mm.selected); i >= 0 && mm.rows[mm.cursor].Ref != mm.selected {
			t.Fatalf("step %d: selected %v is at row %d but the cursor is on %v", step, mm.selected, i, mm.rows[mm.cursor].Ref)
		}
		if mm.cursor < mm.offset || mm.cursor >= mm.offset+mm.visibleRows() {
			t.Fatalf("step %d: cursor %d outside the window [%d, %d)", step, mm.cursor, mm.offset, mm.offset+mm.visibleRows())
		}
		if got := markedRef(t, m); got != mm.rows[mm.cursor].Ref.String() {
			t.Fatalf("step %d: marked %q, cursor row %v", step, got, mm.rows[mm.cursor].Ref)
		}
	}
}
