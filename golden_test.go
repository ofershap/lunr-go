package lunr

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

type jsResult struct {
	Ref       string                                 `json:"ref"`
	Score     float64                                `json:"score"`
	MatchData map[string]map[string]map[string][]any `json:"matchData"`
}

type jsQueryResult struct {
	Query   string     `json:"query"`
	Results []jsResult `json:"results"`
	Error   string     `json:"error"`
}

type jsFixture struct {
	Docs       []map[string]any `json:"docs"`
	Serialized string           `json:"serialized"`
	Results    []jsQueryResult  `json:"results"`
}

func loadFixture(t *testing.T) jsFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/index_from_js.json")
	if err != nil {
		t.Fatal(err)
	}
	var f jsFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func buildFromFixture(t *testing.T, f jsFixture) *Index {
	t.Helper()
	idx, err := New(func(b *Builder) {
		b.Ref("id")
		if err := b.Field("title", FieldAttributes{Boost: 10}); err != nil {
			t.Fatal(err)
		}
		if err := b.Field("body", FieldAttributes{}); err != nil {
			t.Fatal(err)
		}
		b.MetadataWhitelist = []string{"position"}
		for _, doc := range f.Docs {
			b.Add(doc, 0)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// TestSerializeByteIdentical: the Go-built index must serialize byte-for-byte
// like lunr.js JSON.stringify(idx) over the same corpus.
func TestSerializeByteIdentical(t *testing.T) {
	f := loadFixture(t)
	idx := buildFromFixture(t, f)
	got := idx.Serialize()
	if got != f.Serialized {
		n := len(got)
		if len(f.Serialized) < n {
			n = len(f.Serialized)
		}
		for i := 0; i < n; i++ {
			if got[i] != f.Serialized[i] {
				lo := i - 60
				if lo < 0 {
					lo = 0
				}
				t.Fatalf("first diff at byte %d\ngo: %q\njs: %q", i, got[lo:i+40], f.Serialized[lo:i+40])
			}
		}
		t.Fatalf("lengths differ: go %d, js %d", len(got), len(f.Serialized))
	}
}

// TestQueryParity: refs, order and score bit patterns must match lunr.js.
func TestQueryParity(t *testing.T) {
	f := loadFixture(t)
	idx := buildFromFixture(t, f)
	for _, qr := range f.Results {
		got, err := idx.Search(qr.Query)
		if qr.Error != "" {
			if err == nil {
				t.Errorf("query %q: JS errored %q, Go succeeded", qr.Query, qr.Error)
			} else if err.Error() != qr.Error {
				t.Errorf("query %q: Go error %q, JS error %q", qr.Query, err.Error(), qr.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("query %q: Go errored %v, JS ok", qr.Query, err)
			continue
		}
		if len(got) != len(qr.Results) {
			t.Errorf("query %q: Go %d results, JS %d", qr.Query, len(got), len(qr.Results))
			continue
		}
		for i, want := range qr.Results {
			if got[i].Ref != want.Ref {
				t.Errorf("query %q result %d: ref %q, want %q", qr.Query, i, got[i].Ref, want.Ref)
			}
			if math.Float64bits(got[i].Score) != math.Float64bits(want.Score) {
				t.Errorf("query %q result %d: score %v (bits %x), want %v (bits %x)",
					qr.Query, i, got[i].Score, math.Float64bits(got[i].Score), want.Score, math.Float64bits(want.Score))
			}
			if !matchDataEqual(got[i].MatchData, want.MatchData) {
				t.Errorf("query %q result %d: matchData mismatch\ngo: %v\njs: %v", qr.Query, i, got[i].MatchData, want.MatchData)
			}
		}
	}
}

// matchDataEqual compares metadata structures ignoring JSON key order.
// JS position arrays decode as []any of float64 pairs; Go keeps [2]int.
func matchDataEqual(got *MatchData, want map[string]map[string]map[string][]any) bool {
	gj := canonicalMatchData(got.Metadata)
	wj := canonicalMap3(want)
	if len(gj) != len(wj) {
		return false
	}
	for term, fields := range wj {
		gf, ok := gj[term]
		if !ok {
			return false
		}
		for field, keys := range fields {
			gk, ok := gf[field]
			if !ok {
				return false
			}
			for k, vals := range keys {
				gv, ok := gk[k]
				if !ok || !valsEqual(gv, vals) {
					return false
				}
			}
		}
	}
	return true
}

func canonicalMatchData(m map[string]map[string]map[string][]any) map[string]map[string]map[string][]any {
	return canonicalMap3(m)
}

func canonicalMap3(m map[string]map[string]map[string][]any) map[string]map[string]map[string][]any {
	out := map[string]map[string]map[string][]any{}
	for term, fields := range m {
		out[term] = map[string]map[string][]any{}
		for field, keys := range fields {
			out[term][field] = map[string][]any{}
			for k, vals := range keys {
				cv := make([]any, len(vals))
				for i, v := range vals {
					cv[i] = canonicalVal(v)
				}
				out[term][field][k] = cv
			}
		}
	}
	return out
}

func canonicalVal(v any) any {
	switch t := v.(type) {
	case [2]int:
		return []any{float64(t[0]), float64(t[1])}
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = canonicalVal(e)
		}
		return out
	case []string:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = e
		}
		return out
	default:
		return v
	}
}

func valsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		aj, _ := json.Marshal(canonicalVal(a[i]))
		bj, _ := json.Marshal(canonicalVal(b[i]))
		if string(aj) != string(bj) {
			return false
		}
	}
	return true
}

// TestLoadRoundTrip: a lunr.js-serialized index loads into Go and
// re-serializes byte-identically.
func TestLoadRoundTrip(t *testing.T) {
	f := loadFixture(t)
	idx, err := LoadIndex(f.Serialized)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.Serialize(); got != f.Serialized {
		t.Fatalf("round trip mismatch: got %d bytes, want %d", len(got), len(f.Serialized))
	}
}

// TestLoadedIndexQueryParity: queries against the loaded JS index behave
// like queries against the Go-built one.
func TestLoadedIndexQueryParity(t *testing.T) {
	f := loadFixture(t)
	built := buildFromFixture(t, f)
	loaded, err := LoadIndex(f.Serialized)
	if err != nil {
		t.Fatal(err)
	}
	for _, qr := range f.Results {
		a, errA := built.Search(qr.Query)
		b, errB := loaded.Search(qr.Query)
		if (errA == nil) != (errB == nil) {
			t.Errorf("query %q: error parity differs", qr.Query)
			continue
		}
		if len(a) != len(b) {
			t.Errorf("query %q: built %d results, loaded %d", qr.Query, len(a), len(b))
			continue
		}
		for i := range a {
			if a[i].Ref != b[i].Ref || math.Float64bits(a[i].Score) != math.Float64bits(b[i].Score) {
				t.Errorf("query %q result %d differs between built and loaded", qr.Query, i)
			}
		}
	}
}
