// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"errors"
	"fmt"
	"log"
)

// PipelineFunc maps a token to nil (drop it), a single token, or several
// tokens. Mirrors lunr.PipelineFunction.
type PipelineFunc func(token *Token, i int, tokens []*Token) []*Token

// Pipeline maintains an ordered list of functions applied to all tokens.
// Mirrors lunr.Pipeline.
type Pipeline struct {
	stack []registeredFunc
}

// NewPipeline returns an empty pipeline.
func NewPipeline() *Pipeline { return &Pipeline{} }

type registeredFunc struct {
	fn    PipelineFunc
	label string
}

var registeredFunctions = map[string]PipelineFunc{}

// RegisterFunction registers fn under label for pipeline serialization.
// Mirrors lunr.Pipeline.registerFunction.
func RegisterFunction(fn PipelineFunc, label string) {
	if _, exists := registeredFunctions[label]; exists {
		log.Printf("Overwriting existing registered function: %s", label)
	}
	registeredFunctions[label] = fn
}

// LoadPipeline restores a serialized pipeline. Mirrors lunr.Pipeline.load.
func LoadPipeline(labels []string) (*Pipeline, error) {
	p := &Pipeline{}
	for _, label := range labels {
		fn, ok := registeredFunctions[label]
		if !ok {
			return nil, fmt.Errorf("cannot load unregistered function: %s", label)
		}
		p.stack = append(p.stack, registeredFunc{fn: fn, label: label})
	}
	return p, nil
}

// Add appends functions to the pipeline. Mirrors lunr.Pipeline#add.
func (p *Pipeline) Add(fns ...PipelineFunc) {
	for _, fn := range fns {
		p.stack = append(p.stack, registeredFunc{fn: fn, label: labelOf(fn)})
	}
}

// After inserts newFn after the function registered under existingLabel.
// Mirrors lunr.Pipeline#after (which takes function identity; labels are the
// Go port's identity mechanism).
func (p *Pipeline) After(existingLabel string, newFn PipelineFunc) error {
	for i, rf := range p.stack {
		if rf.label == existingLabel {
			p.stack = append(p.stack, registeredFunc{})
			copy(p.stack[i+2:], p.stack[i+1:])
			p.stack[i+1] = registeredFunc{fn: newFn, label: labelOf(newFn)}
			return nil
		}
	}
	return errors.New("cannot find existingFn")
}

// Before inserts newFn before the function registered under existingLabel.
func (p *Pipeline) Before(existingLabel string, newFn PipelineFunc) error {
	for i, rf := range p.stack {
		if rf.label == existingLabel {
			p.stack = append(p.stack, registeredFunc{})
			copy(p.stack[i+1:], p.stack[i:])
			p.stack[i] = registeredFunc{fn: newFn, label: labelOf(newFn)}
			return nil
		}
	}
	return errors.New("cannot find existingFn")
}

// Remove drops the function registered under label. Mirrors lunr.Pipeline#remove.
func (p *Pipeline) Remove(label string) {
	for i, rf := range p.stack {
		if rf.label == label {
			p.stack = append(p.stack[:i], p.stack[i+1:]...)
			return
		}
	}
}

// Reset empties the pipeline. Mirrors lunr.Pipeline#reset.
func (p *Pipeline) Reset() { p.stack = nil }

// Run passes tokens through every function in order. Mirrors lunr.Pipeline#run.
func (p *Pipeline) Run(tokens []*Token) []*Token {
	for _, rf := range p.stack {
		var memo []*Token
		for j, tok := range tokens {
			out := rf.fn(tok, j, tokens)
			memo = append(memo, out...)
		}
		tokens = memo
	}
	return tokens
}

// RunString passes a string through the pipeline. Mirrors lunr.Pipeline#runString.
func (p *Pipeline) RunString(s string, metadata map[string]any) []string {
	out := p.Run([]*Token{NewToken(s, metadata)})
	strs := make([]string, 0, len(out))
	for _, t := range out {
		strs = append(strs, t.Str)
	}
	return strs
}

// Labels serializes the pipeline as registered function labels.
// Mirrors lunr.Pipeline#toJSON.
func (p *Pipeline) Labels() []string {
	labels := make([]string, 0, len(p.stack))
	for _, rf := range p.stack {
		labels = append(labels, rf.label)
	}
	return labels
}

// labelOf finds a function's registered label (empty if unregistered).
func labelOf(fn PipelineFunc) string {
	for label, f := range registeredFunctions {
		if fmt.Sprintf("%p", f) == fmt.Sprintf("%p", fn) {
			return label
		}
	}
	return ""
}

// isWordChar mirrors JS \w: [A-Za-z0-9_], per UTF-16 code unit.
func isWordChar(c uint16) bool {
	return c == '_' ||
		(c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z')
}

// Trimmer mirrors lunr.trimmer: strips leading and trailing \W+ code units.
func Trimmer(token *Token, _ int, _ []*Token) []*Token {
	token.Update(func(s string, _ map[string]any) string {
		u := toU16(s)
		start, end := 0, len(u)
		for start < end && !isWordChar(u[start]) {
			start++
		}
		for end > start && !isWordChar(u[end-1]) {
			end--
		}
		return fromU16(u[start:end])
	})
	return []*Token{token}
}

// GenerateStopWordFilter mirrors lunr.generateStopWordFilter.
func GenerateStopWordFilter(stopWords []string) PipelineFunc {
	words := make(map[string]bool, len(stopWords))
	for _, w := range stopWords {
		words[w] = true
	}
	return func(token *Token, _ int, _ []*Token) []*Token {
		if token != nil && !words[token.Str] {
			return []*Token{token}
		}
		return nil
	}
}

// StopWordFilter is lunr.stopWordFilter: the built-in English stop word list.
var StopWordFilter = GenerateStopWordFilter(stopWords)

var stopWords = []string{
	"a", "able", "about", "across", "after", "all", "almost", "also", "am",
	"among", "an", "and", "any", "are", "as", "at", "be", "because", "been",
	"but", "by", "can", "cannot", "could", "dear", "did", "do", "does",
	"either", "else", "ever", "every", "for", "from", "get", "got", "had",
	"has", "have", "he", "her", "hers", "him", "his", "how", "however", "i",
	"if", "in", "into", "is", "it", "its", "just", "least", "let", "like",
	"likely", "may", "me", "might", "most", "must", "my", "neither", "no",
	"nor", "not", "of", "off", "often", "on", "only", "or", "other", "our",
	"own", "rather", "said", "say", "says", "she", "should", "since", "so",
	"some", "than", "that", "the", "their", "them", "then", "there", "these",
	"they", "this", "tis", "to", "too", "twas", "us", "wants", "was", "we",
	"were", "what", "when", "where", "which", "while", "who", "whom", "why",
	"will", "with", "would", "yet", "you", "your",
}

// Stemmer mirrors lunr.stemmer: the Porter stemmer as a pipeline function.
func Stemmer(token *Token, _ int, _ []*Token) []*Token {
	token.Update(func(s string, _ map[string]any) string { return porterStem(s) })
	return []*Token{token}
}

func init() {
	RegisterFunction(Trimmer, "trimmer")
	RegisterFunction(StopWordFilter, "stopWordFilter")
	RegisterFunction(Stemmer, "stemmer")
}
