package lunr

import (
	"fmt"
	"testing"
)

var benchWords = []string{
	"search", "index", "query", "document", "ranking", "score", "vector",
	"token", "pipeline", "stemmer", "language", "client", "server", "data",
	"system", "engine", "library", "performance", "memory", "build",
}

func benchDocs(n int) []map[string]any {
	docs := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		docs[i] = map[string]any{
			"id":    fmt.Sprint(i),
			"title": fmt.Sprintf("%s %s handbook", benchWords[i%20], benchWords[(i*7)%20]),
			"body": fmt.Sprintf("%s %s %s and %s. %s %s, %s %s %s.",
				benchWords[(i*3)%20], benchWords[(i*5)%20], benchWords[(i*11)%20],
				benchWords[(i*13)%20], benchWords[(i*17)%20], benchWords[(i*19)%20],
				benchWords[(i*2)%20], benchWords[(i*4)%20], benchWords[(i*6)%20]),
		}
	}
	return docs
}

func buildBenchIndex(b *testing.B, n int) *Index {
	idx, err := New(func(bu *Builder) {
		bu.Ref("id")
		_ = bu.Field("title", FieldAttributes{Boost: 10})
		_ = bu.Field("body", FieldAttributes{})
		for _, d := range benchDocs(n) {
			bu.Add(d, 0)
		}
	})
	if err != nil {
		b.Fatal(err)
	}
	return idx
}

func BenchmarkBuild2000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildBenchIndex(b, 2000)
	}
}

func BenchmarkSearchHit(b *testing.B) {
	idx := buildBenchIndex(b, 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := idx.Search("search"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchMiss(b *testing.B) {
	idx := buildBenchIndex(b, 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := idx.Search("zzzqqx"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchFuzzy(b *testing.B) {
	idx := buildBenchIndex(b, 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := idx.Search("search~1"); err != nil {
			b.Fatal(err)
		}
	}
}
