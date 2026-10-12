package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
