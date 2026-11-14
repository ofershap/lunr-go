package lunr

import "fmt"

// ParseError mirrors lunr.QueryParseError (and is reused for malformed
// field refs and serialized-index problems).
type ParseError struct {
	Message string
	Start   int
	End     int
	Name    string
}

func (e *ParseError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("%s: %s", e.Name, e.Message)
	}
	return e.Message
}
