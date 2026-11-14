// JavaScript string fidelity helpers shared with the reference engine:
// lunr.js operates on JS strings (UTF-16 code units), lowercases with
// String.prototype.toLowerCase, and reports token positions in code units.
// lunr-go does the same so positions and scores match bit-exactly.

package lunr

import (
	"strings"
	"unicode"
	"unicode/utf16"
)

func toU16(s string) []uint16 {
	return utf16.Encode([]rune(s))
}

func fromU16(u []uint16) string {
	return string(utf16.Decode(u))
}

// jsLowercaseString mirrors JS String.prototype.toLowerCase: full Unicode
// case mapping, which differs from Go's simple per-rune mapping for a small
// set of runes (see jsLowerDelta, generated from V8).
func jsLowercaseString(s string) string {
	rs := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range rs {
		if mapped, ok := jsLowerDelta[r]; ok {
			b.WriteString(mapped)
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
