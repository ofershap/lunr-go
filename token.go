package lunr

// Token wraps a string token as it passes through the text processing
// pipeline. Mirrors lunr.Token.
type Token struct {
	Str      string
	Metadata map[string]any
}

func NewToken(s string, metadata map[string]any) *Token {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &Token{Str: s, Metadata: metadata}
}

func (t *Token) String() string { return t.Str }

// Update applies fn to the wrapped string. Mirrors lunr.Token#update.
func (t *Token) Update(fn func(s string, metadata map[string]any) string) *Token {
	t.Str = fn(t.Str, t.Metadata)
	return t
}

// Clone returns a copy sharing metadata, optionally transforming the string.
// Mirrors lunr.Token#clone.
func (t *Token) Clone(fn func(s string, metadata map[string]any) string) *Token {
	if fn == nil {
		fn = func(s string, _ map[string]any) string { return s }
	}
	return NewToken(fn(t.Str, t.Metadata), t.Metadata)
}

// FieldRef identifies a field within a document. Mirrors lunr.FieldRef.
type FieldRef struct {
	DocRef   string
	Field    string
	strValue string
}

// FieldRefJoiner joins field name and document ref in a field ref string.
const FieldRefJoiner = "/"

func NewFieldRef(docRef, field, strValue string) FieldRef {
	return FieldRef{DocRef: docRef, Field: field, strValue: strValue}
}

// FieldRefFromString parses "field/docRef". Mirrors lunr.FieldRef.fromString.
func FieldRefFromString(s string) (FieldRef, error) {
	n := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			n = i
			break
		}
	}
	if n == -1 {
		return FieldRef{}, &ParseError{Message: "malformed field ref string", Name: "ParseError"}
	}
	return FieldRef{DocRef: s[n+1:], Field: s[:n], strValue: s}, nil
}

func (fr FieldRef) String() string {
	if fr.strValue == "" {
		fr.strValue = fr.Field + FieldRefJoiner + fr.DocRef
	}
	return fr.strValue
}
