// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import "fmt"

// isJSSpace reports whether a UTF-16 code unit matches JavaScript's \s
// (White_Space per the JS spec, which is a fixed list, not Go's unicode.IsSpace).
func isJSSpace(c uint16) bool {
	switch c {
	case 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x20, 0xA0, 0x1680,
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006,
		0x2007, 0x2008, 0x2009, 0x200A, 0x2028, 0x2029, 0x202F,
		0x205F, 0x3000, 0xFEFF:
		return true
	}
	return false
}

// isSeparator mirrors lunr.tokenizer.separator /[\s\-]+/ applied to a single
// UTF-16 code unit: JS whitespace or hyphen-minus.
func isSeparator(c uint16) bool {
	return c == '-' || isJSSpace(c)
}

// jsAsString mirrors lunr.utils.asString + the tokenizer's toString coercion
// for the values lunr indexes (strings, numbers, bools, dates-as-strings are
// out of scope; objects/arrays are handled by callers).
func jsAsString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return jsNumberToString(t)
	case float32:
		return jsNumberToString(float64(t))
	case int:
		return jsNumberToString(float64(t))
	case int64:
		return jsNumberToString(float64(t))
	default:
		return fmt.Sprintf("%v", t)
	}
}

// Tokenize splits a value into tokens. Mirrors lunr.tokenizer: the value is
// coerced to a string, lowercased with JS toLowerCase semantics, and split on
// runs of JS-whitespace or hyphen. Token positions are UTF-16 code-unit
// offsets, exactly as lunr reports them.
func Tokenize(obj any, metadata map[string]any) []*Token {
	if obj == nil {
		return nil
	}
	if arr, ok := obj.([]any); ok {
		tokens := make([]*Token, 0, len(arr))
		for _, el := range arr {
			tokens = append(tokens, NewToken(jsLowercaseString(jsAsString(el)), cloneMetadata(metadata)))
		}
		return tokens
	}
	if arr, ok := obj.([]string); ok {
		tokens := make([]*Token, 0, len(arr))
		for _, el := range arr {
			tokens = append(tokens, NewToken(jsLowercaseString(el), cloneMetadata(metadata)))
		}
		return tokens
	}

	u := toU16(jsLowercaseString(jsAsString(obj)))
	length := len(u)
	var tokens []*Token
	sliceStart := 0
	for sliceEnd := 0; sliceEnd <= length; sliceEnd++ {
		var char uint16
		if sliceEnd < length {
			char = u[sliceEnd]
		}
		sliceLength := sliceEnd - sliceStart
		if sliceEnd == length || isSeparator(char) {
			if sliceLength > 0 {
				md := cloneMetadata(metadata)
				md["position"] = [2]int{sliceStart, sliceLength}
				md["index"] = len(tokens)
				tokens = append(tokens, NewToken(fromU16(u[sliceStart:sliceEnd]), md))
			}
			sliceStart = sliceEnd + 1
		}
	}
	return tokens
}

func cloneMetadata(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
