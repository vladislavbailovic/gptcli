package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func Test_getGlobalConfigDir(t *testing.T) {
	home := os.Getenv("HOME")
	t.Run("should have config dir in home", func(t *testing.T) {
		if home == "" {
			t.Skip("no HOME env var")
		}

		dir, err := getGlobalConfigDir()
		if err != nil {
			t.Errorf("expected config directory, got error: %v", err)
		}
		if !strings.Contains(dir, home) {
			t.Errorf("expected config dir to be in HOME (%q): %q", home, dir)
		}
	})
	t.Run("should error out if HOME is not set", func(t *testing.T) {
		os.Setenv("HOME", "")
		defer func() { os.Setenv("HOME", home) }()

		dir, err := getGlobalConfigDir()
		if err == nil {
			t.Error("expected error")
		}
		if dir != "" {
			t.Errorf("expected config dir to be empty, got %q", dir)
		}
	})
}

func Test_getConfigFilepath(t *testing.T) {
	home := os.Getenv("HOME")
	t.Run("happy path", func(t *testing.T) {
		if home == "" {
			t.Skip("no HOME env var")
		}

		file, err := getConfigFilepath()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(file, home) {
			t.Errorf("HOME (%q) not in config path: %q", home, file)
		}
		if !strings.HasSuffix(file, configSourceFile) {
			t.Errorf("file path %q does not end with config file: %q", file, configSourceFile)
		}
	})
	t.Run("should error out if HOME is not set", func(t *testing.T) {
		os.Setenv("HOME", "")
		defer func() { os.Setenv("HOME", home) }()

		dir, err := getConfigFilepath()
		if err == nil {
			t.Error("expected error")
		}
		if dir != "" {
			t.Errorf("expected config filepath to be empty, got %q", dir)
		}
	})
}

func Test_hasConfigFile(t *testing.T) {
	home := os.Getenv("HOME")
	t.Run("should be no config file if HOME is not set", func(t *testing.T) {
		os.Setenv("HOME", "")
		defer func() { os.Setenv("HOME", home) }()

		if hasConfigFile() {
			t.Error("expected no config file")
		}
	})
}

func Test_Initialize(t *testing.T) {
	home := os.Getenv("HOME")
	t.Run("should error out if HOME is not set", func(t *testing.T) {
		if hasConfigFile() {
			t.Skip("already initialized")
		}
		os.Setenv("HOME", "")
		defer func() { os.Setenv("HOME", home) }()

		err := initializeConfig()
		if err == nil {
			t.Error("expected error")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Error("expected error to be typed as not-exist")
		}
	})
	t.Run("test init", func(t *testing.T) {
		if hasConfigFile() {
			t.Skip("already initialized")
		}

		err := initializeConfig()
		if err != nil {
			t.Error("expected initialization to succeed")
		}
	})
}
