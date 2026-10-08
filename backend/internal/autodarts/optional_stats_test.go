package autodarts

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestOptionalStatsPersistenceAndBounds(t *testing.T) {
	base := string(example(t, "missing_detail"))
	stats := `"stats":{"match_average":null,"points_scored":null,"darts_thrown":null,"checkout_hits":null,"checkout_attempts":null,%s}`
	for _, field := range []struct {
		name string
		max  int
	}{
		{"first_nine_average", 180}, {"average_until_170", 180}, {"highest_finish", 170},
		{"total_180", 1000}, {"less_60", 1000}, {"plus_60", 1000},
		{"plus_100", 1000}, {"plus_140", 1000}, {"plus_170", 1000},
	} {
		for _, value := range []string{"null", "0", fmt.Sprint(field.max), "-1", fmt.Sprint(field.max + 1), `"1"`, "true", "{}", "[]", "1.5"} {
			t.Run(field.name+"/"+value, func(t *testing.T) {
				extension := fmt.Sprintf(`"%s":%s`, field.name, value)
				body := strings.Replace(base, `"stats": null`, fmt.Sprintf(stats, extension+`,"private":"discard"`), 1)
				got, err := Parse([]byte(body))
				valid := value == "null" || value == "0" || value == fmt.Sprint(field.max) || (value == "1.5" && field.max == 180)
				if !valid {
					if err == nil {
						t.Fatal("accepted invalid optional statistic")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(got.Payload), extension) || strings.Contains(string(got.Payload), "private") {
					t.Fatalf("incorrect filtered JSONB: %s", got.Payload)
				}
				var stored Payload
				if err := json.Unmarshal(got.Payload, &stored); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stored.Players, got.Players) {
					t.Fatal("typed source JSON read lost statistics")
				}
				encoded, err := json.Marshal(stored.Players[0].Stats)
				if err != nil {
					t.Fatal(err)
				}
				if value == "null" {
					if strings.Contains(string(encoded), field.name) {
						t.Fatal("nil extension must omit on typed output")
					}
				} else if !strings.Contains(string(encoded), extension) {
					t.Fatalf("typed output lost known value: %s", encoded)
				}
			})
		}
	}
}

func TestAbsentOptionalStatsDoNotChangeCanonicalPayload(t *testing.T) {
	for key, digest := range map[string]string{
		"producer":       "c158ce1844907add93709652d7428d056feeac9513710804369a0105c0a3764d",
		"missing_detail": "4a09738a84a835a483273ca37c62fdbc01abdde03fde0adccd0be9a36c5fdbf9",
	} {
		got, err := Parse(example(t, key))
		if err != nil {
			t.Fatal(err)
		}
		if got.Digest != digest {
			t.Fatal("unextended v1 golden digest changed")
		}
		var stored Payload
		if err := json.Unmarshal(got.Payload, &stored); err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(stored)
		if err != nil {
			t.Fatal(err)
		}
		again, err := Parse(body)
		if err != nil || again.Digest != got.Digest || string(again.Payload) != string(got.Payload) {
			t.Fatalf("absent extensions changed canonical bytes/digest: %v", err)
		}
	}
}
