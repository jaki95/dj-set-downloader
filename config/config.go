package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel        int           `yaml:"log_level"`
	AudioSource     string        `yaml:"audio_source"`
	TracklistSource string        `yaml:"tracklist_source"`
	FileExtension   string        `yaml:"file_extension"`
	AudioProcessor  string        `yaml:"audio_processor"`
	Storage         StorageConfig `yaml:"storage"`
}

type StorageConfig struct {
	Type            string `yaml:"type"`             // "local" or "gcs"
	DataDir         string `yaml:"data_dir"`         // Base directory for local storage
	OutputDir       string `yaml:"output_dir"`       // Output directory for tracks
	BucketName      string `yaml:"bucket_name"`      // GCS bucket name
	ObjectPrefix    string `yaml:"object_prefix"`    // Prefix for GCS objects
	CredentialsFile string `yaml:"credentials_file"` // Path to GCS credentials file
}

func Load(path string) (*Config, error) {
	// Default configuration
	config := &Config{
		AudioSource:     "soundcloud",
		TracklistSource: "trackids",
		FileExtension:   "mp3",
		AudioProcessor:  "ffmpeg",
		Storage: StorageConfig{
			Type:      "local",
			DataDir:   "storage",
			OutputDir: "output",
		},
	}

	// If no path is provided, return default config
	if path == "" {
		return config, nil
	}

	// Load configuration from file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
