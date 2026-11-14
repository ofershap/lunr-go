// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"errors"
	"sort"
)

// idf mirrors lunr.idf (uses the fdlibm log port for bit-exact scores).
func idf(posting *Posting, documentCount int) float64 {
	documentsWithTerm := 0
	for _, docs := range posting.Fields {
		documentsWithTerm += len(docs)
	}
	x := (float64(documentCount) - float64(documentsWithTerm) + 0.5) / (float64(documentsWithTerm) + 0.5)
	return jsLog(1 + abs(x))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Posting is one entry of the inverted index: the term's vector index plus,
// per field, per document ref, the whitelisted metadata (insertion ordered
// for JS-identical serialization).
type Posting struct {
	Index  float64
	Fields map[string]map[string]map[string][]any
	// FieldDocOrder preserves JS object insertion order of doc refs per
	// field (document add order), needed for stable result ordering.
	FieldDocOrder map[string][]string
	// FieldOrder preserves JS object insertion order of field keys.
	FieldOrder []string
}

// Builder performs indexing and produces an Index. Mirrors lunr.Builder.
type Builder struct {
	ref    string
	fields []string
	// fieldAttrs preserves insertion order via fields.
	fieldAttrs map[string]FieldAttributes
	docBoosts  map[string]float64
	docOrder   []string

	InvertedIndex        map[string]*Posting
	fieldTermFrequencies map[string]map[string]float64 // fieldRef.String() -> term -> tf
	fieldRefOrder        []string
	fieldLengths         map[string]float64

	Tokenizer func(obj any, metadata map[string]any) []*Token
	Pipeline  *Pipeline
	// SearchPipeline becomes the built index's pipeline.
	SearchPipeline *Pipeline

	DocumentCount     int
	B                 float64
	K1                float64
	termIndex         float64
	MetadataWhitelist []string

	averageFieldLength map[string]float64
	fieldVectors       []fieldVectorEntry
	tokenSet           *TokenSet
}

// FieldAttributes mirrors lunr.Builder#field attributes.
type FieldAttributes struct {
	Boost     float64 // 0 means unset (JS undefined); use 1
	Extractor func(doc map[string]any) any
}

// NewBuilder mirrors new lunr.Builder (ref "id", b 0.75, k1 1.2).
func NewBuilder() *Builder {
	return &Builder{
		ref:                  "id",
		fieldAttrs:           map[string]FieldAttributes{},
		docBoosts:            map[string]float64{},
		InvertedIndex:        map[string]*Posting{},
		fieldTermFrequencies: map[string]map[string]float64{},
		fieldLengths:         map[string]float64{},
		Tokenizer:            Tokenize,
		Pipeline:             NewPipeline(),
		SearchPipeline:       NewPipeline(),
		B:                    0.75,
		K1:                   1.2,
	}
}

// Ref sets the document reference field. Mirrors lunr.Builder#ref.
func (b *Builder) Ref(ref string) { b.ref = ref }

// Field adds a field to index. Mirrors lunr.Builder#field.
func (b *Builder) Field(name string, attrs FieldAttributes) error {
	for _, c := range name {
		if c == '/' {
			return errors.New("Field '" + name + "' contains illegal character '/'")
		}
	}
	if _, exists := b.fieldAttrs[name]; !exists {
		b.fields = append(b.fields, name)
	}
	b.fieldAttrs[name] = attrs
	return nil
}

// SetB mirrors lunr.Builder#b (clamped to 0-1).
func (b *Builder) SetB(x float64) {
	if x < 0 {
		x = 0
	} else if x > 1 {
		x = 1
	}
	b.B = x
}

// SetK1 mirrors lunr.Builder#k1.
func (b *Builder) SetK1(x float64) { b.K1 = x }

// Add indexes one document. Mirrors lunr.Builder#add. The document is a
// map; the ref value is coerced with JS String() semantics.
func (b *Builder) Add(doc map[string]any, boost float64) {
	docRef := jsAsString(doc[b.ref])
	b.docBoosts[docRef] = boost
	b.DocumentCount++

	for _, fieldName := range b.fields {
		attrs := b.fieldAttrs[fieldName]
		var fieldVal any
		if attrs.Extractor != nil {
			fieldVal = attrs.Extractor(doc)
		} else {
			fieldVal = doc[fieldName]
		}
		tokens := b.Tokenizer(fieldVal, map[string]any{"fields": []string{fieldName}})
		terms := b.Pipeline.Run(tokens)
		fieldRef := NewFieldRef(docRef, fieldName, "")

		key := fieldRef.String()
		fieldTerms := map[string]float64{}
		b.fieldTermFrequencies[key] = fieldTerms
		b.fieldRefOrder = append(b.fieldRefOrder, key)
		b.fieldLengths[key] = float64(len(terms))

		for _, term := range terms {
			fieldTerms[term.Str]++

			posting, ok := b.InvertedIndex[term.Str]
			if !ok {
				posting = &Posting{Index: b.termIndex, Fields: map[string]map[string]map[string][]any{}, FieldDocOrder: map[string][]string{}}
				b.termIndex++
				for _, f := range b.fields {
					posting.Fields[f] = map[string]map[string][]any{}
					posting.FieldOrder = append(posting.FieldOrder, f)
					posting.FieldDocOrder[f] = nil
				}
				b.InvertedIndex[term.Str] = posting
			}

			fieldPosting := posting.Fields[fieldName]
			docMeta, ok := fieldPosting[docRef]
			if !ok {
				docMeta = map[string][]any{}
				fieldPosting[docRef] = docMeta
				posting.FieldDocOrder[fieldName] = append(posting.FieldDocOrder[fieldName], docRef)
			}

			for _, metadataKey := range b.MetadataWhitelist {
				metadata := term.Metadata[metadataKey]
				docMeta[metadataKey] = append(docMeta[metadataKey], metadata)
			}
		}
	}
}

func (b *Builder) calculateAverageFieldLengths() {
	accumulator := map[string]float64{}
	documentsWithField := map[string]float64{}
	for _, key := range b.fieldRefOrder {
		fr, _ := FieldRefFromString(key)
		documentsWithField[fr.Field]++
		accumulator[fr.Field] += b.fieldLengths[key]
	}
	b.averageFieldLength = map[string]float64{}
	for _, fieldName := range b.fields {
		b.averageFieldLength[fieldName] = accumulator[fieldName] / documentsWithField[fieldName]
	}
}

type fieldVectorEntry struct {
	ref    string
	vector *Vector
}

func (b *Builder) createFieldVectors() {
	termIdfCache := map[string]float64{}
	b.fieldVectors = nil

	for _, key := range b.fieldRefOrder {
		fr, _ := FieldRefFromString(key)
		fieldName := fr.Field
		fieldLength := b.fieldLengths[key]
		fieldVector := NewVector(nil)
		termFrequencies := b.fieldTermFrequencies[key]

		fieldBoost := b.fieldAttrs[fieldName].Boost
		if fieldBoost == 0 {
			fieldBoost = 1
		}
		docBoost := b.docBoosts[fr.DocRef]
		if docBoost == 0 {
			docBoost = 1
		}

		for term, tf := range termFrequencies {
			posting := b.InvertedIndex[term]
			termIndex := posting.Index
			idfVal, ok := termIdfCache[term]
			if !ok {
				idfVal = idf(posting, b.DocumentCount)
				termIdfCache[term] = idfVal
			}
			score := idfVal * ((b.K1 + 1) * tf) / (b.K1*(1-b.B+b.B*(fieldLength/b.averageFieldLength[fieldName])) + tf)
			score *= fieldBoost
			score *= docBoost
			// Math.round(score * 1000) / 1000 - round half away from zero,
			// matching JS Math.round for the non-negative scores here.
			scoreWithPrecision := float64(int64(score*1000+0.5)) / 1000
			fieldVector.Insert(termIndex, scoreWithPrecision)
		}

		b.fieldVectors = append(b.fieldVectors, fieldVectorEntry{ref: key, vector: fieldVector})
	}
}

func (b *Builder) createTokenSet() error {
	terms := make([]string, 0, len(b.InvertedIndex))
	for term := range b.InvertedIndex {
		terms = append(terms, term)
	}
	// JS Array#sort on strings: lexicographic by UTF-16 code unit.
	sort.Slice(terms, func(i, j int) bool { return lessU16(toU16(terms[i]), toU16(terms[j])) })
	ts, err := TokenSetFromArray(terms)
	if err != nil {
		return err
	}
	b.tokenSet = ts
	return nil
}

// Build mirrors lunr.Builder#build.
func (b *Builder) Build() (*Index, error) {
	b.calculateAverageFieldLengths()
	b.createFieldVectors()
	if err := b.createTokenSet(); err != nil {
		return nil, err
	}
	byRef := map[string]*Vector{}
	for _, fve := range b.fieldVectors {
		byRef[fve.ref] = fve.vector
	}
	return &Index{
		InvertedIndex:    b.InvertedIndex,
		fieldVectors:     b.fieldVectors,
		fieldVectorByRef: byRef,
		TokenSet:         b.tokenSet,
		Fields:           append([]string{}, b.fields...),
		Pipeline:         b.SearchPipeline,
	}, nil
}

// Use mirrors lunr.Builder#use (plugins).
func (b *Builder) Use(fn func(builder *Builder)) { fn(b) }
