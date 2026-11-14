package lunr

// MatchData collects metadata about a matching document. Mirrors
// lunr.MatchData. Metadata maps term -> field -> key -> values, with
// insertion-ordered keys for JS-identical behavior.
type MatchData struct {
	Metadata  map[string]map[string]map[string][]any
	termOrder []string
}

func NewMatchData(term, field string, metadata map[string][]any) *MatchData {
	md := &MatchData{Metadata: map[string]map[string]map[string][]any{}}
	if term != "" || field != "" {
		cloned := map[string][]any{}
		for k, v := range metadata {
			cloned[k] = append([]any{}, v...)
		}
		md.Metadata[term] = map[string]map[string][]any{field: cloned}
		md.termOrder = append(md.termOrder, term)
	}
	return md
}

func blankMatchData() *MatchData {
	return &MatchData{Metadata: map[string]map[string]map[string][]any{}}
}

// Combine mirrors lunr.MatchData#combine.
func (md *MatchData) Combine(other *MatchData) {
	for _, term := range other.orderedTerms() {
		fields := other.Metadata[term]
		if _, ok := md.Metadata[term]; !ok {
			md.Metadata[term] = map[string]map[string][]any{}
			md.termOrder = append(md.termOrder, term)
		}
		for field, keys := range fields {
			if _, ok := md.Metadata[term][field]; !ok {
				md.Metadata[term][field] = map[string][]any{}
			}
			for key, vals := range keys {
				if existing, ok := md.Metadata[term][field][key]; ok {
					md.Metadata[term][field][key] = append(existing, vals...)
				} else {
					md.Metadata[term][field][key] = vals
				}
			}
		}
	}
}

// Add mirrors lunr.MatchData#add.
func (md *MatchData) Add(term, field string, metadata map[string][]any) {
	if _, ok := md.Metadata[term]; !ok {
		md.Metadata[term] = map[string]map[string][]any{field: metadata}
		md.termOrder = append(md.termOrder, term)
		return
	}
	if _, ok := md.Metadata[term][field]; !ok {
		md.Metadata[term][field] = metadata
		return
	}
	for key, vals := range metadata {
		if existing, ok := md.Metadata[term][field][key]; ok {
			md.Metadata[term][field][key] = append(existing, vals...)
		} else {
			md.Metadata[term][field][key] = vals
		}
	}
}

func (md *MatchData) orderedTerms() []string {
	if len(md.termOrder) == len(md.Metadata) {
		return md.termOrder
	}
	out := make([]string, 0, len(md.Metadata))
	seen := map[string]bool{}
	for _, t := range md.termOrder {
		out = append(out, t)
		seen[t] = true
	}
	for t := range md.Metadata {
		if !seen[t] {
			out = append(out, t)
		}
	}
	return out
}
