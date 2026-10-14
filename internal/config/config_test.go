package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

func TestPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	abs := filepath.Join(t.TempDir(), "x")
	cases := []struct {
		name, xdg, want string
	}{
		{"absolute", abs, filepath.Join(abs, "prv", "config.toml")},
		{"relative", "relative/dir", filepath.Join(home, ".config", "prv", "config.toml")},
		{"unset", "", filepath.Join(home, ".config", "prv", "config.toml")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", c.xdg)
			got, err := Path()
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("Path() = %q, want %q", got, c.want)
			}
		})
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadInterval(t *testing.T) {
	ok := []struct {
		name, body string
		want       time.Duration
	}{
		{"empty file", "", DefaultInterval},
		{"comment only", "# nothing\n", DefaultInterval},
		{"20s", `refresh_interval = "20s"`, 20 * time.Second},
		{"14s raised", `refresh_interval = "14s"`, 15 * time.Second},
		{"15s", `refresh_interval = "15s"`, 15 * time.Second},
		{"16s", `refresh_interval = "16s"`, 16 * time.Second},
		{"0s raised", `refresh_interval = "0s"`, 15 * time.Second},
		{"1.5s raised", `refresh_interval = "1.5s"`, 15 * time.Second},
		{"2m", `refresh_interval = "2m"`, 2 * time.Minute},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			p := write(t, c.body)
			cfg, err := Load(p)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Interval != c.want {
				t.Errorf("Interval = %v, want %v", cfg.Interval, c.want)
			}
			if cfg.Path != p {
				t.Errorf("Path = %q, want %q", cfg.Path, p)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		cfg, err := Load(filepath.Join(t.TempDir(), "nope", "config.toml"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Interval != DefaultInterval {
			t.Errorf("Interval = %v, want %v", cfg.Interval, DefaultInterval)
		}
	})

	bad := []struct{ name, body string }{
		{"integer", `refresh_interval = 60`},
		{"words", `refresh_interval = "soon"`},
		{"syntax", `refresh_interval = `},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			p := write(t, c.body)
			_, err := Load(p)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.HasPrefix(err.Error(), p) {
				t.Errorf("error %q does not start with the path", err)
			}
		})
	}

	t.Run("oversize", func(t *testing.T) {
		p := write(t, strings.Repeat("#", maxFileSize+1))
		_, err := Load(p)
		if err == nil || !strings.Contains(err.Error(), "larger than 1 MiB") {
			t.Fatalf("err = %v, want 'larger than 1 MiB'", err)
		}
	})
	t.Run("exactly 1 MiB is accepted", func(t *testing.T) {
		p := write(t, strings.Repeat("#", maxFileSize))
		if _, err := Load(p); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("directory", func(t *testing.T) {
		if _, err := Load(t.TempDir()); err == nil {
			t.Fatal("want an error for a directory")
		}
	})
}

func TestLoadAllKeys(t *testing.T) {
	t.Run("empty file gives defaults", func(t *testing.T) {
		cfg, err := Load(write(t, ""))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Interval != DefaultInterval || cfg.ClosedDays != 14 || cfg.Show != domain.AllStates || len(cfg.Warnings) != 0 {
			t.Errorf("got %+v, want all defaults and no warnings", cfg)
		}
	})
	t.Run("missing file gives defaults", func(t *testing.T) {
		cfg, err := Load(filepath.Join(t.TempDir(), "none.toml"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ClosedDays != 14 || cfg.Show != domain.AllStates || len(cfg.Warnings) != 0 {
			t.Errorf("got %+v, want defaults", cfg)
		}
	})
	t.Run("all three keys", func(t *testing.T) {
		cfg, err := Load(write(t, "refresh_interval = \"30s\"\nclosed_days = 30\nshow = [\"open\", \"merged\"]\n"))
		if err != nil {
			t.Fatal(err)
		}
		want := domain.StateSet(0).Toggle(domain.StateOpen).Toggle(domain.StateMerged)
		if cfg.Interval != 30*time.Second || cfg.ClosedDays != 30 || cfg.Show != want {
			t.Errorf("got %+v", cfg)
		}
	})
	t.Run("a missing key takes its default", func(t *testing.T) {
		cfg, err := Load(write(t, "closed_days = 7\n"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Interval != DefaultInterval || cfg.ClosedDays != 7 || cfg.Show != domain.AllStates {
			t.Errorf("got %+v", cfg)
		}
	})
}

func TestClampClosedDays(t *testing.T) {
	cases := []struct {
		body string
		want int
	}{
		{"closed_days = 30", 30},
		{"closed_days = 0", 1},
		{"closed_days = -3", 1},
		{"closed_days = 1", 1},
		{"closed_days = 365", 365},
		{"closed_days = 366", 365},
		{"closed_days = 99999999999", 365},
	}
	for _, c := range cases {
		t.Run(c.body, func(t *testing.T) {
			cfg, err := Load(write(t, c.body))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ClosedDays != c.want {
				t.Errorf("ClosedDays = %d, want %d", cfg.ClosedDays, c.want)
			}
		})
	}
}

func TestShowValues(t *testing.T) {
	only := func(sts ...domain.State) domain.StateSet {
		var s domain.StateSet
		for _, st := range sts {
			s |= domain.StateSet(1) << uint(st)
		}
		return s
	}
	cases := []struct {
		name, body string
		want       domain.StateSet
		warns      []string
	}{
		{"open only", `show = ["open"]`, only(domain.StateOpen), nil},
		{"duplicates", `show = ["open", "open"]`, only(domain.StateOpen), nil},
		{"order a", `show = ["closed", "open"]`, only(domain.StateOpen, domain.StateClosed), nil},
		{"order b", `show = ["open", "closed"]`, only(domain.StateOpen, domain.StateClosed), nil},
		{"all three", `show = ["merged", "closed", "open"]`, domain.AllStates, nil},
		{"empty", `show = []`, domain.AllStates, []string{"config: show is empty; showing all states"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := Load(write(t, c.body))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Show != c.want {
				t.Errorf("Show = %b, want %b", cfg.Show, c.want)
			}
			if !reflect.DeepEqual(cfg.Warnings, c.warns) {
				t.Errorf("Warnings = %q, want %q", cfg.Warnings, c.warns)
			}
		})
	}
}

func TestUnknownKeys(t *testing.T) {
	cfg, err := Load(write(t, "refresh_intervall = \"60s\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != DefaultInterval {
		t.Errorf("Interval = %v, want the default", cfg.Interval)
	}
	want := []string{`config: unknown key "refresh_intervall"`}
	if !reflect.DeepEqual(cfg.Warnings, want) {
		t.Errorf("Warnings = %q, want %q", cfg.Warnings, want)
	}
}

func TestUnknownKeyCleaned(t *testing.T) {
	cfg, err := Load(write(t, "\"bad\\u001b[31mkey\" = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("Warnings = %q, want one", cfg.Warnings)
	}
	if strings.ContainsRune(cfg.Warnings[0], 0x1b) {
		t.Errorf("warning %q contains an ESC byte", cfg.Warnings[0])
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name, body string
		line       int
		mention    string
	}{
		{"string for integer", `closed_days = "fourteen"`, 1, ""},
		{"float for integer", "# c\nclosed_days = 14.5", 2, ""},
		{"syntax", "closed_days = = 14", 1, ""},
		{"unknown show value", `show = ["open", "opne"]`, 1, "opne"},
		{"integer interval", `refresh_interval = 60`, 1, "missing unit"},
		{"word interval", `refresh_interval = "soon"`, 1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := write(t, c.body)
			_, err := Load(p)
			if err == nil {
				t.Fatal("want an error")
			}
			prefix := fmt.Sprintf("%s line %d: ", p, c.line)
			if !strings.HasPrefix(err.Error(), prefix) {
				t.Errorf("error %q does not start with %q", err, prefix)
			}
			if c.mention != "" && !strings.Contains(err.Error(), c.mention) {
				t.Errorf("error %q does not mention %q", err, c.mention)
			}
		})
	}
}
