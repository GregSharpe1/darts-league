package tournament

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

type registrationNotification struct {
	name  string
	entry Entry
}

type signupRecorder []registrationNotification

func (r *signupRecorder) NotifyTournamentRegistered(_ context.Context, name string, entry Entry) {
	*r = append(*r, registrationNotification{name, entry})
}

func TestReturningTournamentRegistrationNotifiesOnce(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifications := signupRecorder{}
	service := Service{Store: store, Now: time.Now, Notifier: &notifications}
	event, err := service.Create(ctx, "City Cup", "2027-03-20", "", DefaultMatchFormat())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetTournamentRegistration(ctx, event.Slug, true); err != nil {
		t.Fatal(err)
	}
	entry := Entry{DisplayName: " Maya  Evans ", Nickname: " Flight ", ReturningIdentityID: 42}
	created, err := service.RegisterEntry(ctx, event.Slug, entry)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.ReturningIdentityID != 42 || created.DisplayName != "Maya Evans" || created.Nickname != "Flight" {
		t.Fatalf("returning registration: %+v", created)
	}
	if _, err := service.RegisterEntry(ctx, event.Slug, entry); !errors.Is(err, league.ErrDuplicatePlayerName) {
		t.Fatalf("duplicate returning registration: %v", err)
	}
	if len(notifications) != 1 || notifications[0].name != event.Name || notifications[0].entry != created {
		t.Fatalf("expected one notification containing the saved entry, got %+v", notifications)
	}
}
