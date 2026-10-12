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
	"time"

	"github.com/BurntSushi/toml"
)

const (
	// DefaultInterval is the refresh interval when the file does not set one.
	DefaultInterval = 60 * time.Second
	// MinInterval is the smallest refresh interval. Smaller values are raised.
	MinInterval = 15 * time.Second

	maxFileSize = 1 << 20
)

// Config holds the effective settings.
type Config struct {
	Path     string
	Interval time.Duration
}

// Default returns the settings used when no file exists.
func Default() Config {
	return Config{Interval: DefaultInterval}
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

type fileShape struct {
	RefreshInterval *duration `toml:"refresh_interval"`
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
	if _, err := toml.Decode(string(data), &shape); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if shape.RefreshInterval != nil {
		cfg.Interval = max(time.Duration(*shape.RefreshInterval), MinInterval)
	}
	return cfg, nil
}
