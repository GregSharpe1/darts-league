package tournament

import "testing"

func TestTournamentSlug(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"  Friday / Night -- Knockout #2! ", "friday-night-knockout-2"},
		{"Caf\u00e9 Cup", "cafe-cup"},
		{"\u0926\u093f\u0932\u094d\u0932\u0940 Cup", "\u0926\u093f\u0932\u094d\u0932\u0940-cup"},
		{"!!!", "tournament"},
	} {
		if got := tournamentSlug(tt.name); got != tt.want {
			t.Errorf("tournamentSlug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
