// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"strings"
)

// QueryParseError mirrors lunr.QueryParseError.
type QueryParseError struct {
	Message    string
	Start, End int
}

func (e *QueryParseError) Error() string { return e.Message }

// Lexer token types mirror lunr.QueryLexer constants.
const (
	lexEOS          = "EOS"
	lexField        = "FIELD"
	lexTerm         = "TERM"
	lexEditDistance = "EDIT_DISTANCE"
	lexBoost        = "BOOST"
	lexPresence     = "PRESENCE"
)

type lexeme struct {
	typ        string
	str        string
	start, end int
}

// queryLexer mirrors lunr.QueryLexer over UTF-16 code units.
type queryLexer struct {
	lexemes            []lexeme
	str                []uint16
	length             int
	pos                int
	start              int
	escapeCharPosition []int
}

func newQueryLexer(s string) *queryLexer {
	u := toU16(s)
	return &queryLexer{str: u, length: len(u)}
}

func (l *queryLexer) run() {
	state := lexTextState
	for state != nil {
		state = state(l)
	}
}

func (l *queryLexer) sliceString() string {
	// JS String#slice clamps indices to the string length (a trailing escape
	// character pushes pos past the end); mirror that.
	clamp := func(i int) int {
		if i > len(l.str) {
			return len(l.str)
		}
		return i
	}
	var subSlices [][]uint16
	sliceStart := l.start
	for _, p := range l.escapeCharPosition {
		sliceEnd := clamp(p)
		subSlices = append(subSlices, l.str[clamp(sliceStart):sliceEnd])
		sliceStart = p + 1
	}
	subSlices = append(subSlices, l.str[clamp(sliceStart):clamp(l.pos)])
	l.escapeCharPosition = nil
	var out []uint16
	for _, s := range subSlices {
		out = append(out, s...)
	}
	return fromU16(out)
}

func (l *queryLexer) emit(typ string) {
	l.lexemes = append(l.lexemes, lexeme{typ: typ, str: l.sliceString(), start: l.start, end: l.pos})
	l.start = l.pos
}

func (l *queryLexer) escapeCharacter() {
	l.escapeCharPosition = append(l.escapeCharPosition, l.pos-1)
	l.pos++
}

// next returns (char, isEOS).
func (l *queryLexer) next() (uint16, bool) {
	if l.pos >= l.length {
		return 0, true
	}
	c := l.str[l.pos]
	l.pos++
	return c, false
}

func (l *queryLexer) width() int { return l.pos - l.start }

func (l *queryLexer) ignore() {
	if l.start == l.pos {
		l.pos++
	}
	l.start = l.pos
}

func (l *queryLexer) backup() { l.pos-- }

func (l *queryLexer) acceptDigitRun() {
	for {
		c, eos := l.next()
		// JS: char.charCodeAt(0) on EOS sentinel "EOS" yields 69 ('E'), not a digit.
		if eos || !(c > 47 && c < 58) {
			if !eos {
				l.backup()
			}
			return
		}
	}
}

func (l *queryLexer) more() bool { return l.pos < l.length }

type lexState func(*queryLexer) lexState

func lexFieldState(l *queryLexer) lexState {
	l.backup()
	l.emit(lexField)
	l.ignore()
	return lexTextState
}

func lexTermState(l *queryLexer) lexState {
	if l.width() > 1 {
		l.backup()
		l.emit(lexTerm)
	}
	l.ignore()
	if l.more() {
		return lexTextState
	}
	return nil
}

func lexEditDistanceState(l *queryLexer) lexState {
	l.ignore()
	l.acceptDigitRun()
	l.emit(lexEditDistance)
	return lexTextState
}

func lexBoostState(l *queryLexer) lexState {
	l.ignore()
	l.acceptDigitRun()
	l.emit(lexBoost)
	return lexTextState
}

func lexEOSState(l *queryLexer) lexState {
	if l.width() > 0 {
		l.emit(lexTerm)
	}
	return nil
}

