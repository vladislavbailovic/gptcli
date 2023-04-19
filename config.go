package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	configSourcePath string = "incsub/gptcli"
	configSourceFile string = "config.json"
)

var ErrConfig = errors.New("configuration error")

type Config struct {
	Token string
	Model gptModel
}

func hasConfigFile() bool {
	file, err := getConfigFilepath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

func loadConfig() Config {
	var config Config

	cfgFile, err := getConfigFilepath()
	if err != nil {
		return config
	}

	if file, err := os.Open(cfgFile); err != nil {
		return config
	} else if err := json.NewDecoder(file).Decode(&config); err != nil {
		return config
	}

	return config
}

func initializeConfig() error {
	if hasConfigFile() {
		return nil // Already initialized
	}

	cfgDir, err := getGlobalConfigDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(cfgDir); err != nil {
		// No directory, make one
		if err := os.MkdirAll(cfgDir, 0700); err != nil {
			return err
		}
	}

	cfgFile, err := getConfigFilepath()
	if err != nil {
		return err
	}

	initial, err := json.MarshalIndent(Config{}, "", "\t")
	if err := os.WriteFile(cfgFile, initial, 0600); err != nil {
		return err
	}

	return nil
}

func getGlobalConfigDir() (string, error) {
	var fullpath string
	if dir, err := os.UserConfigDir(); err == nil { // If all is well and we have standard config dir
		fullpath = filepath.Join(dir, configSourcePath)
	} else if dir, err := os.UserHomeDir(); err == nil { // If all is well and we have standard config dir
		fullpath = filepath.Join(dir, "."+configSourcePath)
	}

	if fullpath != "" {
		return fullpath, nil
	}

	return fullpath, ErrConfig
}

func getConfigFilepath() (string, error) {
	cfgDir, err := getGlobalConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cfgDir, configSourceFile), nil
}
