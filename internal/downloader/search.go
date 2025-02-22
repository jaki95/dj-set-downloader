package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

type soundCloudClient struct {
	baseURL  string
	clientID string
}

func NewSoundCloudClient() (*soundCloudClient, error) {
	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("SOUNDCLOUD_CLIENT_ID not set")
	}
	return &soundCloudClient{
		baseURL:  "https://api-v2.soundcloud.com",
		clientID: clientID,
	}, nil
}

func (s *soundCloudClient) Search(query string) (string, error) {
	slog.Debug("searching soundcloud for set", "query", query)
	encodedQuery := url.QueryEscape(query)
	res, err := http.Get(fmt.Sprintf("%s/search?q=%s&client_id=%s", s.baseURL, encodedQuery, s.clientID))
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response interface{}

	err = json.Unmarshal(resBody, &response)
	if err != nil {
		return "", err
	}

	responseList := response.(map[string]any)["collection"].([]interface{})

	if len(responseList) == 0 {
		return "", fmt.Errorf("no results in search")
	}

	firstResult := responseList[0].(map[string]any)

	return firstResult["permalink_url"].(string), nil
}
