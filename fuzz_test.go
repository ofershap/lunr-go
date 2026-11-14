package lunr

import (
	"testing"
	"unicode/utf8"
)

// FuzzTokenize: never panics; emitted positions are valid UTF-16 code-unit
// ranges of the lowercased input.
func FuzzTokenize(f *testing.F) {
	f.Add("hello world")
	f.Add("café résumé 🚀")
	f.Add("state-of-the-art  search")
	f.Add("")
	f.Add("---")
	f.Add("a  b\t c\n")
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) {
			return
		}
		tokens := Tokenize(s, map[string]any{"fields": []string{"body"}})
		lowerU16 := len(toU16(jsLowercaseString(s)))
		for _, tok := range tokens {
			pos, ok := tok.Metadata["position"].([2]int)
			if !ok {
				t.Fatalf("token %q missing position metadata", tok.Str)
			}
			if pos[0] < 0 || pos[1] < 0 || pos[0]+pos[1] > lowerU16 {
				t.Fatalf("token %q position %v out of range %d", tok.Str, pos, lowerU16)
			}
			if fromU16(toU16(jsLowercaseString(s))[pos[0]:pos[0]+pos[1]]) != tok.Str {
				t.Fatalf("token %q does not match sliced input at %v", tok.Str, pos)
			}
		}
	})
}

// FuzzQueryParser: never panics on arbitrary query strings.
func FuzzQueryParser(f *testing.F) {
	f.Add("title:go^10 ~2 +required -prohibited *wild*")
	f.Add("\\\\~ \\\\: foo~bar ^")
	f.Add("+ - : ~ ^ *")
	f.Add("🚀 café")
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) || len(s) > 4096 {
			return
		}
		_, _ = ParseQuery(s, NewQuery([]string{"title", "body"}))
	})
}

// FuzzSearch: never panics searching a fixed index with arbitrary strings.
func FuzzSearch(f *testing.F) {
	f.Add("go")
	f.Add("title:go^10")
	f.Add("*")
	f.Add("~99")
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) || len(s) > 1024 {
			return
		}
		// Skip edit distances beyond lunr's documented guidance (< 3):
		// lunr.js and this port both build a token set whose size grows
		// with the distance, so absurd values like ~99999999 hang by
		// design (verified against node). Not a port defect.
		if hasLargeEditDistance(s, 3) {
			return
		}
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
		_, _ = idx.Search(s)
	})
}

// hasLargeEditDistance reports whether s contains a ~N modifier with N
// greater than max.
func hasLargeEditDistance(s string, max int) bool {
	u := toU16(s)
	for i := 0; i < len(u); i++ {
		if u[i] != '~' {
			continue
		}
		j := i + 1
		n := 0
		for j < len(u) && u[j] >= '0' && u[j] <= '9' {
			n = n*10 + int(u[j]-'0')
			if n > max {
				return true
			}
			j++
		}
	}
	return false
}
