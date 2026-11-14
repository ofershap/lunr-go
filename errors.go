// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

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
