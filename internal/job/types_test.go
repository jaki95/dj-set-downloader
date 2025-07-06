package job

import (
	"testing"
)

func TestValidateMaxConcurrentTasks(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "zero value returns default",
			input:    0,
			expected: DefaultMaxConcurrentTasks,
		},
		{
			name:     "negative value returns default",
			input:    -1,
			expected: DefaultMaxConcurrentTasks,
		},
		{
			name:     "valid value within range",
			input:    10,
			expected: 10,
		},
		{
			name:     "value at max limit",
			input:    MaxAllowedConcurrentTasks,
			expected: MaxAllowedConcurrentTasks,
		},
		{
			name:     "excessive value is capped",
			input:    1000,
			expected: MaxAllowedConcurrentTasks,
		},
		{
			name:     "extremely large value is capped",
			input:    1000000,
			expected: MaxAllowedConcurrentTasks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMaxConcurrentTasks(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateMaxConcurrentTasks(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}