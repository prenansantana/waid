package waid

import "fmt"

// WAIDError represents an error returned by the WAID API.
type WAIDError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *WAIDError) Error() string {
	return fmt.Sprintf("waid: HTTP %d: %s", e.StatusCode, e.Message)
}
