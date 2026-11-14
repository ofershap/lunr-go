package lunr

// New mirrors the lunr(config) shortcut: a builder with the default English
// pipeline (trimmer, stopWordFilter, stemmer) and a search pipeline with the
// stemmer. Configure fields and documents in fn.
func New(fn func(b *Builder)) (*Index, error) {
	builder := NewBuilder()
	builder.Pipeline.Add(Trimmer, StopWordFilter, Stemmer)
	builder.SearchPipeline.Add(Stemmer)
	fn(builder)
	return builder.Build()
}
