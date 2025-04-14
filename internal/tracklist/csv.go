package tracklist

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type CSVImporter struct {
}

func NewCSVImporter() *CSVImporter {
	return &CSVImporter{}
}

func (c *CSVImporter) Name() string {
	return "csv"
}

func (c *CSVImporter) Import(ctx context.Context, filePath string) (*domain.Tracklist, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1

	tracklist, err := c.parseTracklist(reader)
	if err != nil {
		return nil, err
	}

	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in CSV file")
	}

	return tracklist, nil
}

func (c *CSVImporter) parseTracklist(reader *csv.Reader) (*domain.Tracklist, error) {
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	slog.Debug("Header row", "header", header)

	metadata, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV metadata row: %w", err)
	}
	slog.Debug("Metadata row", "metadata", metadata)

	if len(metadata) < 9 {
		return nil, fmt.Errorf("invalid CSV metadata row: expected at least 9 fields, got %d", len(metadata))
	}

	tracklist := &domain.Tracklist{
		Artist: metadata[1],
		Name:   metadata[2],
	}
	totalDuration := metadata[0]
	slog.Debug("Tracklist metadata",
		"totalDuration", totalDuration,
		"artist", tracklist.Artist,
		"name", tracklist.Name)

	trackCounter := 1
	previousEndTime := ""

	if err := c.processFirstTrack(tracklist, metadata, &trackCounter, &previousEndTime); err != nil {
		return nil, err
	}

	if err := c.processRemainingTracks(reader, tracklist, totalDuration, &trackCounter, &previousEndTime); err != nil {
		return nil, err
	}

	return tracklist, nil
}

func (c *CSVImporter) processFirstTrack(tracklist *domain.Tracklist, metadata []string, trackCounter *int, previousEndTime *string) error {
	startTime, endTime := metadata[4], metadata[5]
	artist, title := metadata[6], metadata[7]

	if startTime != "00:00:00" {
		c.addIDTrack(tracklist, "00:00:00", startTime, *trackCounter)
		*trackCounter++
	}

	c.addTrack(tracklist, artist, title, startTime, endTime, *trackCounter)
	*trackCounter++
	*previousEndTime = endTime
	return nil
}

func (c *CSVImporter) processRemainingTracks(reader *csv.Reader, tracklist *domain.Tracklist, totalDuration string, trackCounter *int, previousEndTime *string) error {
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}
		if len(record) < 6 {
			return fmt.Errorf("invalid CSV record: expected at least 6 fields, got %d", len(record))
		}

		startTime, endTime := record[4], record[5]
		artist, title := record[6], record[7]

		c.handleTrackGap(tracklist, *previousEndTime, startTime, trackCounter)
		c.addTrack(tracklist, artist, title, startTime, endTime, *trackCounter)
		*trackCounter++
		*previousEndTime = endTime
	}

	return c.handleFinalGap(tracklist, *previousEndTime, totalDuration, trackCounter)
}

func (c *CSVImporter) handleTrackGap(tracklist *domain.Tracklist, previousEndTime, startTime string, trackCounter *int) {
	if previousEndTime == "" {
		return
	}

	gapDuration := c.calculateDuration(previousEndTime, startTime)
	slog.Debug("Track gap",
		"previousEnd", previousEndTime,
		"start", startTime,
		"duration", gapDuration)

	if gapDuration <= 0 {
		return
	}

	if gapDuration < 60 {
		midpointTime := c.calculateMidpoint(previousEndTime, startTime)
		slog.Debug("Gap < 60s, setting midpoint", "midpoint", midpointTime)
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = midpointTime
	} else {
		slog.Debug("Gap >= 60s, inserting ID track",
			"start", previousEndTime,
			"end", startTime)
		c.addIDTrack(tracklist, previousEndTime, startTime, *trackCounter)
		*trackCounter++
	}
}

func (c *CSVImporter) handleFinalGap(tracklist *domain.Tracklist, previousEndTime, totalDuration string, trackCounter *int) error {
	if previousEndTime == "" || previousEndTime == totalDuration {
		return nil
	}

	finalGap := c.calculateDuration(previousEndTime, totalDuration)
	if finalGap <= 0 || finalGap >= 86400 {
		return fmt.Errorf("invalid final gap (%d seconds)", finalGap)
	}

	if finalGap < 60 {
		slog.Debug("Final gap < 60s, extending last track", "end", totalDuration)
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = totalDuration
	} else {
		slog.Debug("Final gap >= 60s, inserting final ID track",
			"start", previousEndTime,
			"end", totalDuration)
		c.addIDTrack(tracklist, previousEndTime, totalDuration, *trackCounter)
	}
	return nil
}

func (c *CSVImporter) addTrack(tracklist *domain.Tracklist, artist, title, startTime, endTime string, trackNumber int) {
	tracklist.Tracks = append(tracklist.Tracks, &domain.Track{
		Artist:      artist,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		TrackNumber: trackNumber,
	})
}

func (c *CSVImporter) addIDTrack(tracklist *domain.Tracklist, startTime, endTime string, trackNumber int) {
	c.addTrack(tracklist, "ID", "ID", startTime, endTime, trackNumber)
	slog.Debug("ID track added",
		"start", startTime,
		"end", endTime)
}

func (c *CSVImporter) calculateDuration(startTime, endTime string) int {
	start := c.parseTime(startTime)
	end := c.parseTime(endTime)
	if start.IsZero() || end.IsZero() {
		return -1
	}
	return int(end.Sub(start).Seconds())
}

func (c *CSVImporter) parseTime(timeStr string) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return time.Time{}
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.Atoi(parts[2])

	return time.Date(0, 1, 1, hours, minutes, seconds, 0, time.UTC)
}

func (c *CSVImporter) calculateMidpoint(startTime, endTime string) string {
	start := c.parseTime(startTime)
	end := c.parseTime(endTime)
	duration := end.Sub(start)
	midpoint := start.Add(duration / 2)
	return fmt.Sprintf("%02d:%02d:%02d", midpoint.Hour(), midpoint.Minute(), midpoint.Second())
}
