package downloader

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestEnv() func() {
	// Save original environment variables
	originalAPIKey := os.Getenv("GOOGLE_API_KEY")
	originalSCID := os.Getenv("GOOGLE_SEARCH_ID_SOUNDCLOUD")
	originalClientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")

	// Set test environment variables
	os.Setenv("GOOGLE_API_KEY", "test-api-key")
	os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", "test-soundcloud-id")
	os.Setenv("SOUNDCLOUD_CLIENT_ID", "test-client-id")

	// Return cleanup function
	return func() {
		os.Setenv("GOOGLE_API_KEY", originalAPIKey)
		os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", originalSCID)
		os.Setenv("SOUNDCLOUD_CLIENT_ID", originalClientID)
	}
}

func TestNewSoundCloudDownloaderWithoutClientID(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	// Unset SOUNDCLOUD_CLIENT_ID for this test
	os.Unsetenv("SOUNDCLOUD_CLIENT_ID")

	// Test
	client, err := NewSoundCloudDownloader()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "SOUNDCLOUD_CLIENT_ID not set")
}

func TestNewSoundCloudDownloaderWithClientID(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	// Test
	client, err := NewSoundCloudDownloader()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test-client-id", client.clientID)
	assert.Equal(t, "https://api-v2.soundcloud.com", client.baseURL)
}

func TestFindURLWithEmptyQuery(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	// Setup
	client := &soundCloudClient{
		baseURL:  "https://api-v2.soundcloud.com",
		clientID: "test-client-id",
	}

	// Test
	url, err := client.FindURL(context.Background(), "")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "invalid query")
}

// Integration test - commented out as it requires real API calls
// func TestFindURLWithValidQuery(t *testing.T) {
// 	// Skip if we don't have a real client ID
// 	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
// 	if clientID == "" {
// 		t.Skip("SOUNDCLOUD_CLIENT_ID not set, skipping integration test")
// 	}
//
// 	client := &soundCloudClient{
// 		baseURL:  "https://api-v2.soundcloud.com",
// 		clientID: clientID,
// 	}
//
// 	// Test with a popular DJ name
// 	url, err := client.FindURL(context.Background(), "Bonobo Essential Mix")
//
// 	// Assert
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, url)
// 	assert.Contains(t, url, "soundcloud.com")
// }

// Download tests would require mocking the command line tool 'scdl'
// which is complex for a unit test. Integration tests could be added
// but would require an actual SoundCloud client ID and would make network calls.
