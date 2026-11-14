// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import (
	"fmt"
	"strconv"
	"strings"
)

// omap is an insertion-ordered JSON object.
type omap struct {
	keys []string
	vals map[string]any
}

func newOmap() *omap { return &omap{vals: map[string]any{}} }
func (o *omap) set(k string, v any) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}
func (o *omap) get(k string) (any, bool) { v, ok := o.vals[k]; return v, ok }

type jsonParser struct {
	s   string
	pos int
}

func (p *jsonParser) err(msg string) error {
	return fmt.Errorf("lunr: invalid serialized index JSON at %d: %s", p.pos, msg)
}

func (p *jsonParser) ws() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *jsonParser) value() (any, error) {
	p.ws()
	if p.pos >= len(p.s) {
		return nil, p.err("unexpected end")
	}
	switch c := p.s[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		return p.str()
	case c == 't':
		if strings.HasPrefix(p.s[p.pos:], "true") {
			p.pos += 4
			return true, nil
		}
	case c == 'f':
		if strings.HasPrefix(p.s[p.pos:], "false") {
			p.pos += 5
			return false, nil
		}
	case c == 'n':
		if strings.HasPrefix(p.s[p.pos:], "null") {
			p.pos += 4
			return nil, nil
		}
	default:
		return p.number()
	}
	return nil, p.err("unexpected character")
}

func (p *jsonParser) object() (*omap, error) {
	p.pos++ // {
	o := newOmap()
	p.ws()
	if p.pos < len(p.s) && p.s[p.pos] == '}' {
		p.pos++
		return o, nil
	}
	for {
		p.ws()
		k, err := p.str()
		if err != nil {
			return nil, err
		}
		p.ws()
		if p.pos >= len(p.s) || p.s[p.pos] != ':' {
			return nil, p.err("expected ':'")
		}
		p.pos++
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		o.set(k.(string), v)
		p.ws()
		if p.pos >= len(p.s) {
			return nil, p.err("unexpected end in object")
		}
		if p.s[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.s[p.pos] == '}' {
			p.pos++
			return o, nil
		}
		return nil, p.err("expected ',' or '}'")
	}
}

func (p *jsonParser) array() ([]any, error) {
	p.pos++ // [
	var out []any
	p.ws()
	if p.pos < len(p.s) && p.s[p.pos] == ']' {
		p.pos++
		return out, nil
	}
	for {
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		p.ws()
		if p.pos >= len(p.s) {
			return nil, p.err("unexpected end in array")
		}
		if p.s[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.s[p.pos] == ']' {
			p.pos++
			return out, nil
		}
		return nil, p.err("expected ',' or ']'")
	}
}

func (p *jsonParser) str() (any, error) {
	if p.pos >= len(p.s) || p.s[p.pos] != '"' {
		return nil, p.err("expected string")
	}
	p.pos++
	var sb strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '"' {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' && p.pos+1 < len(p.s) {
			p.pos++
			switch e := p.s[p.pos]; e {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case '/':
				sb.WriteByte('/')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case 'u':
				if p.pos+4 >= len(p.s) {
					return nil, p.err("bad \\u escape")
				}
				n, err := strconv.ParseUint(p.s[p.pos+1:p.pos+5], 16, 32)
				if err != nil {
					return nil, p.err("bad \\u escape")
				}
				sb.WriteRune(rune(n))
				p.pos += 4
			default:
				return nil, p.err("bad escape")
			}
			p.pos++
			continue
		}
		sb.WriteByte(c)
		p.pos++
	}
	return nil, p.err("unterminated string")
}

func (p *jsonParser) number() (any, error) {
	start := p.pos
	for p.pos < len(p.s) && strings.IndexByte("0123456789+-.eE", p.s[p.pos]) >= 0 {
		p.pos++
	}
	if start == p.pos {
		return nil, p.err("bad number")
	}
	n, err := strconv.ParseFloat(p.s[start:p.pos], 64)
	if err != nil {
		return nil, p.err("bad number")
	}
	return n, nil
}

func fixSurrogates(s string) string { return s } // JSON from lunr.js is valid UTF-8 already

// LoadIndex mirrors lunr.Index.load: restores an index serialized by
// lunr.js or by Serialize.
func LoadIndex(data string) (*Index, error) {
	p := &jsonParser{s: data}
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	root, ok := v.(*omap)
	if !ok {
		return nil, fmt.Errorf("lunr: serialized index must be an object")
	}

	version, _ := root.get("version")
	if version != Version {
		fmt.Printf("lunr: version mismatch when loading serialized index: %v (current %s)\n", version, Version)
	}

	fieldsAny, _ := root.get("fields")
	var fields []string
	if arr, ok := fieldsAny.([]any); ok {
		for _, f := range arr {
			fields = append(fields, f.(string))
		}
	}

	var fves []fieldVectorEntry
	byRef := map[string]*Vector{}
	fvAny, _ := root.get("fieldVectors")
	if arr, ok := fvAny.([]any); ok {
		for _, tuple := range arr {
			pair := tuple.([]any)
			ref := pair[0].(string)
			elemsAny := pair[1].([]any)
			elems := make([]float64, len(elemsAny))
			for i, e := range elemsAny {
				elems[i] = e.(float64)
			}
			vec := NewVector(elems)
			fves = append(fves, fieldVectorEntry{ref: ref, vector: vec})
			byRef[ref] = vec
		}
	}

	invertedIndex := map[string]*Posting{}
	builder := &tokenSetBuilder{root: NewTokenSet(), minimizedNodes: map[string]*TokenSet{}}
	iiAny, _ := root.get("invertedIndex")
	if arr, ok := iiAny.([]any); ok {
		for _, tuple := range arr {
			pair := tuple.([]any)
			term := pair[0].(string)
			postingObj := pair[1].(*omap)
			posting := &Posting{
				Fields:        map[string]map[string]map[string][]any{},
				FieldDocOrder: map[string][]string{},
			}
			for _, key := range postingObj.keys {
				if key == "_index" {
					posting.Index = postingObj.vals[key].(float64)
					continue
				}
				posting.FieldOrder = append(posting.FieldOrder, key)
				docsObj := postingObj.vals[key].(*omap)
				docs := map[string]map[string][]any{}
				for _, docRef := range docsObj.keys {
					metaObj := docsObj.vals[docRef].(*omap)
					meta := map[string][]any{}
					for _, mk := range metaObj.keys {
						if vals, ok := metaObj.vals[mk].([]any); ok {
							meta[mk] = vals
						}
					}
					docs[docRef] = meta
					posting.FieldDocOrder[key] = append(posting.FieldDocOrder[key], docRef)
				}
				posting.Fields[key] = docs
			}
			if err := builder.insert(toU16(term)); err != nil {
				return nil, err
			}
			invertedIndex[term] = posting
		}
	}
	builder.finish()

	pipelineAny, _ := root.get("pipeline")
	var labels []string
	if arr, ok := pipelineAny.([]any); ok {
		for _, l := range arr {
			labels = append(labels, l.(string))
		}
	}
	pipeline, err := LoadPipeline(labels)
	if err != nil {
		return nil, err
	}

	return &Index{
		InvertedIndex:    invertedIndex,
		fieldVectors:     fves,
		fieldVectorByRef: byRef,
		TokenSet:         builder.root,
		Fields:           fields,
		Pipeline:         pipeline,
	}, nil
}
