package lunr

import (
	"encoding/json"
	"os"
	"testing"
)

// TestExportGoIndex rewrites testdata/index_from_go.json so
// testdata/gen/gen.js can verify JS consumption of the Go serialization.
func TestExportGoIndex(t *testing.T) {
	f := loadFixture(t)
	idx := buildFromFixture(t, f)
	out, _ := json.Marshal(map[string]string{"serialized": idx.Serialize()})
	if err := os.WriteFile("testdata/index_from_go.json", out, 0o644); err != nil {
		t.Fatal(err)
	}
}
