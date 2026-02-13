package node

import "fmt"

// ValidationError is returned when request input is invalid.
type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func invalidf(format string, args ...any) error {
	return &ValidationError{
		message: fmt.Sprintf(format, args...),
	}
}
