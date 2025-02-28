package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel        int    `yaml:"log_level"`
	AudioProcessor  string `yaml:"audio_processor"`
	AudioSource     string `yaml:"audio_source"`
	TracklistSource string `yaml:"tracklist_source"`
	FileExtension   string `yaml:"file_extension"`
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

	return config, nil
}
