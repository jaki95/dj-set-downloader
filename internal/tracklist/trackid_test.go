package tracklist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferArtistFromTitle(t *testing.T) {
	testCases := []struct {
		title          string
		expectedArtist string
	}{
		// Common patterns
		{"Armin van Buuren - ASOT 1000", "Armin van Buuren"},
		{"John Digweed @ Burning Man 2023", "John Digweed"},
		{"Carl Cox | Tomorrowland 2022", "Carl Cox"},
		{"Solomun presents Diynamic Showcase", "Solomun"},
		{"Nina Kraviz live at Awakenings", "Nina Kraviz"},

		// More complex cases
		{"Adam Beyer b2b Ida Engberg - Time Warp 2021", "Adam Beyer b2b Ida Engberg"},
		{"Charlotte de Witte @ EDC Las Vegas Main Stage", "Charlotte de Witte"},
		{"Tale Of Us | Afterlife Opening Party Ibiza", "Tale Of Us"},
		{"Boris Brejcha presents FCKNG SERIOUS", "Boris Brejcha"},

		// First capitalized words heuristic
		{"Drumcode Radio Live", "Drumcode Radio"},
		{"Anjunadeep Edition Episode 400", "Anjunadeep Edition"},

		// Edge cases
		{"ABCD", ""},                  // Too short, no pattern matches
		{"", ""},                      // Empty string
		{"The Sound of Tomorrow", ""}, // Starts with "The" which is filtered out
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			result := inferArtistFromTitle(tc.title)
			assert.Equal(t, tc.expectedArtist, result)
		})
	}
}
