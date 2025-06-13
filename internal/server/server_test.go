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

func newTestServer(t *testing.T) *Server {
	cfg := &config.Config{
		LogLevel:       -4, // Slog debug level
		AudioProcessor: "ffmpeg",
		FileExtension:  "m4a",
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: t.TempDir(),
		},
	}
	server, err := New(cfg)
	require.NoError(t, err)
	return server
}

func TestHealthCheck(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestProcessRequestValidation(t *testing.T) {
	server := newTestServer(t)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			requestBody: ProcessUrlRequest{
				DownloadURL: "https://example.com/set.mp3",
				Tracklist:   `{"artist":"Test Artist", "name":"Test Mix", "tracks":[]}`,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "missing required fields",
			requestBody:    ProcessUrlRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				jsonData, _ := json.Marshal(tt.requestBody)
				body.Write(jsonData)
			}

			req, err := http.NewRequest("POST", "/api/v1/process", &body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetJobStatus_NotFound(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/api/v1/jobs/non-existent-job", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListJobs(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/api/v1/jobs", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "jobs")
	jobs := response["jobs"].([]interface{})
	assert.Empty(t, jobs)
}