func lexTextState(l *queryLexer) lexState {
	for {
		char, eos := l.next()
		if eos {
			return lexEOSState
		}

		// Escape character is '\'
		if char == 92 {
			l.escapeCharacter()
			continue
		}

		if char == ':' {
			return lexFieldState
		}

		if char == '~' {
			l.backup()
			if l.width() > 0 {
				l.emit(lexTerm)
			}
			return lexEditDistanceState
		}

		if char == '^' {
			l.backup()
			if l.width() > 0 {
				l.emit(lexTerm)
			}
			return lexBoostState
		}

		// "+" / "-" only as leading characters indicate presence
		if char == '+' && l.width() == 1 {
			l.emit(lexPresence)
			return lexTextState
		}
		if char == '-' && l.width() == 1 {
			l.emit(lexPresence)
			return lexTextState
		}

		if isSeparator(char) {
			return lexTermState
		}
	}
}

// queryParser mirrors lunr.QueryParser.
type queryParser struct {
	lexer         *queryLexer
	query         *Query
	currentClause *Clause
	lexemes       []lexeme
	lexemeIdx     int
}

// ParseQuery mirrors new lunr.QueryParser(str, query).parse().
func ParseQuery(str string, query *Query) (*Query, error) {
	p := &queryParser{lexer: newQueryLexer(str), query: query, currentClause: &Clause{}}
	p.lexer.run()
	p.lexemes = p.lexer.lexemes

	state := parseClauseState
	for state != nil {
		var err error
		state, err = state(p)
		if err != nil {
			return nil, err
		}
	}
	return p.query, nil
}

func (p *queryParser) peekLexeme() *lexeme {
	if p.lexemeIdx >= len(p.lexemes) {
		return nil
	}
	return &p.lexemes[p.lexemeIdx]
}

func (p *queryParser) consumeLexeme() *lexeme {
	l := p.peekLexeme()
	p.lexemeIdx++
	return l
}

func (p *queryParser) nextClause() {
	p.query.AddClause(p.currentClause)
	p.currentClause = &Clause{}
}

type parseState func(*queryParser) (parseState, error)

func parseClauseState(p *queryParser) (parseState, error) {
	l := p.peekLexeme()
	if l == nil {
		return nil, nil
	}
	switch l.typ {
	case lexPresence:
		return parsePresenceState, nil
	case lexField:
		return parseFieldState, nil
	case lexTerm:
		return parseTermState, nil
	default:
		msg := "expected either a field or a term, found " + l.typ
		if len(toU16(l.str)) >= 1 {
			msg += " with value '" + l.str + "'"
		}
		return nil, &QueryParseError{Message: msg, Start: l.start, End: l.end}
	}
}

func parsePresenceState(p *queryParser) (parseState, error) {
	l := p.consumeLexeme()
	if l == nil {
		return nil, nil
	}
	switch l.str {
	case "-":
		p.currentClause.Presence = PresenceProhibited
		p.currentClause.hasPresence = true
	case "+":
		p.currentClause.Presence = PresenceRequired
		p.currentClause.hasPresence = true
	default:
		return nil, &QueryParseError{Message: "unrecognised presence operator'" + l.str + "'", Start: l.start, End: l.end}
	}

	next := p.peekLexeme()
	if next == nil {
		return nil, &QueryParseError{Message: "expecting term or field, found nothing", Start: l.start, End: l.end}
	}
	switch next.typ {
	case lexField:
		return parseFieldState, nil
	case lexTerm:
		return parseTermState, nil
	default:
		return nil, &QueryParseError{Message: "expecting term or field, found '" + next.typ + "'", Start: next.start, End: next.end}
	}
}

