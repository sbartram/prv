package domain

import (
	"maps"
	"reflect"
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

func TestReconcile(t *testing.T) {
	ref := Ref{"octo/tools", 42}
	key := ref.Key()
	loaded := func() PR {
		return PR{ID: "PR_42", Ref: ref, Title: "Speed up the build", State: StateOpen, CI: CIPassing,
			Manual: true, Watched: Ref{"Octo/Tools", 42}}
	}
	failed := func() PR { return PR{Ref: ref, Manual: true, Watched: ref, Unavail: LoadFailed("server error", "")} }
	notFound := func() PR { return PR{Ref: ref, Manual: true, Watched: ref, Unavail: NotFound()} }
	sso := func() PR { return PR{Ref: ref, Manual: true, Watched: ref, Unavail: SSORequired("octo")} }

	tests := []struct {
		name        string
		prev, next  []PR
		strikes     map[Ref]int
		wantKind    UnavailKind
		wantTitle   string
		wantStrikes map[Ref]int
	}{
		{"first launch, other error: unavailable at once", nil, []PR{failed()}, nil, UnavailOther, "", map[Ref]int{key: 1}},
		{"first strike keeps the last data", []PR{loaded()}, []PR{failed()}, nil, UnavailNone, "Speed up the build", map[Ref]int{key: 1}},
		{"second strike in a row is unavailable", []PR{loaded()}, []PR{failed()}, map[Ref]int{key: 1}, UnavailOther, "", map[Ref]int{key: 2}},
		{"third strike stays unavailable", []PR{failed()}, []PR{failed()}, map[Ref]int{key: 2}, UnavailOther, "", map[Ref]int{key: 3}},
		{"a PR that loads clears the count", []PR{loaded()}, []PR{loaded()}, map[Ref]int{key: 1}, UnavailNone, "Speed up the build", map[Ref]int{}},
		{"not found is unavailable at once", []PR{loaded()}, []PR{notFound()}, nil, UnavailNotFound, "", map[Ref]int{}},
		{"SSO is unavailable at once", []PR{loaded()}, []PR{sso()}, map[Ref]int{key: 1}, UnavailSSO, "", map[Ref]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev, next := slices.Clone(tt.prev), slices.Clone(tt.next)
			var strikes map[Ref]int
			if tt.strikes != nil {
				strikes = map[Ref]int{}
				for k, v := range tt.strikes {
					strikes[k] = v
				}
			}
			out, got := Reconcile(tt.prev, tt.next, tt.strikes)
			if len(out) != 1 {
				t.Fatalf("rows = %d, want 1", len(out))
			}
			if out[0].Unavail.Kind != tt.wantKind || out[0].Title != tt.wantTitle {
				t.Errorf("row = kind %v, title %q; want kind %v, title %q", out[0].Unavail.Kind, out[0].Title, tt.wantKind, tt.wantTitle)
			}
			if !maps.Equal(got, tt.wantStrikes) {
				t.Errorf("strikes = %v, want %v", got, tt.wantStrikes)
			}
			if !reflect.DeepEqual(tt.prev, prev) || !reflect.DeepEqual(tt.next, next) {
				t.Error("Reconcile changed an input slice")
			}
			if tt.strikes != nil && !maps.Equal(tt.strikes, strikes) {
				t.Errorf("Reconcile changed the input map: %v, was %v", tt.strikes, strikes)
			}
		})
	}

	t.Run("kept row stays saved and is not auto", func(t *testing.T) {
		out, _ := Reconcile([]PR{loaded()}, []PR{failed()}, nil)
		if !out[0].Manual || out[0].Auto || out[0].Watched != ref {
			t.Errorf("kept row = Manual %v, Auto %v, Watched %v; want Manual, not Auto, Watched %v",
				out[0].Manual, out[0].Auto, out[0].Watched, ref)
		}
	})
	t.Run("a prev row that was not saved is not reused", func(t *testing.T) {
		prev := loaded()
		prev.Manual = false
		out, _ := Reconcile([]PR{prev}, []PR{failed()}, nil)
		if out[0].Unavail.Kind != UnavailOther {
			t.Errorf("kind = %v, want unavailable", out[0].Unavail.Kind)
		}
	})
	t.Run("a row replaced by its last data is sorted into place", func(t *testing.T) {
		fail := PR{ID: "F", Ref: Ref{"acme/x", 1}, State: StateOpen, CI: CIFailing, Auto: true}
		out, _ := Reconcile([]PR{loaded()}, []PR{failed(), fail}, nil)
		if len(out) != 2 || out[0].ID != "F" || out[1].ID != "PR_42" {
			t.Errorf("order = %v, want the failing PR first (bucket 1), the kept PR second (bucket 5)", out)
		}
	})
	t.Run("rows that were not replaced keep their order", func(t *testing.T) {
		a := PR{ID: "A", Ref: Ref{"acme/a", 1}, State: StateClosed}
		b := PR{ID: "B", Ref: Ref{"acme/b", 1}, State: StateOpen, CI: CIFailing}
		out, _ := Reconcile(nil, []PR{a, b}, nil)
		if out[0].ID != "A" || out[1].ID != "B" {
			t.Errorf("order = %v, want next's order", out)
		}
	})
}
