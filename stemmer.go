package lunr

// Porter stemmer, ported from lunr.stemmer (lunr.js 2.3.9), which is Seth
// Emont's JavaScript port of the classic algorithm. The original drives every
// step with JS regexes over UTF-16 code units; this port evaluates the same
// patterns over code units with a small backtracking matcher, so edge cases
// (initial y, doubled consonants, non-BMP input) resolve exactly as V8 does.

func isAeiou(c uint16) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}

func isAeiouy(c uint16) bool { return isAeiou(c) || c == 'y' }

func notAeiou(c uint16) bool  { return !isAeiou(c) }
func notAeiouy(c uint16) bool { return !isAeiouy(c) }
func notAeiouwxy(c uint16) bool {
	return !(isAeiou(c) || c == 'w' || c == 'x' || c == 'y')
}

// relm is one regex element: a char class with a repetition range.
type relm struct {
	cls      func(uint16) bool
	min, max int // max < 0 means unbounded
}

// matchSeq reports whether elems matches s anchored at the start
// (and at the end when endAnchored), with greedy backtracking like JS regex.
func matchSeq(s []uint16, elems []relm, endAnchored bool) bool {
	var rec func(pos, ei int) bool
	rec = func(pos, ei int) bool {
		if ei == len(elems) {
			if endAnchored {
				return pos == len(s)
			}
			return true
		}
		e := elems[ei]
		maxTake := 0
		for pos+maxTake < len(s) && e.cls(s[pos+maxTake]) && (e.max < 0 || maxTake < e.max) {
			maxTake++
		}
		for take := maxTake; take >= e.min; take-- {
			if rec(pos+take, ei+1) {
				return true
			}
		}
		return false
	}
	return rec(0, 0)
}

var (
	reC = []relm{{notAeiou, 1, 1}, {notAeiouy, 0, -1}} // C = [^aeiou][^aeiouy]*
	reV = []relm{{isAeiouy, 1, 1}, {isAeiou, 0, -1}}   // V = [aeiouy][aeiou]*
)

// catC returns patterns with an optional leading C (JS "(C)?" prefix).
func withOptC(rest ...relm) [][]relm {
	with := append(append([]relm{}, reC...), rest...)
	without := append([]relm{}, rest...)
	return [][]relm{with, without}
}

func anyMatch(s []uint16, patterns [][]relm, endAnchored bool) bool {
	for _, p := range patterns {
		if matchSeq(s, p, endAnchored) {
			return true
		}
	}
	return false
}

// mgr0: ^(C)?VC   - measure > 0
func mgr0(s []uint16) bool {
	return anyMatch(s, withOptC(append(append([]relm{}, reV...), reC...)...), false)
}

// mgr1: ^(C)?VCVC - measure > 1
func mgr1(s []uint16) bool {
	vcvc := append(append(append(append([]relm{}, reV...), reC...), reV...), reC...)
	return anyMatch(s, withOptC(vcvc...), false)
}

// meq1: ^(C)?VC(V)?$ - measure = 1 (the trailing V is optional)
func meq1(s []uint16) bool {
	vc := append(append([]relm{}, reV...), reC...)
	vcv := append(append([]relm{}, vc...), reV...)
	return anyMatch(s, append(withOptC(vc...), withOptC(vcv...)...), true)
}

// hasVowelRe: ^(C)?v - vowel in stem
func hasVowelRe(s []uint16) bool {
	return anyMatch(s, withOptC(relm{isAeiouy, 1, 1}), false)
}

// cvcNoWXY: ^C v [^aeiouwxy]$ (re4_1b_2 / re3_5)
func cvcNoWXY(s []uint16) bool {
	elems := append(append(append([]relm{}, reC...), relm{isAeiouy, 1, 1}), relm{notAeiouwxy, 1, 1})
	return matchSeq(s, elems, true)
}

// doubledConsNotLSZ: ([^aeiouylsz])\1$ - ends in a doubled consonant other
// than l, s or z.
func doubledConsNotLSZ(u []uint16) bool {
	n := len(u)
	if n < 2 || u[n-1] != u[n-2] {
		return false
	}
	c := u[n-1]
	return !(isAeiou(c) || c == 'y' || c == 'l' || c == 's' || c == 'z')
}

func hasSuffixU16(u []uint16, suffix string) bool {
	n := len(suffix)
	if len(u) < n {
		return false
	}
	for i := 0; i < n; i++ {
		if u[len(u)-n+i] != uint16(suffix[i]) {
			return false
		}
	}
	return true
}

