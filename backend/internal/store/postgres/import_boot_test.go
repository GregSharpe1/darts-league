package postgres

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestBootMigratesOldPendingTable(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	schema := pgx.Identifier{"import_boot_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err := conn.Exec(ctx, `CREATE SCHEMA `+schema+`; SET search_path TO `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := conn.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
	}()
	if _, err := conn.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := conn.QueryRow(ctx, `INSERT INTO pending_results(player_one_name,player_one_legs,player_two_name,player_two_legs,status) VALUES('A',3,'B',1,'rejected') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	for range 2 {
		s, err := Open(ctx, u.String())
		if err != nil {
			t.Fatal(err)
		}
		record, err := s.GetImport(ctx, id)
		s.Close()
		if err != nil {
			t.Fatal(err)
		}
		if record.Import.Digest == "" || record.Import.SettingsEvidence != "legacy_stored_summary" || record.Pending.Status != "rejected" || record.Import.ExternalMatchID != "" {
			t.Fatal("lost legacy row during boot")
		}
	}
}
