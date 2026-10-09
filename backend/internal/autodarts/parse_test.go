package autodarts

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func example(t *testing.T, key string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(b, &examples); err != nil {
		t.Fatal(err)
	}
	return examples[key]
}

func TestParseContractExamples(t *testing.T) {
	for _, key := range []string{"producer", "missing_detail", "legacy"} {
		t.Run(key, func(t *testing.T) {
			got, err := Parse(example(t, key))
			if err != nil {
				t.Fatal(err)
			}
			if got.ReviewReason != "" || len(got.Digest) != 64 || len(got.Players) != 2 {
				t.Fatalf("unexpected import: %+v", got)
			}
			if key == "legacy" && (got.Players[0].Stats != nil || got.PlayedAt != nil || got.Detail != nil) {
				t.Fatal("invented legacy data")
			}
		})
	}
}

func TestParseRejectsUntrustedShapes(t *testing.T) {
	base := string(example(t, "missing_detail"))
	for _, body := range []string{
		strings.Replace(base, `"completed": true`, `"completed": true,"completed":false`, 1),
		strings.Replace(base, `"playedAt": null,`, ``, 1),
		strings.Replace(base, `"legs_won": 1`, `"legs_won": null`, 1),
		strings.Replace(base, `"stats": null`, `"stats":{"match_average":1e999}`, 1),
		strings.Replace(base, `autodarts.import.v1`, `autodarts.import.v2`, 1),
		base + `{}`, strings.Repeat(" ", 128*1024) + base,
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted invalid input: %.100s", body)
		}
	}
}

func TestFilteringDoesNotChangeDigest(t *testing.T) {
	base := example(t, "producer")
	a, err := Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse([]byte(strings.Replace(string(base), `"x": 0`, `"email":"private@example.invalid","x": -0.0`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest || strings.Contains(string(b.Payload), "email") {
		t.Fatal("unknown fields retained or affected digest")
	}
}
