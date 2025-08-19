package probe

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Interval represents a time interval that can be specified as a number (seconds) or duration string
type Interval struct {
	time.Duration
}

// UnmarshalYAML implements custom YAML unmarshaling for Interval
func (i *Interval) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case int:
		// If it's an integer, treat as seconds
		i.Duration = time.Duration(v) * time.Second
	case float64:
		// YAML might parse numbers as float64
		i.Duration = time.Duration(int(v)) * time.Second
	case uint64:
		// YAML might parse numbers as uint64
		i.Duration = time.Duration(v) * time.Second
	case string:
		// If it's a string, try to parse as duration
		// First check if it's a plain number (backward compatibility)
		if matched, _ := regexp.MatchString(`^\d+$`, v); matched {
			// Plain number, treat as seconds
			if seconds, err := strconv.Atoi(v); err == nil {
				i.Duration = time.Duration(seconds) * time.Second
			} else {
				return fmt.Errorf("invalid interval: %s", v)
			}
		} else {
			// Parse as duration string
			if dur, err := time.ParseDuration(v); err == nil {
				i.Duration = dur
			} else {
				return fmt.Errorf("invalid interval format: %s", v)
			}
		}
	default:
		return fmt.Errorf("interval must be a number or duration string, got %T: %v", v, v)
	}

	return nil
}

// MarshalYAML implements custom YAML marshaling for Interval
func (i Interval) MarshalYAML() (interface{}, error) {
	// If it's a whole number of seconds, return as integer
	if i.Duration%time.Second == 0 {
		return int(i.Duration / time.Second), nil
	}
	// Otherwise, return as duration string
	return i.String(), nil
}

// Repeat defines the repeat configuration for jobs
type Repeat struct {
	Count    int      `yaml:"count" validate:"required,gte=0,lt=100"`
	Interval Interval `yaml:"interval"`
}

// StepRetry defines the retry configuration for steps until success (status 0)
type StepRetry struct {
	MaxAttempts  int      `yaml:"max_attempts" validate:"required,gte=1,lte=100"`
	Interval     Interval `yaml:"interval"`
	InitialDelay Interval `yaml:"initial_delay,omitempty"`
}
