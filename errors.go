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

// Fuzzy matching bounds. lunr.js accepts any edit distance on any term and
// builds a token set whose size grows roughly with term-length^distance, so
// absurd values (for example ~99999999) hang for a very long time, and even
// moderate combinations get expensive (measured on the 2,000-document bench
// corpus: distance 4 on a 44-character term takes ~15s). lunr's own docs
// advise distances below 3. This port rejects fuzzy clauses beyond the bounds
// below with a clear error instead of hanging: an intentional, documented
// divergence from lunr.js behaviour. Worst measured case inside the bounds
// (distance 3 on a 32-character term): ~150ms.
const (
	// MaxEditDistance is the largest accepted fuzzy edit distance.
	MaxEditDistance = 3
	// MaxFuzzyTermLength is the largest term length, in UTF-16 code units
	// (JavaScript string semantics), accepted for fuzzy matching.
	MaxFuzzyTermLength = 32
)

// FuzzyLimitError is returned by Index.Query and Index.Search when a fuzzy
// clause exceeds MaxEditDistance or MaxFuzzyTermLength.
type FuzzyLimitError struct {
	Distance      int
	TermLength    int
	MaxDistance   int
	MaxTermLength int
}

func (e *FuzzyLimitError) Error() string {
	if e.Distance > e.MaxDistance {
		return fmt.Sprintf("edit distance %d exceeds the supported maximum of %d", e.Distance, e.MaxDistance)
	}
	return fmt.Sprintf("fuzzy term length %d exceeds the supported maximum of %d", e.TermLength, e.MaxTermLength)
}
