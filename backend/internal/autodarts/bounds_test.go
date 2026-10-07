package autodarts

import (
	"errors"
	"strings"
	"testing"
)

func TestJSONBoundaryLimits(t *testing.T) {
	base := string(example(t, "producer"))
	for _, tc := range []struct {
		name, body string
		want       error
	}{
		{"null root", `null`, ErrInvalidJSON},
		{"null object", strings.Replace(base, `"settings": {`, `"settings": null,"ignored": {`, 1), ErrInvalidPayload},
		{"null coordinate", strings.Replace(base, `"x": 0`, `"x": null`, 1), ErrInvalidPayload},
		{"duplicate discarded key", strings.Replace(base, `"x": 0`, `"private":1,"private":2,"x":0`, 1), ErrInvalidJSON},
		{"invalid coordinate type", strings.Replace(base, `"x": 0`, `"x": "0"`, 1), ErrInvalidPayload},
		{"nonfinite discarded number", strings.Replace(base, `"x": 0`, `"private":1e999,"x":0`, 1), ErrInvalidJSON},
		{"depth", `{"ignored":` + strings.Repeat(`[`, 12) + `0` + strings.Repeat(`]`, 12) + `}`, ErrInvalidPayload},
		{"count", `{"ignored":[` + strings.Repeat(`0,`, 3000) + `0]}`, ErrInvalidPayload},
		{"bytes", base + strings.Repeat(" ", 128*1024-len(base)+1), ErrPayloadTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.body))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	if _, err := Parse([]byte(base + strings.Repeat(" ", 128*1024-len(base)))); err != nil {
		t.Fatal("rejected exact byte bound", err)
	}
}
