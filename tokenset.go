// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"errors"
	"sort"
	"strconv"
)

// TokenSet is a minimal DFA over UTF-16 code units, mirroring lunr.TokenSet.
// Edge iteration follows V8 Object.keys order: integer-like keys ("0".."9")
// ascending, then other keys in insertion order.
type TokenSet struct {
	Final bool
	Edges map[uint16]*TokenSet

	edgeOrder []uint16 // insertion order of edge labels
	id        int
	str       string // cached signature
	hasStr    bool
}

var tokenSetNextID = 1

func NewTokenSet() *TokenSet {
	ts := &TokenSet{Edges: map[uint16]*TokenSet{}, id: tokenSetNextID}
	tokenSetNextID++
	return ts
}

// addEdge registers an edge, preserving insertion order.
func (ts *TokenSet) addEdge(label uint16, node *TokenSet) {
	if _, exists := ts.Edges[label]; !exists {
		ts.edgeOrder = append(ts.edgeOrder, label)
	}
	ts.Edges[label] = node
}

// keys returns edge labels in V8 Object.keys order.
func (ts *TokenSet) keys() []uint16 {
	var digits, rest []uint16
	for _, l := range ts.edgeOrder {
		if l >= '0' && l <= '9' {
			digits = append(digits, l)
		} else {
			rest = append(rest, l)
		}
	}
	sort.Slice(digits, func(i, j int) bool { return digits[i] < digits[j] })
	return append(digits, rest...)
}

// sortedLabels returns edge labels sorted by code unit (JS Array#sort on
// single-char strings).
func (ts *TokenSet) sortedLabels() []uint16 {
	labels := append([]uint16{}, ts.edgeOrder...)
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	return labels
}

// TokenSetFromArray mirrors lunr.TokenSet.fromArray.
func TokenSetFromArray(arr []string) (*TokenSet, error) {
	b := &tokenSetBuilder{root: NewTokenSet(), minimizedNodes: map[string]*TokenSet{}}
	for _, w := range arr {
		if err := b.insert(toU16(w)); err != nil {
			return nil, err
		}
	}
	b.finish()
	return b.root, nil
}

// TokenSetFromString mirrors lunr.TokenSet.fromString ("*" wildcards).
func TokenSetFromString(s string) *TokenSet {
	node := NewTokenSet()
	root := node
	u := toU16(s)
	for i, char := range u {
		final := i == len(u)-1
		if char == '*' {
			node.addEdge(char, node)
			node.Final = final
		} else {
			next := NewTokenSet()
			next.Final = final
			node.addEdge(char, next)
			node = next
		}
	}
	return root
}

type fuzzyFrame struct {
	node           *TokenSet
	editsRemaining int
	str            []uint16
}

// TokenSetFromFuzzyString mirrors lunr.TokenSet.fromFuzzyString.
func TokenSetFromFuzzyString(s string, editDistance int) *TokenSet {
	root := NewTokenSet()
	stack := []fuzzyFrame{{node: root, editsRemaining: editDistance, str: toU16(s)}}

	for len(stack) > 0 {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// no edit
		if len(frame.str) > 0 {
			char := frame.str[0]
			noEditNode, ok := frame.node.Edges[char]
			if !ok {
				noEditNode = NewTokenSet()
				frame.node.addEdge(char, noEditNode)
			}
			if len(frame.str) == 1 {
				noEditNode.Final = true
			}
			stack = append(stack, fuzzyFrame{node: noEditNode, editsRemaining: frame.editsRemaining, str: frame.str[1:]})
		}

		if frame.editsRemaining == 0 {
			continue
		}

		// insertion
		insertionNode, ok := frame.node.Edges['*']
		if !ok {
			insertionNode = NewTokenSet()
			frame.node.addEdge('*', insertionNode)
		}
		if len(frame.str) == 0 {
			insertionNode.Final = true
		}
		stack = append(stack, fuzzyFrame{node: insertionNode, editsRemaining: frame.editsRemaining - 1, str: frame.str})

		// deletion of a middle character
		if len(frame.str) > 1 {
			stack = append(stack, fuzzyFrame{node: frame.node, editsRemaining: frame.editsRemaining - 1, str: frame.str[1:]})
		}

		// deletion of the last character
		if len(frame.str) == 1 {
			frame.node.Final = true
		}

		// substitution
		if len(frame.str) >= 1 {
			substitutionNode, ok := frame.node.Edges['*']
			if !ok {
				substitutionNode = NewTokenSet()
				frame.node.addEdge('*', substitutionNode)
			}
			if len(frame.str) == 1 {
				substitutionNode.Final = true
			}
			stack = append(stack, fuzzyFrame{node: substitutionNode, editsRemaining: frame.editsRemaining - 1, str: frame.str[1:]})
		}

		// transposition
		if len(frame.str) > 1 {
			charA, charB := frame.str[0], frame.str[1]
			transposeNode, ok := frame.node.Edges[charB]
			if !ok {
				transposeNode = NewTokenSet()
				frame.node.addEdge(charB, transposeNode)
			}
			if len(frame.str) == 1 {
				transposeNode.Final = true
			}
			rest := append([]uint16{charA}, frame.str[2:]...)
			stack = append(stack, fuzzyFrame{node: transposeNode, editsRemaining: frame.editsRemaining - 1, str: rest})
		}
	}

	return root
}