var step2list = []struct{ suffix, repl string }{
	{"ational", "ate"}, {"tional", "tion"}, {"enci", "ence"}, {"anci", "ance"},
	{"izer", "ize"}, {"bli", "ble"}, {"alli", "al"}, {"entli", "ent"},
	{"eli", "e"}, {"ousli", "ous"}, {"ization", "ize"}, {"ation", "ate"},
	{"ator", "ate"}, {"alism", "al"}, {"iveness", "ive"}, {"fulness", "ful"},
	{"ousness", "ous"}, {"aliti", "al"}, {"iviti", "ive"}, {"biliti", "ble"},
	{"logi", "log"},
}

var step3list = []struct{ suffix, repl string }{
	{"icate", "ic"}, {"ative", ""}, {"alize", "al"}, {"iciti", "ic"},
	{"ical", "ic"}, {"ful", ""}, {"ness", ""},
}

var step4list = []string{
	"al", "ance", "ence", "er", "ic", "able", "ible", "ant", "ement", "ment",
	"ent", "ou", "ism", "ate", "iti", "ous", "ive", "ize",
}

// suffixScan mirrors ^(.+?)(alt1|alt2|...)$ with a lazy stem: stems grow
// from one code unit; at each stem length the alternatives are tried in
// list order. Returns the stem and the winning alternative.
func suffixScan(u []uint16, alts []string) (stem []uint16, alt string, ok bool) {
	for stemLen := 1; stemLen < len(u); stemLen++ {
		tailLen := len(u) - stemLen
		for _, a := range alts {
			if len(a) == tailLen && hasSuffixU16(u, a) {
				return u[:stemLen], a, true
			}
		}
	}
	return nil, "", false
}

func porterStem(w string) string {
	u := toU16(w)
	if len(u) < 3 {
		return w
	}

	firstY := u[0] == 'y'
	if firstY {
		u[0] = 'Y'
	}

	// Step 1a
	if stem, _, ok := suffixScan(u, []string{"sses", "ies"}); ok {
		u = u[:len(u)-2]
		_ = stem
	} else if n := len(u); n >= 3 && u[n-1] == 's' && u[n-2] != 's' {
		u = u[:n-1]
	}

	// Step 1b
	if len(u) >= 4 && hasSuffixU16(u, "eed") {
		stem := u[:len(u)-3]
		if mgr0(stem) {
			u = u[:len(u)-1]
		}
	} else if stem, _, ok := suffixScan(u, []string{"ed", "ing"}); ok {
		if hasVowelRe(stem) {
			u = stem
			if hasSuffixU16(u, "at") || hasSuffixU16(u, "bl") || hasSuffixU16(u, "iz") {
				u = append(u, 'e')
			} else if doubledConsNotLSZ(u) {
				u = u[:len(u)-1]
			} else if cvcNoWXY(u) {
				u = append(u, 'e')
			}
		}
	}

	// Step 1c
	if n := len(u); n >= 3 && u[n-1] == 'y' && !isAeiou(u[n-2]) {
		u[n-1] = 'i'
	}

	// Step 2
	if stem, alt, ok := suffixScan(u, step2suffixes()); ok {
		if mgr0(stem) {
			u = append(stem, step2repl(alt)...)
		}
	}

	// Step 3
	if stem, alt, ok := suffixScan(u, step3suffixes()); ok {
		if mgr0(stem) {
			u = append(stem, step3repl(alt)...)
		}
	}

	// Step 4
	if stem, _, ok := suffixScan(u, step4list); ok {
		if mgr1(stem) {
			u = stem
		}
	} else if n := len(u); n >= 5 && (hasSuffixU16(u, "sion") || hasSuffixU16(u, "tion")) {
		stem := u[:n-3]
		if mgr1(stem) {
			u = stem
		}
	}

	// Step 5
	if n := len(u); n >= 2 && u[n-1] == 'e' {
		stem := u[:n-1]
		if mgr1(stem) || (meq1(stem) && !cvcNoWXY(stem)) {
			u = stem
		}
	}

	if n := len(u); n >= 2 && u[n-1] == 'l' && u[n-2] == 'l' && mgr1(u) {
		u = u[:n-1]
	}

	if firstY {
		u[0] = 'y'
	}
	return fromU16(u)
}

func step2suffixes() []string {
	out := make([]string, len(step2list))
	for i, e := range step2list {
		out[i] = e.suffix
	}
	return out
}

func step2repl(suffix string) []uint16 {
	for _, e := range step2list {
		if e.suffix == suffix {
			return toU16(e.repl)
		}
	}
	return nil
}

func step3suffixes() []string {
	out := make([]string, len(step3list))
	for i, e := range step3list {
		out[i] = e.suffix
	}
	return out
}

func step3repl(suffix string) []uint16 {
	for _, e := range step3list {
		if e.suffix == suffix {
			return toU16(e.repl)
		}
	}
	return nil
}
