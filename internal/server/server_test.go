package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/job"
)

func newTestServer(t *testing.T) *Server {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
		},
		Storage: config.StorageConfig{
			Type:      "local",
			OutputDir: t.TempDir(),
		},
	}
	server := New(cfg)
	server.router = server.setupTestRoutes()
	return server
}

func (s *Server) setupTestRoutes() *gin.Engine {
	router := gin.Default()
	s.setupRoutes(router)
	return router
}

func TestHealthCheck(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}
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
			requestBody: job.Request{
				URL:       "https://example.com/set.mp3",
				Tracklist: `{"artist":"Test Artist", "name":"Test Mix", "tracks":[{"name":"Track 1","startTime":"00:00","endTime":"03:00"}]}`,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "missing required fields",
			requestBody:    job.Request{},
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

			req, err := http.NewRequest("POST", "/api/process", &body)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGetJobStatus_NotFound(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/api/jobs/non-existent-job", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestListJobs(t *testing.T) {
	server := newTestServer(t)
	req, err := http.NewRequest("GET", "/api/jobs", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := response["jobs"]; !exists {
		t.Error("Expected 'jobs' field in response")
	}
}
