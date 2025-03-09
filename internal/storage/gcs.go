package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSStorage implements the Storage interface for Google Cloud Storage
type GCSStorage struct {
	client       *storage.Client
	bucket       string
	objectPrefix string
	ctx          context.Context
	tempLocalDir string
}

// NewGCSStorage creates a new GCSStorage instance
func NewGCSStorage(ctx context.Context, bucketName, objectPrefix, credentialsFile string) (*GCSStorage, error) {
	var client *storage.Client
	var err error

	// Create a temporary local directory for operation that require local files
	tempDir, err := os.MkdirTemp("", "dj-set-downloader-gcs-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create a client
	if credentialsFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	} else {
		// Use application default credentials
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSStorage{
		client:       client,
		bucket:       bucketName,
		objectPrefix: objectPrefix,
		ctx:          ctx,
		tempLocalDir: tempDir,
	}, nil
}

// CreateDir creates a "directory" in GCS (not actually needed since GCS is object-based)
func (s *GCSStorage) CreateDir(path string) error {
	// GCS doesn't have directories, but we can create a zero-byte object with a trailing slash
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	objectName := s.getObjectName(path)

	// Create an empty object
	wc := s.client.Bucket(s.bucket).Object(objectName).NewWriter(s.ctx)
	wc.ContentType = "application/x-directory"
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to create directory placeholder: %w", err)
	}

	return nil
}

// WriteFile writes data to a file in GCS
func (s *GCSStorage) WriteFile(path string, data []byte) error {
	objectName := s.getObjectName(path)

	wc := s.client.Bucket(s.bucket).Object(objectName).NewWriter(s.ctx)
	if _, err := wc.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to finalize write: %w", err)
	}

	return nil
}

// ReadFile reads the contents of a file from GCS
func (s *GCSStorage) ReadFile(path string) ([]byte, error) {
	objectName := s.getObjectName(path)

	rc, err := s.client.Bucket(s.bucket).Object(objectName).NewReader(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open object for reading: %w", err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// FileExists checks if a file exists in GCS
func (s *GCSStorage) FileExists(path string) (bool, error) {
	objectName := s.getObjectName(path)

	_, err := s.client.Bucket(s.bucket).Object(objectName).Attrs(s.ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if object exists: %w", err)
	}

	return true, nil
}

// DeleteFile removes a file from GCS
func (s *GCSStorage) DeleteFile(path string) error {
	objectName := s.getObjectName(path)

	if err := s.client.Bucket(s.bucket).Object(objectName).Delete(s.ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// ListDir lists files in a "directory" in GCS
func (s *GCSStorage) ListDir(path string) ([]string, error) {
	// Ensure path ends with a slash
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	prefix := s.getObjectName(path)

	it := s.client.Bucket(s.bucket).Objects(s.ctx, &storage.Query{
		Prefix: prefix,
	})

	var result []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Skip the directory placeholder itself
		if attrs.Name == prefix {
			continue
		}

		result = append(result, attrs.Name)
	}

	return result, nil
}

// CreateProcessDir creates a process directory
func (s *GCSStorage) CreateProcessDir(processID string) (string, error) {
	// Create the directory structure in GCS
	processesPath := filepath.Join("processes", processID)
	if err := s.CreateDir(processesPath); err != nil {
		return "", err
	}

	downloadPath := s.GetDownloadDir(processID)
	if err := s.CreateDir(downloadPath); err != nil {
		return "", err
	}

	tempPath := s.GetTempDir(processID)
	if err := s.CreateDir(tempPath); err != nil {
		return "", err
	}

	// Also create local temp directories for the process
	localProcessDir := filepath.Join(s.tempLocalDir, processID)
	if err := os.MkdirAll(localProcessDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local process directory: %w", err)
	}

	localDownloadDir := filepath.Join(localProcessDir, "download")
	if err := os.MkdirAll(localDownloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local download directory: %w", err)
	}

	localTempDir := filepath.Join(localProcessDir, "temp")
	if err := os.MkdirAll(localTempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local temp directory: %w", err)
	}

	return processesPath, nil
}

// GetDownloadDir returns the download directory path for a process
func (s *GCSStorage) GetDownloadDir(processID string) string {
	return filepath.Join("processes", processID, "download")
}

// GetTempDir returns the temporary directory path for a process
func (s *GCSStorage) GetTempDir(processID string) string {
	return filepath.Join("processes", processID, "temp")
}

// GetOutputDir returns the output directory path for a set
func (s *GCSStorage) GetOutputDir(setName string) string {
	return filepath.Join("output", setName)
}

// CleanupProcessDir removes a process directory
func (s *GCSStorage) CleanupProcessDir(processID string) error {
	// Sleep for a bit to ensure all files are finished being processed
	time.Sleep(5 * time.Minute)

	// Delete all objects with the process prefix
	processPath := filepath.Join("processes", processID)
	prefix := s.getObjectName(processPath)

	it := s.client.Bucket(s.bucket).Objects(s.ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects for cleanup: %w", err)
		}

		// Delete the object
		if err := s.client.Bucket(s.bucket).Object(attrs.Name).Delete(s.ctx); err != nil {
			return fmt.Errorf("failed to delete object during cleanup: %w", err)
		}
	}

	// Also clean up the local temp directories
	localProcessDir := filepath.Join(s.tempLocalDir, processID)
	if err := os.RemoveAll(localProcessDir); err != nil {
		return fmt.Errorf("failed to remove local process directory: %w", err)
	}

	return nil
}

// Close cleans up resources
func (s *GCSStorage) Close() error {
	// Clean up the temp directory
	if err := os.RemoveAll(s.tempLocalDir); err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}

	// Close the client
	return s.client.Close()
}

// getObjectName constructs the full object name with prefix
func (s *GCSStorage) getObjectName(path string) string {
	if s.objectPrefix == "" {
		return path
	}
	return filepath.Join(s.objectPrefix, path)
}

// GetLocalDownloadDir returns a local filesystem path for the download directory
// For GCSStorage, this is a temporary local directory that can be used by external tools
func (s *GCSStorage) GetLocalDownloadDir(processID string) string {
	return filepath.Join(s.tempLocalDir, processID, "download")
}
