package tracklist

import (
	"context"
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// Scraper scrapes a tracklist from a given source.
type Scraper interface {
	Scrape(ctx context.Context, source string) (*domain.Tracklist, error)
	Name() string
}

// CompositeScraper tries multiple scrapers in sequence until one succeeds
type CompositeScraper struct {
	scrapers []Scraper
}

func (c *CompositeScraper) Name() string {
	return "composite"
}

func NewCompositeScraper(config *config.Config) (*CompositeScraper, error) {
	// Create 1001Tracklists importer with the search client
	scraper, err := New1001TracklistsScraper(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1001tracklists importer: %v", err)
	}

	return &CompositeScraper{
		scrapers: []Scraper{
			scraper,
			NewTrackIDScraper(),
		},
	}, nil
}

func (c *CompositeScraper) Scrape(ctx context.Context, source string) (*domain.Tracklist, error) {
	var errors []error
	for _, scraper := range c.scrapers {
		tracklist, err := scraper.Scrape(ctx, source)
		if err == nil {
			return tracklist, nil
		}
		errors = append(errors, fmt.Errorf("%s: %v", scraper.Name(), err))
	}
	return nil, fmt.Errorf("all scrapers failed: %v", errors)
}

func NewScraper(config *config.Config) (Scraper, error) {
	switch config.TracklistSource {
	case "1001tracklists":
		return New1001TracklistsScraper(config)
	case "trackid":
		return NewTrackIDScraper(), nil
	default:
		return NewCompositeScraper(config)
	}
}
