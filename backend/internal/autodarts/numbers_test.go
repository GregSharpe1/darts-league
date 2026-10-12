package autodarts

import (
	"strings"
	"testing"
)

func TestIntegerExponentHasSameDigest(t *testing.T) {
	body := string(example(t, "missing_detail"))
	want, err := Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse([]byte(strings.Replace(body, `"base_score": 501`, `"base_score": 5.01e2`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest != want.Digest {
		t.Fatal("numeric spelling changed digest")
	}
}

func TestTimestampRequiresRFC3339Offset(t *testing.T) {
	for _, timestamp := range []string{"2026-06-15T10:00:00+24:00", "2026-06-15T10:00:00+01:60", "2026-06-15T10:00:00,1Z", "2026-06-15T10:00:00"} {
		body := strings.Replace(string(example(t, "producer")), "2026-06-15T10:00:00+01:00", timestamp, 1)
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted %s", timestamp)
		}
	}
}

func TestFirstToTwoMustMatchReportedLegs(t *testing.T) {
	body := strings.Replace(string(example(t, "missing_detail")), `"legs_to_win": 3`, `"legs_to_win": 2`, 1)
	if _, err := Parse([]byte(body)); err == nil {
		t.Fatal("accepted 3-1 reported as first to 2")
	}
	if _, err := Parse([]byte(strings.Replace(body, `"legs_won": 3`, `"legs_won": 2`, 1))); err != nil {
		t.Fatalf("rejected 2-1 first to 2: %v", err)
	}
}
