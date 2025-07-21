package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("should load a valid config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test_config.yaml")
		configContent := `
log_level: -4
audio_processor: ffmpeg
file_extension: m4a
storage:
  type: "local"
  output_dir: "test_output"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		cfg, err := Load(configPath)

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, -4, cfg.LogLevel)
		assert.Equal(t, "ffmpeg", cfg.AudioProcessor)
		assert.Equal(t, "m4a", cfg.FileExtension)
		assert.Equal(t, "local", cfg.Storage.Type)
		assert.Equal(t, "test_output", cfg.Storage.OutputDir)
	})

	t.Run("should return an error for a non-existent file", func(t *testing.T) {
		cfg, err := Load("non_existent_file.yaml")

		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("should return an error for invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid_config.yaml")
		configContent := `
log_level: -4
invalid_yaml: [this is not valid yaml
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		cfg, err := Load(configPath)

		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("should set default values", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test_config.yaml")
		configContent := `
log_level: 0
file_extension: mp3
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		cfg, err := Load(configPath)

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "local", cfg.Storage.Type)
		assert.Equal(t, "output", cfg.Storage.OutputDir)
	})
}
