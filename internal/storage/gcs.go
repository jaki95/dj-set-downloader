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
	client        *storage.Client
	bucket        string
	tempDir       string
	objectPrefix  string
	ctx           context.Context
	publicBaseURL string
}

// NewGCSStorage creates a new GCSStorage instance
func NewGCSStorage(ctx context.Context, bucketName, objectPrefix, tempDir, credentialsFile string) (*GCSStorage, error) {
	var client *storage.Client
	var err error

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

	// Create local temp directory if it doesn't exist
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &GCSStorage{
		client:       client,
		bucket:       bucketName,
		tempDir:      tempDir,
		objectPrefix: objectPrefix,
		ctx:          ctx,
	}, nil
}

// SaveDownloadedSet returns the path for storing a downloaded set
func (s *GCSStorage) SaveDownloadedSet(setName string, ext string) (string, error) {
	// For downloads, we'll always save to a temporary file first
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("%s.%s", setName, ext))

	// Create the temp directory if it doesn't exist
	if err := os.MkdirAll(s.tempDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return tempFile, nil
}

// SaveTrack returns the path for storing a track
func (s *GCSStorage) SaveTrack(setName, trackName string, ext string) (string, error) {
	// For tracks, we'll save to a temporary file first and then upload
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("%s_%s.%s", setName, trackName, ext))
	return tempFile, nil
}

// GetSetPath returns the path to a downloaded set
func (s *GCSStorage) GetSetPath(setName string, ext string) string {
	// The local temporary path for the set
	return filepath.Join(s.tempDir, fmt.Sprintf("%s.%s", setName, ext))
}

// GetCoverArtPath returns the path for temporary cover art storage
func (s *GCSStorage) GetCoverArtPath() string {
	return filepath.Join(s.tempDir, "cover_temp.jpg")
}

// CreateSetOutputDir creates the output directory for a set's tracks
// For GCS, we don't need to create directories, but we'll create a local temp directory
func (s *GCSStorage) CreateSetOutputDir(setName string) error {
	tempOutputDir := filepath.Join(s.tempDir, setName)
	return os.MkdirAll(tempOutputDir, os.ModePerm)
}

// Cleanup removes temporary files
func (s *GCSStorage) Cleanup() error {
	// Remove cover art
	if err := os.Remove(s.GetCoverArtPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup cover art: %w", err)
	}
	return nil
}

// GetReader returns a reader for a file
func (s *GCSStorage) GetReader(path string) (io.ReadCloser, error) {
	// If the path is local (in the temp directory), open the local file
	if strings.HasPrefix(path, s.tempDir) {
		return os.Open(path)
	}

	// Otherwise, assume it's a GCS object path
	objectName := strings.TrimPrefix(path, "/")
	if s.objectPrefix != "" {
		objectName = s.objectPrefix + "/" + objectName
	}

	return s.client.Bucket(s.bucket).Object(objectName).NewReader(s.ctx)
}

// GetWriter returns a writer for a file
func (s *GCSStorage) GetWriter(path string) (io.WriteCloser, error) {
	// If the path is local (in the temp directory), create/open the local file
	if strings.HasPrefix(path, s.tempDir) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		return os.Create(path)
	}

	// Otherwise, assume it's a GCS object path
	objectName := strings.TrimPrefix(path, "/")
	if s.objectPrefix != "" {
		objectName = s.objectPrefix + "/" + objectName
	}

	return s.client.Bucket(s.bucket).Object(objectName).NewWriter(s.ctx), nil
}

// FileExists checks if a file exists
func (s *GCSStorage) FileExists(path string) bool {
	// If the path is local, check the local filesystem
	if strings.HasPrefix(path, s.tempDir) {
		_, err := os.Stat(path)
		return err == nil
	}

	// Otherwise, check in GCS
	objectName := strings.TrimPrefix(path, "/")
	if s.objectPrefix != "" {
		objectName = s.objectPrefix + "/" + objectName
	}

	_, err := s.client.Bucket(s.bucket).Object(objectName).Attrs(s.ctx)
	return err == nil
}

// ListFiles lists files in a directory matching a pattern
func (s *GCSStorage) ListFiles(dir string, pattern string) ([]string, error) {
	// If dir is empty or refers to the temp directory, list local files
	if dir == "" || dir == s.tempDir {
		listDir := s.tempDir
		if dir != "" {
			listDir = dir
		}

		files, err := os.ReadDir(listDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		var results []string
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Match pattern (simple prefix for now)
			if pattern != "" && !strings.HasPrefix(file.Name(), pattern) {
				continue
			}

			results = append(results, filepath.Join(listDir, file.Name()))
		}

		return results, nil
	}

	// Otherwise, list objects in GCS
	prefix := dir
	if dir != "" && !strings.HasSuffix(dir, "/") {
		prefix = dir + "/"
	}

	if s.objectPrefix != "" && !strings.HasPrefix(prefix, s.objectPrefix) {
		prefix = s.objectPrefix + "/" + prefix
	}

	it := s.client.Bucket(s.bucket).Objects(s.ctx, &storage.Query{
		Prefix: prefix,
	})

	var results []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}

		// Skip directories (objects ending with /)
		if strings.HasSuffix(attrs.Name, "/") {
			continue
		}

		// Get the base filename
		fileName := filepath.Base(attrs.Name)

		// Match pattern (simple prefix for now)
		if pattern != "" && !strings.HasPrefix(fileName, pattern) {
			continue
		}

		// Construct a GCS path
		results = append(results, attrs.Name)
	}

	return results, nil
}

// UploadFile uploads a local file to GCS
func (s *GCSStorage) UploadFile(localPath, objectName string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer f.Close()

	if s.objectPrefix != "" {
		objectName = s.objectPrefix + "/" + objectName
	}

	ctx, cancel := context.WithTimeout(s.ctx, time.Minute*5)
	defer cancel()

	wc := s.client.Bucket(s.bucket).Object(objectName).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return "", fmt.Errorf("failed to copy file to GCS: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	// Return the public URL if available, or just the object name
	if s.publicBaseURL != "" {
		return fmt.Sprintf("%s/%s", s.publicBaseURL, objectName), nil
	}
	return objectName, nil
}

// Close closes the GCS client
func (s *GCSStorage) Close() error {
	return s.client.Close()
}
