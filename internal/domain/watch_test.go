package domain

import (
	"slices"
	"testing"
)

func TestMarkWatched(t *testing.T) {
	prs := []PR{{ID: "A", Ref: Ref{"acme/api", 1}}, {ID: "B", Ref: Ref{"acme/web", 2}}}
	saved := []Ref{{"ACME/API", 1}, {"octo/tools", 42}, {"octo/tools", 42}}

	rest := MarkWatched(prs, saved)

	if want := []Ref{{"octo/tools", 42}}; !slices.Equal(rest, want) {
		t.Errorf("rest = %v, want %v", rest, want)
	}
	if !prs[0].Manual || prs[0].Watched != (Ref{"ACME/API", 1}) {
		t.Errorf("acme/api#1 = Manual %v, Watched %v, want Manual with Watched ACME/API#1", prs[0].Manual, prs[0].Watched)
	}
	if prs[1].Manual || prs[1].Watched != (Ref{}) {
		t.Errorf("acme/web#2 was marked: %+v", prs[1])
	}
}

func TestMarkWatchedNothingSaved(t *testing.T) {
	prs := []PR{{ID: "A", Ref: Ref{"acme/api", 1}}}
	if rest := MarkWatched(prs, nil); len(rest) != 0 || prs[0].Manual {
		t.Errorf("rest = %v, Manual = %v, want nothing", rest, prs[0].Manual)
	}
}

func TestCombine(t *testing.T) {
	auto := []PR{{ID: "A", Ref: Ref{"acme/new", 9}, Auto: true}}
	looked := []PR{
		{ID: "A", Ref: Ref{"acme/old", 9}, Manual: true, Watched: Ref{"acme/old", 9}},
		{Ref: Ref{"acme/gone", 5}, Manual: true, Watched: Ref{"acme/gone", 5}, Unavail: NotFound()},
		{Ref: Ref{"acme/gone", 6}, Manual: true, Watched: Ref{"acme/gone", 6}, Unavail: NotFound()},
	}

	got := Combine(auto, looked)

	if len(got) != 3 {
		t.Fatalf("Combine = %d rows, want 3: %+v", len(got), got)
	}
	if r := got[0]; r.Ref != (Ref{"acme/new", 9}) || !r.Auto || !r.Manual || r.Watched != (Ref{"acme/old", 9}) {
		t.Errorf("row 0 = %+v, want acme/new#9 Auto+Manual with Watched acme/old#9", r)
	}
	if r := got[1]; r.Ref != (Ref{"acme/gone", 5}) || r.Unavail.Kind != UnavailNotFound {
		t.Errorf("row 1 = %+v, want unavailable acme/gone#5", r)
	}
	if r := got[2]; r.Ref != (Ref{"acme/gone", 6}) {
		t.Errorf("row 2 = %+v, want acme/gone#6 (an empty ID is never merged by ID)", r)
	}
	if auto[0].Manual || len(auto) != 1 || len(looked) != 3 {
		t.Errorf("Combine changed an input: %+v", auto)
	}
}

// TestEverySavedRefOnce is the invariant of the watch list: every saved ref that
// resolves to a distinct PR is in the rows exactly once, as a loaded row or an
// unavailable row, and no node ID is listed twice.
func TestEverySavedRefOnce(t *testing.T) {
	loaded := func(id string, ref Ref) PR { return PR{ID: id, Ref: ref, Manual: true, Watched: ref} }
	cases := []struct {
		name    string
		auto    []PR
		saved   []Ref
		resolve map[Ref]PR // by Key of the saved ref; a missing key is not found
	}{
		{"nothing saved", []PR{{ID: "A", Ref: Ref{"acme/api", 1}, Auto: true}}, nil, nil},
		{"saved matches discovery in another letter case",
			[]PR{{ID: "A", Ref: Ref{"acme/api", 1}, Auto: true}},
			[]Ref{{"ACME/Api", 1}}, nil},
		{"saved only",
			nil,
			[]Ref{{"octo/tools", 42}},
			map[Ref]PR{{"octo/tools", 42}: loaded("B", Ref{"octo/tools", 42})}},
		{"renamed repo folds into its discovery row",
			[]PR{{ID: "A", Ref: Ref{"acme/new", 9}, Auto: true}},
			[]Ref{{"acme/old", 9}},
			map[Ref]PR{{"acme/old", 9}: loaded("A", Ref{"acme/new", 9})}},
		{"not found",
			[]PR{{ID: "A", Ref: Ref{"acme/api", 1}, Auto: true}},
			[]Ref{{"acme/gone", 5}, {"acme/gone", 6}}, nil},
		{"all together, with repeats",
			[]PR{{ID: "A", Ref: Ref{"acme/new", 9}, Auto: true}, {ID: "C", Ref: Ref{"acme/api", 1}, Auto: true}},
			[]Ref{{"acme/old", 9}, {"ACME/API", 1}, {"acme/gone", 5}, {"octo/tools", 42}, {"Octo/Tools", 42}, {"acme/old", 9}},
			map[Ref]PR{
				{"acme/old", 9}:    loaded("A", Ref{"acme/new", 9}),
				{"octo/tools", 42}: loaded("B", Ref{"octo/tools", 42}),
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			auto := slices.Clone(c.auto)
			rest := MarkWatched(auto, c.saved)
			var looked []PR
			for _, r := range rest {
				if pr, ok := c.resolve[r.Key()]; ok {
					pr.Watched = r
					looked = append(looked, pr)
					continue
				}
				looked = append(looked, PR{Ref: r, Manual: true, Watched: r, Unavail: NotFound()})
			}
			rows := Combine(auto, looked)

			ids := map[string]int{}
			watched := map[Ref]int{}
			for _, r := range rows {
				if r.ID != "" {
					ids[r.ID]++
				}
				if r.Manual {
					watched[r.Watched.Key()]++
				}
			}
			for id, n := range ids {
				if n != 1 {
					t.Errorf("node ID %s is listed %d times", id, n)
				}
			}
			want := map[Ref]bool{}
			for _, r := range c.saved {
				want[r.Key()] = true
			}
			for k := range want {
				if watched[k] != 1 {
					t.Errorf("saved %v is in %d rows, want exactly 1", k, watched[k])
				}
			}
			for k := range watched {
				if !want[k] {
					t.Errorf("row for %v is marked saved but was not saved", k)
				}
			}
		})
	}
}
