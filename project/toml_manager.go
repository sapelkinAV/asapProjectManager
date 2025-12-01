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

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configPath := filepath.Join(configDir, "asap-project-manager")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	filePath := filepath.Join(configPath, "projects.toml")

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

func SaveConfig(config *Config) error {

	home, err := os.UserHomeDir()

	if err != nil {

		return fmt.Errorf("failed to get user home directory: %w", err)

	}

	configDir := os.Getenv("XDG_CONFIG_HOME")

	if configDir == "" {

		configDir = filepath.Join(home, ".config")

	}

	configPath := filepath.Join(configDir, "asap-project-manager")

	if err := os.MkdirAll(configPath, 0755); err != nil {

		return fmt.Errorf("failed to create config directory: %w", err)

	}

	filePath := filepath.Join(configPath, "projects.toml")

	file, err := os.Create(filePath)

	if err != nil {

		return fmt.Errorf("failed to create config file: %w", err)

	}

	defer func() { _ = file.Close() }()

	encoder := toml.NewEncoder(file)

	if err := encoder.Encode(config); err != nil {

		return fmt.Errorf("failed to encode config to TOML: %w", err)

	}

	return nil

}

func GuessLanguage(path string) []string {
	var languages []string

	checks := map[string]string{
		"go.mod":           "go",
		"Cargo.toml":       "rust",
		"package.json":     "javascript",
		"requirements.txt": "python",
		"pom.xml":          "java",
		"build.gradle":     "java", // Gradle can be used for Java
		"Makefile":         "c",
	}

	for file, lang := range checks {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			// Avoid duplicates
			found := false
			for _, l := range languages {
				if l == lang {
					found = true
					break
				}
			}
			if !found {
				languages = append(languages, lang)
			}
		}
	}

	// Check for Lua files
	if entries, err := os.ReadDir(path); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".lua" {
				languages = append(languages, "lua")
				break // Only add once
			}
		}
	}

	return languages
}
