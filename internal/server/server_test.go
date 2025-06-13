package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel:       -4,
		AudioProcessor: "ffmpeg",
		AudioSource:    "soundcloud",
		FileExtension:  "m4a",
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: "test-output",
		},
	}

	// Create server
	server, err := New(cfg)
	require.NoError(t, err)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Perform request
	server.router.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "dj-set-processor", response["service"])
	assert.NotEmpty(t, response["timestamp"])
}

func TestProcessRequestValidation(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel:       -4,
		AudioProcessor: "ffmpeg",
		AudioSource:    "soundcloud",
		FileExtension:  "m4a",
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: "test-output",
		},
	}

	// Create server
	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			requestBody: ProcessUrlRequest{
				DownloadURL:        "https://soundcloud.com/example/set",
				Tracklist:          `{"title":"Test Set","tracks":[{"title":"Track 1","startTime":0,"endTime":180},{"title":"Track 2","startTime":180,"endTime":360}]}`,
				FileExtension:      "m4a",
				MaxConcurrentTasks: 4,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "missing required fields",
			requestBody: ProcessUrlRequest{
				FileExtension: "m4a",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				jsonData, _ := json.Marshal(tt.requestBody)
				body.Write(jsonData)
			}

			// Create test request
			req, err := http.NewRequest("POST", "/api/v1/process", &body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Perform request
			server.router.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusAccepted {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response["jobId"])
				assert.Equal(t, "accepted", response["status"])
			}
		})
	}
}

func TestGetJobStatus_NotFound(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel:       -4,
		AudioProcessor: "ffmpeg",
		AudioSource:    "soundcloud",
		FileExtension:  "m4a",
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: "test-output",
		},
	}

	// Create server
	server, err := New(cfg)
	require.NoError(t, err)

	// Create test request for non-existent job
	req, err := http.NewRequest("GET", "/api/v1/jobs/non-existent-job", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Perform request
	server.router.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Job not found", response["error"])
}

func TestListJobs(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel:       -4,
		AudioProcessor: "ffmpeg",
		AudioSource:    "soundcloud",
		FileExtension:  "m4a",
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: "test-output",
		},
	}

	// Create server
	server, err := New(cfg)
	require.NoError(t, err)

	// Create test request
	req, err := http.NewRequest("GET", "/api/v1/jobs", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Perform request
	server.router.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response structure
	assert.Contains(t, response, "jobs")
	assert.Contains(t, response, "page")
	assert.Contains(t, response, "pageSize")
	assert.Contains(t, response, "totalJobs")
	assert.Contains(t, response, "totalPages")

	// Initially should have no jobs
	jobs := response["jobs"].([]interface{})
	assert.Empty(t, jobs)
}
