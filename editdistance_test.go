// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func newFuzzyLimitTestIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := New(func(b *Builder) {
		b.Ref("id")
		_ = b.Field("title", FieldAttributes{})
		_ = b.Field("body", FieldAttributes{})
		b.Add(map[string]any{"id": "1", "title": "Go programming", "body": "Search and index everything."}, 0)
		b.Add(map[string]any{"id": "2", "title": "JavaScript", "body": "Client-side search with lunr."}, 0)
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// Regression: a query string with an absurd edit distance (~99999999) must
// fail fast with a clear error instead of hanging. lunr.js 2.3.9 accepts the
// same query and hangs on it (verified against Node 22); this port diverges
// intentionally and rejects distances above MaxEditDistance.
func TestSearchRejectsAbsurdEditDistance(t *testing.T) {
	idx := newFuzzyLimitTestIndex(t)
	start := time.Now()
	_, err := idx.Search("go~99999999")
	if err == nil {
		t.Fatal("expected an error for edit distance 99999999")
	}
	var fl *FuzzyLimitError
	if !errors.As(err, &fl) {
		t.Fatalf("expected *FuzzyLimitError, got %T: %v", err, err)
	}
	if fl.Distance != 99999999 || fl.MaxDistance != MaxEditDistance {
		t.Fatalf("unexpected error fields: %+v", fl)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("rejection took %v, expected fail-fast", elapsed)
	}
}

// The guard applies to programmatically built queries too, not only to the
// query-string parser.
func TestQueryRejectsAbsurdEditDistance(t *testing.T) {
	idx := newFuzzyLimitTestIndex(t)
	_, err := idx.Query(func(q *Query) {
		q.Term([]string{"go"}, WithEditDistance(MaxEditDistance+1))
	})
	var fl *FuzzyLimitError
	if !errors.As(err, &fl) {
		t.Fatalf("expected *FuzzyLimitError, got %T: %v", err, err)
	}
}

// Fuzzy token-set cost grows roughly with term-length^distance (measured:
// distance 3 on a 100-character term takes ~14s on the 2,000-document bench
// corpus), so long fuzzy terms are rejected even at small distances. The
// length bound uses UTF-16 code units, JavaScript string semantics.
func TestSearchRejectsLongFuzzyTerm(t *testing.T) {
	idx := newFuzzyLimitTestIndex(t)
	term := strings.Repeat("a", MaxFuzzyTermLength) + "b"
	_, err := idx.Search(term + "~1")
	var fl *FuzzyLimitError
	if !errors.As(err, &fl) {
		t.Fatalf("expected *FuzzyLimitError, got %T: %v", err, err)
	}
	if fl.TermLength != MaxFuzzyTermLength+1 {
		t.Fatalf("unexpected error fields: %+v", fl)
	}
}

// A long term without fuzzy matching is unaffected by the bound.
func TestSearchAllowsLongExactTerm(t *testing.T) {
	idx := newFuzzyLimitTestIndex(t)
	term := strings.Repeat("a", 200)
	if _, err := idx.Search(term); err != nil {
		t.Fatalf("long exact term should be accepted: %v", err)
	}
}

// Distances and term lengths inside the bounds keep lunr.js behaviour.
func TestSearchAllowsBoundedFuzzy(t *testing.T) {
	idx := newFuzzyLimitTestIndex(t)
	if _, err := idx.Search("search~2"); err != nil {
		t.Fatalf("edit distance 2 should be accepted: %v", err)
	}
	if _, err := idx.Search("search~3"); err != nil {
		t.Fatalf("edit distance 3 should be accepted: %v", err)
	}
}
