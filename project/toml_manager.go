package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Project struct {
	Name     string `toml:"name"`
	Path     string `toml:"path"`
	Language string `toml:"language"`
}

type Config struct {
	Projects []Project `toml:"projects"`
}

func LoadConfig() (*Config, error) {

	home, err := os.UserHomeDir()

	if err != nil {

		return nil, fmt.Errorf("failed to get user home directory: %w", err)

	}

	filePath := filepath.Join(home, ".asap_projects.toml")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {

		return &Config{Projects: []Project{}}, nil

	}

	var config Config

	if _, err := toml.DecodeFile(filePath, &config); err != nil {

		return nil, fmt.Errorf("failed to decode TOML file: %w", err)

	}

	for i := range config.Projects {
		if !filepath.IsAbs(config.Projects[i].Path) {
			config.Projects[i].Path = filepath.Join(home, config.Projects[i].Path)
		}
	}

	return &config, nil

}
