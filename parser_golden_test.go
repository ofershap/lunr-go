package lunr

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

type jsClause struct {
	Term         string   `json:"term"`
	Fields       []string `json:"fields"`
	Boost        float64  `json:"boost"`
	EditDistance *float64 `json:"editDistance"`
	UsePipeline  bool     `json:"usePipeline"`
	Presence     int      `json:"presence"`
}

type jsParserFixture struct {
	Query   string     `json:"query"`
	Clauses []jsClause `json:"clauses"`
	Error   string     `json:"error"`
}

// TestQueryParserParity: 250 seeded random query strings, clause-by-clause.
func TestQueryParserParity(t *testing.T) {
	data, err := os.ReadFile("testdata/index_from_js.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Parser []jsParserFixture `json:"parser"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Parser) < 200 {
		t.Fatalf("too few parser fixtures: %d", len(f.Parser))
	}
	for _, pf := range f.Parser {
		q := NewQuery([]string{"title", "body"})
		_, err := ParseQuery(pf.Query, q)
		if pf.Error != "" {
			if err == nil {
				t.Errorf("query %q: JS errored %q, Go parsed %+v", pf.Query, pf.Error, q.Clauses)
			} else if err.Error() != pf.Error {
				t.Errorf("query %q: Go error %q, JS error %q", pf.Query, err.Error(), pf.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("query %q: Go errored %v, JS ok", pf.Query, err)
			continue
		}
		if len(q.Clauses) != len(pf.Clauses) {
			t.Errorf("query %q: Go %d clauses, JS %d", pf.Query, len(q.Clauses), len(pf.Clauses))
			continue
		}
		for i, want := range pf.Clauses {
			got := q.Clauses[i]
			if got.Term != want.Term {
				t.Errorf("query %q clause %d: term %q, want %q", pf.Query, i, got.Term, want.Term)
			}
			if !strsEqual(got.Fields, want.Fields) {
				t.Errorf("query %q clause %d: fields %v, want %v", pf.Query, i, got.Fields, want.Fields)
			}
			if math.Float64bits(got.Boost) != math.Float64bits(want.Boost) {
				t.Errorf("query %q clause %d: boost %v, want %v", pf.Query, i, got.Boost, want.Boost)
			}
			if (want.EditDistance != nil) != got.HasEditDistance ||
				(want.EditDistance != nil && int(*want.EditDistance) != got.EditDistance) {
				t.Errorf("query %q clause %d: editDistance %v (has %v), want %v",
					pf.Query, i, got.EditDistance, got.HasEditDistance, want.EditDistance)
			}
			if got.UsePipeline != want.UsePipeline {
				t.Errorf("query %q clause %d: usePipeline %v, want %v", pf.Query, i, got.UsePipeline, want.UsePipeline)
			}
			if got.Presence != want.Presence {
				t.Errorf("query %q clause %d: presence %v, want %v", pf.Query, i, got.Presence, want.Presence)
			}
		}
	}
}

func strsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