// ToArray mirrors lunr.TokenSet#toArray.
func (ts *TokenSet) ToArray() []string {
	var words []string
	type frame struct {
		prefix []uint16
		node   *TokenSet
	}
	stack := []frame{{prefix: []uint16{}, node: ts}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		edges := f.node.keys()
		if f.node.Final {
			words = append(words, fromU16(f.prefix))
		}
		for _, edge := range edges {
			stack = append(stack, frame{
				prefix: append(append([]uint16{}, f.prefix...), edge),
				node:   f.node.Edges[edge],
			})
		}
	}
	return words
}

// signature mirrors lunr.TokenSet#toString (used as minimization key).
func (ts *TokenSet) signature() string {
	if ts.hasStr {
		return ts.str
	}
	s := "0"
	if ts.Final {
		s = "1"
	}
	for _, label := range ts.sortedLabels() {
		s += string(rune(label)) + strconv.Itoa(ts.Edges[label].id)
	}
	return s
}

// Intersect mirrors lunr.TokenSet#intersect (wildcard-aware).
func (ts *TokenSet) Intersect(b *TokenSet) *TokenSet {
	output := NewTokenSet()
	type frame struct {
		qNode, output, node *TokenSet
	}
	stack := []frame{{qNode: b, output: output, node: ts}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		qEdges := f.qNode.keys()
		nEdges := f.node.keys()
		for _, qEdge := range qEdges {
			for _, nEdge := range nEdges {
				if nEdge == qEdge || qEdge == '*' {
					node := f.node.Edges[nEdge]
					qNode := f.qNode.Edges[qEdge]
					final := node.Final && qNode.Final
					var next *TokenSet
					if existing, ok := f.output.Edges[nEdge]; ok {
						next = existing
						next.Final = next.Final || final
					} else {
						next = NewTokenSet()
						next.Final = final
						f.output.addEdge(nEdge, next)
					}
					stack = append(stack, frame{qNode: qNode, output: next, node: node})
				}
			}
		}
	}
	return output
}

type uncheckedNode struct {
	parent *TokenSet
	char   uint16
	child  *TokenSet
}

// tokenSetBuilder mirrors lunr.TokenSet.Builder (sorted insertion only).
type tokenSetBuilder struct {
	previousWord    []uint16
	hasPreviousWord bool
	root            *TokenSet
	uncheckedNodes  []uncheckedNode
	minimizedNodes  map[string]*TokenSet
}

// lessU16 mirrors JS string comparison (lexicographic by UTF-16 code unit).
func lessU16(a, b []uint16) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func (b *tokenSetBuilder) insert(word []uint16) error {
	if b.hasPreviousWord && lessU16(word, b.previousWord) {
		return errors.New("out of order word insertion")
	}
	commonPrefix := 0
	for i := 0; i < len(word) && i < len(b.previousWord); i++ {
		if word[i] != b.previousWord[i] {
			break
		}
		commonPrefix++
	}

	b.minimize(commonPrefix)

	var node *TokenSet
	if len(b.uncheckedNodes) == 0 {
		node = b.root
	} else {
		node = b.uncheckedNodes[len(b.uncheckedNodes)-1].child
	}

	for i := commonPrefix; i < len(word); i++ {
		nextNode := NewTokenSet()
		char := word[i]
		node.addEdge(char, nextNode)
		b.uncheckedNodes = append(b.uncheckedNodes, uncheckedNode{parent: node, char: char, child: nextNode})
		node = nextNode
	}
	node.Final = true
	b.previousWord = append([]uint16{}, word...)
	b.hasPreviousWord = true
	return nil
}

func (b *tokenSetBuilder) finish() { b.minimize(0) }

func (b *tokenSetBuilder) minimize(downTo int) {
	for i := len(b.uncheckedNodes) - 1; i >= downTo; i-- {
		node := b.uncheckedNodes[i]
		childKey := node.child.signature()
		if existing, ok := b.minimizedNodes[childKey]; ok {
			node.parent.Edges[node.char] = existing
		} else {
			node.child.str = childKey
			node.child.hasStr = true
			b.minimizedNodes[childKey] = node.child
		}
		b.uncheckedNodes = b.uncheckedNodes[:len(b.uncheckedNodes)-1]
	}
}
