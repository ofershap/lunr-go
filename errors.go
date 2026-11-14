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

// MaxEditDistance bounds fuzzy-matching edit distance. lunr.js accepts any
// distance and builds a token set whose size grows with it, so absurd values
// (for example ~99999999) hang for a very long time; lunr's own docs advise
// distances below 3. This port rejects distances above MaxEditDistance with a
// clear error instead of hanging: an intentional, documented divergence from
// lunr.js behaviour.
const MaxEditDistance = 10

// EditDistanceError is returned by Index.Query and Index.Search when a clause
// requests a fuzzy edit distance above MaxEditDistance.
type EditDistanceError struct {
	Distance int
	Max      int
}

func (e *EditDistanceError) Error() string {
	return fmt.Sprintf("edit distance %d exceeds the supported maximum of %d", e.Distance, e.Max)
}
