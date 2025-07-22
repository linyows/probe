package probe

import (
	"fmt"
	"testing"
	"time"
)

// Test Interval YAML unmarshaling
func TestIntervalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected time.Duration
		hasError bool
	}{
		{
			name:     "integer seconds",
			input:    10,
			expected: 10 * time.Second,
			hasError: false,
		},
		{
			name:     "uint64 seconds", 
			input:    uint64(5),
			expected: 5 * time.Second,
			hasError: false,
		},
		{
			name:     "float64 seconds",
			input:    float64(3),
			expected: 3 * time.Second,
			hasError: false,
		},
		{
			name:     "duration string",
			input:    "2m30s",
			expected: 2*time.Minute + 30*time.Second,
			hasError: false,
		},
		{
			name:     "milliseconds string",
			input:    "500ms", 
			expected: 500 * time.Millisecond,
			hasError: false,
		},
		{
			name:     "plain number string",
			input:    "15",
			expected: 15 * time.Second,
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid type",
			input:    []string{"invalid"},
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var interval Interval
			
			unmarshalFunc := func(v interface{}) error {
				// Simulate the unmarshal behavior
				if vPtr, ok := v.(*interface{}); ok {
					*vPtr = tt.input
					return nil
				}
				return fmt.Errorf("invalid unmarshal target")
			}
			
			err := interval.UnmarshalYAML(unmarshalFunc)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if interval.Duration != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, interval.Duration)
			}
		})
	}
}

// Test Interval YAML marshaling
func TestIntervalMarshal(t *testing.T) {
	tests := []struct {
		name     string
		interval Interval
		expected interface{}
	}{
		{
			name:     "whole seconds",
			interval: Interval{Duration: 10 * time.Second},
			expected: 10,
		},
		{
			name:     "milliseconds",
			interval: Interval{Duration: 500 * time.Millisecond},
			expected: "500ms",
		},
		{
			name:     "complex duration", 
			interval: Interval{Duration: 2*time.Minute + 30*time.Second},
			expected: 150, // 2m30s = 150 seconds, will be marshaled as integer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.interval.MarshalYAML()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}