// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

// set is lunr.Set: a string set with complete/empty singletons.
type set struct {
	elements map[string]bool
	complete bool // lunr.Set.complete
	empty    bool // lunr.Set.empty
}

var (
	completeSet = &set{complete: true}
	emptySet    = &set{empty: true}
)

func newSet(elements []string) *set {
	s := &set{elements: map[string]bool{}}
	for _, e := range elements {
		s.elements[e] = true
	}
	return s
}

func (s *set) contains(e string) bool {
	if s.complete {
		return true
	}
	if s.empty {
		return false
	}
	return s.elements[e]
}

func (s *set) intersect(other *set) *set {
	if s.complete {
		return other
	}
	if s.empty {
		return s
	}
	if other == completeSet {
		return s
	}
	if other == emptySet {
		return other
	}
	a, b := s, other
	if len(s.elements) >= len(other.elements) {
		a, b = other, s
	}
	var intersection []string
	for e := range a.elements {
		if _, ok := b.elements[e]; ok {
			intersection = append(intersection, e)
		}
	}
	return newSet(intersection)
}

func (s *set) union(other *set) *set {
	if s.complete || other == completeSet {
		return completeSet
	}
	if s.empty {
		return other
	}
	if other == emptySet {
		return s
	}
	elements := make([]string, 0, len(s.elements)+len(other.elements))
	for e := range s.elements {
		elements = append(elements, e)
	}
	for e := range other.elements {
		elements = append(elements, e)
	}
	return newSet(elements)
}
