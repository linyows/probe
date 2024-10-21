package probe

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	messages []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error:\n%s", strings.Join(e.messages, "\n"))
}

func (e *ValidationError) HasError() bool {
	return len(e.messages) > 0
}

func (e *ValidationError) AddMessage(s string) {
	e.messages = append(e.messages, s)
}
