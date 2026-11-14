// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"sort"
)

// Version mirrors lunr.version.
const Version = "2.3.9"

// Index is a built lunr index. Mirrors lunr.Index.
type Index struct {
	InvertedIndex    map[string]*Posting
	fieldVectors     []fieldVectorEntry
	fieldVectorByRef map[string]*Vector
	TokenSet         *TokenSet
	Fields           []string
	Pipeline         *Pipeline
}

// Result mirrors lunr.Index~Result.
type Result struct {
	Ref       string
	Score     float64
	MatchData *MatchData
}

// Search mirrors lunr.Index#search: query-string syntax.
func (idx *Index) Search(queryString string) ([]*Result, error) {
	var parseErr error
	results, err := idx.Query(func(q *Query) {
		_, parseErr = ParseQuery(queryString, q)
	})
	if err != nil {
		return nil, err
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return results, nil
}

func tokenSetFromClause(c *Clause) *TokenSet {
	if c.HasEditDistance {
		return TokenSetFromFuzzyString(c.Term, c.EditDistance)
	}
	return TokenSetFromString(c.Term)
}

// Query mirrors lunr.Index#query.
func (idx *Index) Query(fn func(q *Query)) ([]*Result, error) {
	query := NewQuery(idx.Fields)
	matchingFields := map[string]*MatchData{}
	var matchingFieldOrder []string
	queryVectors := map[string]*Vector{}
	termFieldCache := map[string]bool{}
	requiredMatches := map[string]*set{}
	prohibitedMatches := map[string]*set{}

	for _, f := range idx.Fields {
		queryVectors[f] = NewVector(nil)
	}

	fn(query)

	// Guard the resource hazard lunr.js leaves open: a huge edit distance
	// builds a token set whose size grows with the distance and effectively
	// hangs the search. Reject it before any work is done.
	for _, clause := range query.Clauses {
		if clause.HasEditDistance && clause.EditDistance > MaxEditDistance {
			return nil, &EditDistanceError{Distance: clause.EditDistance, Max: MaxEditDistance}
		}
	}

	for _, clause := range query.Clauses {
		var terms []string
		clauseMatches := emptySet

		if clause.UsePipeline {
			terms = idx.Pipeline.RunString(clause.Term, map[string]any{"fields": clause.Fields})
		} else {
			terms = []string{clause.Term}
		}

	termLoop:
		for _, term := range terms {
			clause.Term = term

			termTokenSet := tokenSetFromClause(clause)
			expandedTerms := idx.TokenSet.Intersect(termTokenSet).ToArray()

			if len(expandedTerms) == 0 && clause.Presence == PresenceRequired {
				for _, field := range clause.Fields {
					requiredMatches[field] = emptySet
				}
				break termLoop
			}

			for _, expandedTerm := range expandedTerms {
				posting := idx.InvertedIndex[expandedTerm]
				termIndex := posting.Index

				for _, field := range clause.Fields {
					fieldPosting := posting.Fields[field]
					matchingDocumentRefs := orderedDocRefs(posting, field)
					termField := expandedTerm + "/" + field
					matchingDocumentsSet := newSet(matchingDocumentRefs)

					if clause.Presence == PresenceRequired {
						clauseMatches = clauseMatches.union(matchingDocumentsSet)
						if _, ok := requiredMatches[field]; !ok {
							requiredMatches[field] = completeSet
						}
					}

					if clause.Presence == PresenceProhibited {
						if _, ok := prohibitedMatches[field]; !ok {
							prohibitedMatches[field] = emptySet
						}
						prohibitedMatches[field] = prohibitedMatches[field].union(matchingDocumentsSet)
						continue
					}

					queryVectors[field].Upsert(termIndex, clause.Boost, func(a, b float64) float64 { return a + b })

					if termFieldCache[termField] {
						continue
					}

					for _, docRef := range matchingDocumentRefs {
						fieldRef := NewFieldRef(docRef, field, "")
						metadata := fieldPosting[docRef]
						if existing, ok := matchingFields[fieldRef.String()]; !ok {
							matchingFields[fieldRef.String()] = NewMatchData(expandedTerm, field, metadata)
							matchingFieldOrder = append(matchingFieldOrder, fieldRef.String())
						} else {
							existing.Add(expandedTerm, field, metadata)
						}
					}

					termFieldCache[termField] = true
				}
			}
		}

		if clause.Presence == PresenceRequired {
			for _, field := range clause.Fields {
				requiredMatches[field] = requiredMatches[field].intersect(clauseMatches)
			}
		}
	}

	allRequiredMatches := completeSet
	allProhibitedMatches := emptySet

	for _, field := range idx.Fields {
		if rm, ok := requiredMatches[field]; ok {
			allRequiredMatches = allRequiredMatches.intersect(rm)
		}
		if pm, ok := prohibitedMatches[field]; ok {
			allProhibitedMatches = allProhibitedMatches.union(pm)
		}
	}

	if query.IsNegated() {
		matchingFieldOrder = nil
		matchingFields = map[string]*MatchData{}
		for _, fve := range idx.fieldVectors {
			matchingFields[fve.ref] = blankMatchData()
			matchingFieldOrder = append(matchingFieldOrder, fve.ref)
		}
	}

	var results []*Result
	matches := map[string]*Result{}

	for _, fieldRefStr := range matchingFieldOrder {
		fieldRef, err := FieldRefFromString(fieldRefStr)
		if err != nil {
			return nil, err
		}
		docRef := fieldRef.DocRef

		if !allRequiredMatches.contains(docRef) {
			continue
		}
		if allProhibitedMatches.contains(docRef) {
			continue
		}

		fieldVector := idx.fieldVectorByRef[fieldRefStr]
		score := queryVectors[fieldRef.Field].Similarity(fieldVector)

		if docMatch, ok := matches[docRef]; ok {
			docMatch.Score += score
			docMatch.MatchData.Combine(matchingFields[fieldRefStr])
		} else {
			match := &Result{Ref: docRef, Score: score, MatchData: matchingFields[fieldRefStr]}
			matches[docRef] = match
			results = append(results, match)
		}
	}

	// V8's Array#sort is stable; Go SliceStable matches.
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, nil
}

// orderedDocRefs returns doc refs for one posting field in V8 Object.keys
// order: canonical array indices ascending, then insertion order.
func orderedDocRefs(p *Posting, field string) []string {
	return v8SortKeys(p.FieldDocOrder[field])
}

// v8SortKeys partitions keys like V8 Object.keys: integer-like keys first in
// numeric order, the rest keep their given (insertion) order.
func v8SortKeys(keys []string) []string {
	var nums []uint64
	numSet := map[uint64]string{}
	var rest []string
	for _, k := range keys {
		if n, ok := jsArrayIndex(k); ok {
			if _, dup := numSet[n]; !dup {
				nums = append(nums, n)
			}
			numSet[n] = k
		} else {
			rest = append(rest, k)
		}
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	out := make([]string, 0, len(keys))
	for _, n := range nums {
		out = append(out, numSet[n])
	}
	return append(out, rest...)
}

// jsArrayIndex reports whether s is a canonical JS array index key.
func jsArrayIndex(s string) (uint64, bool) {
	if s == "" || len(s) > 10 {
		return 0, false
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + uint64(s[i]-'0')
	}
	if n >= 4294967295 {
		return 0, false
	}
	return n, true
}