func parseFieldState(p *queryParser) (parseState, error) {
	l := p.consumeLexeme()
	if l == nil {
		return nil, nil
	}
	found := false
	for _, f := range p.query.AllFields {
		if f == l.str {
			found = true
			break
		}
	}
	if !found {
		quoted := make([]string, len(p.query.AllFields))
		for i, f := range p.query.AllFields {
			quoted[i] = "'" + f + "'"
		}
		msg := "unrecognised field '" + l.str + "', possible fields: " + strings.Join(quoted, ", ")
		return nil, &QueryParseError{Message: msg, Start: l.start, End: l.end}
	}

	p.currentClause.Fields = []string{l.str}
	p.currentClause.hasFields = true

	next := p.peekLexeme()
	if next == nil {
		return nil, &QueryParseError{Message: "expecting term, found nothing", Start: l.start, End: l.end}
	}
	switch next.typ {
	case lexTerm:
		return parseTermState, nil
	default:
		return nil, &QueryParseError{Message: "expecting term, found '" + next.typ + "'", Start: next.start, End: next.end}
	}
}

func parseTermState(p *queryParser) (parseState, error) {
	l := p.consumeLexeme()
	if l == nil {
		return nil, nil
	}

	p.currentClause.Term = jsLowercaseString(l.str)
	if strings.Contains(l.str, "*") {
		p.currentClause.UsePipeline = false
		p.currentClause.hasUsePipeline = true
	}

	next := p.peekLexeme()
	if next == nil {
		p.nextClause()
		return nil, nil
	}
	switch next.typ {
	case lexTerm:
		p.nextClause()
		return parseTermState, nil
	case lexField:
		p.nextClause()
		return parseFieldState, nil
	case lexEditDistance:
		return parseEditDistanceState, nil
	case lexBoost:
		return parseBoostState, nil
	case lexPresence:
		p.nextClause()
		return parsePresenceState, nil
	default:
		return nil, &QueryParseError{Message: "Unexpected lexeme type '" + next.typ + "'", Start: next.start, End: next.end}
	}
}

// parseIntJS mirrors JS parseInt(str, 10) for lexer digit-run strings.
func parseIntJS(s string) (int, bool) {
	u := toU16(s)
	i := 0
	neg := false
	if i < len(u) && (u[i] == '+' || u[i] == '-') {
		neg = u[i] == '-'
		i++
	}
	start := i
	n := 0
	for i < len(u) && u[i] >= '0' && u[i] <= '9' {
		n = n*10 + int(u[i]-'0')
		i++
	}
	if i == start {
		return 0, false
	}
	if neg {
		n = -n
	}
	return n, true
}

func parseEditDistanceState(p *queryParser) (parseState, error) {
	l := p.consumeLexeme()
	if l == nil {
		return nil, nil
	}
	editDistance, ok := parseIntJS(l.str)
	if !ok {
		return nil, &QueryParseError{Message: "edit distance must be numeric", Start: l.start, End: l.end}
	}
	p.currentClause.EditDistance = editDistance
	p.currentClause.HasEditDistance = true

	next := p.peekLexeme()
	if next == nil {
		p.nextClause()
		return nil, nil
	}
	return termContinuation(p, next)
}

func parseBoostState(p *queryParser) (parseState, error) {
	l := p.consumeLexeme()
	if l == nil {
		return nil, nil
	}
	boost, ok := parseIntJS(l.str)
	if !ok {
		return nil, &QueryParseError{Message: "boost must be numeric", Start: l.start, End: l.end}
	}
	p.currentClause.Boost = float64(boost)
	p.currentClause.hasBoost = true

	next := p.peekLexeme()
	if next == nil {
		p.nextClause()
		return nil, nil
	}
	return termContinuation(p, next)
}

func termContinuation(p *queryParser, next *lexeme) (parseState, error) {
	switch next.typ {
	case lexTerm:
		p.nextClause()
		return parseTermState, nil
	case lexField:
		p.nextClause()
		return parseFieldState, nil
	case lexEditDistance:
		return parseEditDistanceState, nil
	case lexBoost:
		return parseBoostState, nil
	case lexPresence:
		p.nextClause()
		return parsePresenceState, nil
	default:
		return nil, &QueryParseError{Message: "Unexpected lexeme type '" + next.typ + "'", Start: next.start, End: next.end}
	}
}
