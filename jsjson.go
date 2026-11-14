package lunr

import (
	"fmt"
	"strings"
)

// jsjson.go implements JSON serialization that is byte-identical to
// JavaScript's JSON.stringify (no indent), plus an order-preserving parser
// for loading serialized indexes. This powers index interchange with
// lunr.js: a Go-serialized index byte-matches what lunr.js produces, and a
// lunr.js-serialized index loads into Go losslessly.

// jsWriteString appends s JSON-escaped exactly as JS JSON.stringify does:
// only ", \ and control characters are escaped; U+2028/U+2029 are emitted
// raw (modern V8); hex escapes use lowercase digits.
func jsWriteString(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
}

// jsWriteValue appends v as JSON with JS number formatting. Supported
// types: string, float64, int, [2]int, []any, []string, bool, nil.
func jsWriteValue(sb *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case string:
		jsWriteString(sb, t)
	case bool:
		if t {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case float64:
		sb.WriteString(jsNumberToString(t))
	case []float64:
		sb.WriteByte('[')
		for i, el := range t {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(jsNumberToString(el))
		}
		sb.WriteByte(']')
	case int:
		sb.WriteString(jsNumberToString(float64(t)))
	case int64:
		sb.WriteString(jsNumberToString(float64(t)))
	case [2]int:
		sb.WriteByte('[')
		sb.WriteString(jsNumberToString(float64(t[0])))
		sb.WriteByte(',')
		sb.WriteString(jsNumberToString(float64(t[1])))
		sb.WriteByte(']')
	case []any:
		sb.WriteByte('[')
		for i, el := range t {
			if i > 0 {
				sb.WriteByte(',')
			}
			jsWriteValue(sb, el)
		}
		sb.WriteByte(']')
	case []string:
		sb.WriteByte('[')
		for i, el := range t {
			if i > 0 {
				sb.WriteByte(',')
			}
			jsWriteString(sb, el)
		}
		sb.WriteByte(']')
	default:
		panic(fmt.Sprintf("jsWriteValue: unsupported type %T", v))
	}
}

// Serialize mirrors JSON.stringify(lunr.Index#toJSON()).
func (idx *Index) Serialize() string {
	var sb strings.Builder
	sb.WriteString(`{"version":`)
	jsWriteString(&sb, Version)
	sb.WriteString(`,"fields":`)
	jsWriteValue(&sb, idx.Fields)
	sb.WriteString(`,"fieldVectors":[`)
	for i, fve := range idx.fieldVectors {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('[')
		jsWriteString(&sb, fve.ref)
		sb.WriteByte(',')
		jsWriteValue(&sb, fve.vector.Elements)
		sb.WriteByte(']')
	}
	sb.WriteString(`],"invertedIndex":[`)
	terms := make([]string, 0, len(idx.InvertedIndex))
	for term := range idx.InvertedIndex {
		terms = append(terms, term)
	}
	sortByCodeUnit(terms)
	for i, term := range terms {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('[')
		jsWriteString(&sb, term)
		sb.WriteByte(',')
		jsWritePosting(&sb, idx.InvertedIndex[term])
		sb.WriteByte(']')
	}
	sb.WriteString(`],"pipeline":`)
	jsWriteValue(&sb, idx.Pipeline.Labels())
	sb.WriteByte('}')
	return sb.String()
}

// jsWritePosting writes a posting object: {"_index":N,"field":{docRef:meta}}.
// Field order is the posting's insertion order (tracked via FieldDocOrder
// keys); doc refs follow V8 Object.keys order.
func jsWritePosting(sb *strings.Builder, p *Posting) {
	sb.WriteString(`{"_index":`)
	sb.WriteString(jsNumberToString(p.Index))
	for _, field := range orderedFieldNames(p) {
		sb.WriteByte(',')
		jsWriteString(sb, field)
		sb.WriteString(`:{`)
		docs := p.Fields[field]
		docRefs := v8SortKeys(p.FieldDocOrder[field])
		for i, docRef := range docRefs {
			if i > 0 {
				sb.WriteByte(',')
			}
			jsWriteString(sb, docRef)
			sb.WriteByte(':')
			jsWriteMetadata(sb, docs[docRef])
		}
		sb.WriteByte('}')
	}
	sb.WriteByte('}')
}

func orderedFieldNames(p *Posting) []string {
	if len(p.FieldOrder) > 0 {
		return p.FieldOrder
	}
	out := make([]string, 0, len(p.Fields))
	for f := range p.Fields {
		out = append(out, f)
	}
	sortByCodeUnit(out)
	return out
}

func jsWriteMetadata(sb *strings.Builder, meta map[string][]any) {
	sb.WriteByte('{')
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sortByCodeUnit(keys)
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		jsWriteString(sb, k)
		sb.WriteByte(':')
		jsWriteValue(sb, meta[k])
	}
	sb.WriteByte('}')
}

func sortByCodeUnit(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && lessU16(toU16(s[j]), toU16(s[j-1])); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
