package runtimecontrol_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	apiEnt "github.com/rayip/rayip/services/api/ent"
	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestEntRepositoryFirstChangeStartsAtSeqOne(t *testing.T) {
	dsn := os.Getenv("RAYIP_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("RAYIP_TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	client := apiEnt.NewClient(apiEnt.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	t.Cleanup(func() { _ = client.Close() })

	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	accountID := "acct-first-seq"
	nodeID := "node-first-seq"
	cleanupRuntimeControlRows(t, sqlDB, accountID, nodeID)
	t.Cleanup(func() { cleanupRuntimeControlRows(t, sqlDB, accountID, nodeID) })

	svc := runtimecontrol.NewService(runtimecontrol.NewEntRepository(client), func() time.Time { return now })
	result, err := svc.UpsertProxyAccount(ctx, runtimecontrol.ResourceInput{
		ProxyAccountID: accountID,
		NodeID:         nodeID,
		RuntimeEmail:   accountID,
		Protocol:       runtimecontrol.ProtocolSOCKS5,
		ListenIP:       "127.0.0.1",
		Port:           18080,
		Username:       "user",
		Password:       "pass",
	})
	if err != nil {
		t.Fatalf("UpsertProxyAccount() error = %v", err)
	}
	if result.Change.Seq != 1 {
		t.Fatalf("first change seq = %d, want 1", result.Change.Seq)
	}
}

func cleanupRuntimeControlRows(t *testing.T, db *sql.DB, accountID string, nodeID string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `delete from outbox_events where aggregate_key = $1`, nodeID); err != nil {
		t.Fatalf("cleanup outbox_events error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `delete from runtime_change_log where node_id = $1`, nodeID); err != nil {
		t.Fatalf("cleanup runtime_change_log error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `delete from runtime_account_states where proxy_account_id = $1`, accountID); err != nil {
		t.Fatalf("cleanup runtime_account_states error = %v", err)
	}
}
