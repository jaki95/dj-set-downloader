package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel         int    `yaml:"log_level"`
	AudioProcessor   string `yaml:"audio_processor"`
	AudioSource      string `yaml:"audio_source"`
	TracklistSource  string `yaml:"tracklist_source"`
	TracklistWebsite string `yaml:"tracklist_website"`
	FileExtension    string `yaml:"file_extension"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`
}

type StorageConfig struct {
	// Type of storage: "local" or "gcs"
	Type string `yaml:"type"`

	// Local storage options
	OutputDir string `yaml:"output_dir"`

	// GCS storage options
	BucketName      string `yaml:"bucket_name"`
	ObjectPrefix    string `yaml:"object_prefix"`
	CredentialsFile string `yaml:"credentials_file"`
	PublicBaseURL   string `yaml:"public_base_url"`
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
	if config.Storage.Type == "" {
		config.Storage.Type = "local"
	}

	if config.Storage.OutputDir == "" {
		config.Storage.OutputDir = "output"
	}

	return config, nil
}
