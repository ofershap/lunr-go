// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

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
