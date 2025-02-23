package audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeToSeconds(t *testing.T) {
	testCases := []struct {
		name           string
		timestamp      string
		expectedErr    string
		expectedOutput float64
	}{
		{
			name:           "minute timestamp",
			timestamp:      "45:23",
			expectedErr:    "",
			expectedOutput: 45*60 + 23,
		},
		{
			name:           "hour timestamp",
			timestamp:      "3:55:33",
			expectedErr:    "",
			expectedOutput: 3*3600 + 55*60 + 33,
		},
		{
			name:           "invalid parts",
			timestamp:      "1:03:55:33",
			expectedErr:    "invalid timestamp",
			expectedOutput: 0,
		},
		{
			name:           "invalid timestamp",
			timestamp:      "03-55-33",
			expectedErr:    "invalid timestamp",
			expectedOutput: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seconds, err := timeToSeconds(tc.timestamp)
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedOutput, seconds)

		})
	}
}
