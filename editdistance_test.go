// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"errors"
	"testing"
	"time"
)

func newEditDistanceTestIndex(t *testing.T) *Index {
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
	idx := newEditDistanceTestIndex(t)
	start := time.Now()
	_, err := idx.Search("go~99999999")
	if err == nil {
		t.Fatal("expected an error for edit distance 99999999")
	}
	var ed *EditDistanceError
	if !errors.As(err, &ed) {
		t.Fatalf("expected *EditDistanceError, got %T: %v", err, err)
	}
	if ed.Distance != 99999999 || ed.Max != MaxEditDistance {
		t.Fatalf("unexpected error fields: %+v", ed)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("rejection took %v, expected fail-fast", elapsed)
	}
}

// The guard applies to programmatically built queries too, not only to the
// query-string parser.
func TestQueryRejectsAbsurdEditDistance(t *testing.T) {
	idx := newEditDistanceTestIndex(t)
	_, err := idx.Query(func(q *Query) {
		q.Term([]string{"go"}, WithEditDistance(MaxEditDistance+1))
	})
	var ed *EditDistanceError
	if !errors.As(err, &ed) {
		t.Fatalf("expected *EditDistanceError, got %T: %v", err, err)
	}
}

// Distances at or below the bound keep lunr.js behaviour.
func TestSearchAllowsBoundedEditDistance(t *testing.T) {
	idx := newEditDistanceTestIndex(t)
	if _, err := idx.Search("search~2"); err != nil {
		t.Fatalf("edit distance 2 should be accepted: %v", err)
	}
}
