// Package config reads prv's settings file. It never writes the file and
// prints nothing.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/sbartram/prv/internal/domain"
)

const (
	// DefaultInterval is the refresh interval when the file does not set one.
	DefaultInterval = 60 * time.Second
	// MinInterval is the smallest refresh interval. Smaller values are raised.
	MinInterval = 15 * time.Second

	// MinClosedDays and MaxClosedDays bound the closed window. The config value
	// is clamped to them; the --closed-days flag is rejected outside them.
	MinClosedDays = 1
	MaxClosedDays = 365

	maxFileSize = 1 << 20
)

// Config holds the effective settings.
type Config struct {
	Path       string
	Interval   time.Duration
	ClosedDays int
	Show       domain.StateSet
	// Warnings are problems that do not stop prv, such as an unknown key.
	Warnings []string
}

// Default returns the settings used when no file exists.
func Default() Config {
	return Config{Interval: DefaultInterval, ClosedDays: domain.DefaultClosedDays, Show: domain.AllStates}
}

// Path returns the settings file path: $XDG_CONFIG_HOME/prv/config.toml when
// XDG_CONFIG_HOME is set and absolute, else ~/.config/prv/config.toml. It does
// not use the standard per-user config helper, which returns
// ~/Library/Application Support on macOS.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, "prv", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "prv", "config.toml"), nil
}

// duration reads a Go duration string. A native time.Duration field would read
// the TOML integer 60 as 60 ns.
type duration time.Duration

func (d *duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b))
	if err != nil {
		return fmt.Errorf("refresh_interval: %w", err)
	}
	*d = duration(v)
	return nil
}

// showState is one element of the show array.
type showState string

func (s *showState) UnmarshalText(b []byte) error {
	switch v := string(b); v {
	case "open", "merged", "closed":
		*s = showState(v)
		return nil
	default:
		return fmt.Errorf("show: unknown state %q (want open, merged or closed)", v)
	}
}

type fileShape struct {
	RefreshInterval *duration    `toml:"refresh_interval"`
	ClosedDays      *int         `toml:"closed_days"`
	Show            *[]showState `toml:"show"`
}

// wrongType matches the errors toml returns for a value of the wrong type.
// They are plain errors, not toml.ParseError values.
var wrongType = regexp.MustCompile(`^toml: line (\d+) \(last key "[^"]*"\): (.*)$`)

// decodeError gives "<path> line N: <problem>" when the line is known.
func decodeError(path string, err error) error {
	var pe toml.ParseError
	if errors.As(err, &pe) {
		return fmt.Errorf("%s line %d: %s", path, pe.Position.Line, domain.Clean(pe.Message))
	}
	if m := wrongType.FindStringSubmatch(err.Error()); m != nil {
		return fmt.Errorf("%s line %s: %s", path, m[1], domain.Clean(m[2]))
	}
	return fmt.Errorf("%s: %s", path, domain.Clean(err.Error()))
}

var stateOf = map[showState]domain.State{
	"open":   domain.StateOpen,
	"merged": domain.StateMerged,
	"closed": domain.StateClosed,
}

// Load reads the settings file at path. A missing file gives the defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	cfg.Path = path

	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	f.Close()
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(data) > maxFileSize {
		return Config{}, fmt.Errorf("%s: file is larger than 1 MiB", path)
	}

	var shape fileShape
	md, err := toml.Decode(string(data), &shape)
	if err != nil {
		return Config{}, decodeError(path, err)
	}
	if shape.RefreshInterval != nil {
		cfg.Interval = max(time.Duration(*shape.RefreshInterval), MinInterval)
	}
	if shape.ClosedDays != nil {
		cfg.ClosedDays = min(max(*shape.ClosedDays, MinClosedDays), MaxClosedDays)
	}
	if shape.Show != nil {
		var set domain.StateSet
		for _, v := range *shape.Show {
			if st := stateOf[v]; !set.Has(st) {
				set = set.Toggle(st)
			}
		}
		if set == 0 {
			set = domain.AllStates
			cfg.Warnings = append(cfg.Warnings, "config: show is empty; showing all states")
		}
		cfg.Show = set
	}
	for _, key := range md.Undecoded() {
		cfg.Warnings = append(cfg.Warnings, `config: unknown key "`+domain.Clean(key.String())+`"`)
	}
	return cfg, nil
}
