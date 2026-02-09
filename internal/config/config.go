package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	WorkPath       string `json:"WorkPath"`
	PictureDirName string `json:"PictureDirName"`
	SizeTablePath  string `json:"SizeTablePath"`
	ColorPicPath   string `json:"ColorPicPath"`
	Width          string `json:"width"`  // Keeping as string to match original JSON, but logic might need int
	Height         string `json:"height"` // Keeping as string to match original JSON
	Quality        int    `json:"quality"`
	ApiUrl         string `json:"ApiUrl"`
	FtpHost        string `json:"FtpHost"`
	FtpPort        string `json:"FtpPort"`
	FtpUser        string `json:"FtpUser"`
	FtpPassword    string `json:"FtpPassword"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		WorkPath:       "",
		PictureDirName: "org",
		SizeTablePath:  "",
		ColorPicPath:   "",
		Width:          "500",
		Height:         "700",
		Quality:        90,
		ApiUrl:         "http://localhost/api",
		FtpHost:        "localhost",
		FtpPort:        "21",
		FtpUser:        "user",
		FtpPassword:    "pass",
	}
}

// Load reads the config from the given path
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

// Save writes the config to the given path
func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetConfigPath returns the path to config.json relative to the executable
func GetConfigPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(execPath), "config.json")
}
