package lunr

// Wildcard constants mirror lunr.Query.wildcard.
const (
	WildcardNone     = 0
	WildcardLeading  = 1
	WildcardTrailing = 2
)

// Presence constants mirror lunr.Query.presence.
const (
	PresenceOptional   = 1
	PresenceRequired   = 2
	PresenceProhibited = 3
)

// Clause mirrors a lunr.Query clause. Fields nil means "all fields";
// HasEditDistance tracks JS 'editDistance' in clause.
type Clause struct {
	Term            string
	Fields          []string
	Boost           float64
	EditDistance    int
	HasEditDistance bool
	UsePipeline     bool
	Wildcard        int
	Presence        int

	hasFields      bool
	hasBoost       bool
	hasUsePipeline bool
	hasWildcard    bool
	hasPresence    bool
}

// Query mirrors lunr.Query.
type Query struct {
	Clauses   []*Clause
	AllFields []string
}

func NewQuery(allFields []string) *Query {
	return &Query{AllFields: allFields}
}

// ClauseOption customizes a clause added via Term.
type ClauseOption func(*Clause)

// WithFields scopes the clause to fields.
func WithFields(fields ...string) ClauseOption {
	return func(c *Clause) { c.Fields = fields; c.hasFields = true }
}

// WithBoost sets the clause boost.
func WithBoost(boost float64) ClauseOption {
	return func(c *Clause) { c.Boost = boost; c.hasBoost = true }
}

// WithEditDistance enables fuzzy matching.
func WithEditDistance(d int) ClauseOption {
	return func(c *Clause) { c.EditDistance = d; c.HasEditDistance = true }
}

// WithUsePipeline controls search-pipeline processing of the term.
func WithUsePipeline(use bool) ClauseOption {
	return func(c *Clause) { c.UsePipeline = use; c.hasUsePipeline = true }
}

// WithWildcard sets automatic wildcard insertion (WildCard constants).
func WithWildcard(w int) ClauseOption {
	return func(c *Clause) { c.Wildcard = w; c.hasWildcard = true }
}

// WithPresence sets the term presence requirement.
func WithPresence(p int) ClauseOption {
	return func(c *Clause) { c.Presence = p; c.hasPresence = true }
}

// AddClause mirrors lunr.Query#clause: defaults are applied here, matching
// JS 'key in clause' checks against options that were explicitly set.
func (q *Query) AddClause(c *Clause) *Query {
	if !c.hasFields {
		c.Fields = q.AllFields
	}
	if !c.hasBoost {
		c.Boost = 1
	}
	if !c.hasUsePipeline {
		c.UsePipeline = true
	}
	if !c.hasWildcard {
		c.Wildcard = WildcardNone
	}
	if (c.Wildcard&WildcardLeading) != 0 && !startsWithCodeUnit(c.Term, '*') {
		c.Term = "*" + c.Term
	}
	if (c.Wildcard&WildcardTrailing) != 0 && !endsWithCodeUnit(c.Term, '*') {
		c.Term = c.Term + "*"
	}
	if !c.hasPresence {
		c.Presence = PresenceOptional
	}
	q.Clauses = append(q.Clauses, c)
	return q
}

func startsWithCodeUnit(s string, c uint16) bool {
	u := toU16(s)
	return len(u) > 0 && u[0] == c
}

func endsWithCodeUnit(s string, c uint16) bool {
	u := toU16(s)
	return len(u) > 0 && u[len(u)-1] == c
}

// Term mirrors lunr.Query#term: one clause per term, options shared.
func (q *Query) Term(terms []string, opts ...ClauseOption) *Query {
	for _, t := range terms {
		c := &Clause{Term: t}
		for _, opt := range opts {
			opt(c)
		}
		q.AddClause(c)
	}
	return q
}

// IsNegated mirrors lunr.Query#isNegated.
func (q *Query) IsNegated() bool {
	for _, c := range q.Clauses {
		if c.Presence != PresenceProhibited {
			return false
		}
	}
	return true
}
