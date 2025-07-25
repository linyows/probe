package probe

import (
	"testing"
	"time"
)

func TestStep_parseWaitDuration(t *testing.T) {
	tests := []struct {
		name     string
		wait     string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "seconds as integer string",
			wait:     "5",
			expected: 5 * time.Second,
			hasError: false,
		},
		{
			name:     "duration string with seconds",
			wait:     "3s",
			expected: 3 * time.Second,
			hasError: false,
		},
		{
			name:     "duration string with milliseconds",
			wait:     "500ms",
			expected: 500 * time.Millisecond,
			hasError: false,
		},
		{
			name:     "duration string with minutes",
			wait:     "2m",
			expected: 2 * time.Minute,
			hasError: false,
		},
		{
			name:     "invalid format",
			wait:     "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty string",
			wait:     "",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{}
			duration, err := step.parseWaitDuration(tt.wait)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for wait '%s', but got none", tt.wait)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for wait '%s': %v", tt.wait, err)
				}
				if duration != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, duration)
				}
			}
		})
	}
}

func TestStep_formatWaitTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "exact seconds",
			duration: 5 * time.Second,
			expected: "5s",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "mixed duration",
			duration: 2*time.Minute + 30*time.Second,
			expected: "150s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{}
			result := step.formatWaitTime(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStep_getWaitTimeForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		wait     string
		expected string
	}{
		{
			name:     "no wait",
			wait:     "",
			expected: "",
		},
		{
			name:     "seconds",
			wait:     "5",
			expected: "5s",
		},
		{
			name:     "duration string",
			wait:     "500ms",
			expected: "500ms",
		},
		{
			name:     "invalid format returns empty",
			wait:     "invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{Wait: tt.wait}
			result := step.getWaitTimeForDisplay()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStep_handleWait(t *testing.T) {
	tests := []struct {
		name         string
		wait         string
		expectedTime string
		minDuration  time.Duration
		maxDuration  time.Duration
	}{
		{
			name:         "no wait",
			wait:         "",
			expectedTime: "",
			minDuration:  0,
			maxDuration:  10 * time.Millisecond,
		},
		{
			name:         "short wait",
			wait:         "10ms",
			expectedTime: "10ms",
			minDuration:  8 * time.Millisecond,
			maxDuration:  50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{Wait: tt.wait}
			jCtx := &JobContext{
				Output: &Output{},
			}

			start := time.Now()
			result := step.handleWait(jCtx)
			duration := time.Since(start)

			if result != tt.expectedTime {
				t.Errorf("expected %s, got %s", tt.expectedTime, result)
			}

			if duration < tt.minDuration {
				t.Errorf("Duration %v should be at least %v", duration, tt.minDuration)
			}

			if duration > tt.maxDuration {
				t.Errorf("Duration %v should not exceed %v", duration, tt.maxDuration)
			}
		})
	}
}