package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel      int    `yaml:"log_level"`
	FileExtension string `yaml:"file_extension"`

	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type StorageConfig struct {
	// Type of storage: "local"
	Type string `yaml:"type"`

	// Local storage options
	OutputDir string `yaml:"output_dir"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *Config

	// Unmarshal the YAML data into the struct
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Set defaults if not provided
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}

	if config.Storage.Type == "" {
		config.Storage.Type = "local"
	}

	if config.Storage.OutputDir == "" {
		config.Storage.OutputDir = "output"
	}

	return config, nil
}
