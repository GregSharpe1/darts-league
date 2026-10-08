package league

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func TestMappedImportPreservesEveryReferenceAndThrow(t *testing.T) {
	data, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatal(err)
	}
	imported, err := autodarts.Parse(examples["producer"])
	if err != nil {
		t.Fatal(err)
	}
	record := ImportRecord{Import: imported, Mapping: map[string]int64{"seat-a": 22, "seat-b": 11}}
	mapped, err := record.MappedImport()
	if err != nil {
		t.Fatal(err)
	}
	if mapped.Payload != nil {
		t.Fatal("mapped projection carries raw source")
	}
	for i, p := range mapped.Players {
		original := record.Import.Players[i]
		if p.ID != strconv.FormatInt(record.Mapping[original.ID], 10) || p.AccountID != nil || !reflect.DeepEqual(p.Stats, original.Stats) || p.LegsWon != original.LegsWon {
			t.Fatalf("player identity/stats changed: %+v", p)
		}
	}
	for i, leg := range mapped.Detail.Legs {
		original := record.Import.Detail.Legs[i]
		if original.WinnerID != nil && (leg.WinnerID == nil || *leg.WinnerID != strconv.FormatInt(record.Mapping[*original.WinnerID], 10)) {
			t.Fatal("winner reference not remapped")
		}
		for j, visit := range leg.Visits {
			previous := original.Visits[j]
			if visit.PlayerID != strconv.FormatInt(record.Mapping[previous.PlayerID], 10) || !reflect.DeepEqual(visit.Throws, previous.Throws) || visit.Bust != previous.Bust || visit.StartRemaining != previous.StartRemaining || visit.EndRemaining != previous.EndRemaining {
				t.Fatal("visit identity or evidence changed")
			}
		}
	}
}
