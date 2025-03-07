package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test config file
	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
log_level: -4
audio_processor: ffmpeg
audio_source: soundcloud
tracklist_source: trackids
file_extension: mp3
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading the config
	cfg, err := Load(configPath)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, -4, cfg.LogLevel)
	assert.Equal(t, "ffmpeg", cfg.AudioProcessor)
	assert.Equal(t, "soundcloud", cfg.AudioSource)
	assert.Equal(t, "trackids", cfg.TracklistSource)
	assert.Equal(t, "mp3", cfg.FileExtension)
}

func TestLoadNonExistentFile(t *testing.T) {
	// Test loading a non-existent config file
	cfg, err := Load("non_existent_file.yaml")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadInvalidYAML(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an invalid YAML file
	configPath := filepath.Join(tempDir, "invalid_config.yaml")
	configContent := `
log_level: -4
audio_processor: ffmpeg
audio_source: soundcloud
tracklist_source: trackids
file_extension: mp3
invalid_yaml: [this is not valid yaml
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading the invalid config
	cfg, err := Load(configPath)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
