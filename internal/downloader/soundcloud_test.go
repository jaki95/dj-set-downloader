package downloader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSoundCloudDownloaderWithoutClientID(t *testing.T) {
	// Ensure SOUNDCLOUD_CLIENT_ID is not set
	originalClientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	defer os.Setenv("SOUNDCLOUD_CLIENT_ID", originalClientID) // Restore original value

	os.Unsetenv("SOUNDCLOUD_CLIENT_ID")

	// Test
	client, err := NewSoundCloudDownloader()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "SOUNDCLOUD_CLIENT_ID not set")
}

func TestNewSoundCloudDownloaderWithClientID(t *testing.T) {
	// Set test client ID
	originalClientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	defer os.Setenv("SOUNDCLOUD_CLIENT_ID", originalClientID) // Restore original value

	os.Setenv("SOUNDCLOUD_CLIENT_ID", "test_client_id")

	// Test
	client, err := NewSoundCloudDownloader()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test_client_id", client.clientID)
	assert.Equal(t, "https://api-v2.soundcloud.com", client.baseURL)
}

func TestFindURLWithEmptyQuery(t *testing.T) {
	// Setup
	client := &soundCloudClient{
		baseURL:  "https://api-v2.soundcloud.com",
		clientID: "test_client_id",
	}

	// Test
	url, err := client.FindURL("")

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
// 	url, err := client.FindURL("Bonobo Essential Mix")
//
// 	// Assert
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, url)
// 	assert.Contains(t, url, "soundcloud.com")
// }

// Download tests would require mocking the command line tool 'scdl'
// which is complex for a unit test. Integration tests could be added
// but would require an actual SoundCloud client ID and would make network calls.
