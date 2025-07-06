package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		expected float64
		wantErr  bool
	}{
		{
			name:     "zero time",
			timeStr:  "00:00:00",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "minutes and seconds",
			timeStr:  "05:30",
			expected: 330, // 5*60 + 30
			wantErr:  false,
		},
		{
			name:     "hours, minutes, seconds",
			timeStr:  "01:30:45",
			expected: 5445, // 1*3600 + 30*60 + 45
			wantErr:  false,
		},
		{
			name:     "invalid format",
			timeStr:  "1:30:45",
			expected: 5445,
			wantErr:  false,
		},
		{
			name:     "non-numeric",
			timeStr:  "aa:bb:cc",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := timeToSeconds(tt.timeStr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
