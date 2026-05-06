package operation_log_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/db"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://cbuser:cbpassword@localhost:5433/circular_board_test?sslmode=disable"
	}

	if os.Getenv("MIGRATIONS_PATH") == "" {
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			backendDir := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
			_ = os.Setenv("MIGRATIONS_PATH", filepath.Join(backendDir, "db", "migrations"))
		}
	}

	var err error
	testPool, err = connectTestDB(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "テストDB接続失敗: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	if err := db.RunMigrations(context.Background(), testPool); err != nil {
		fmt.Fprintf(os.Stderr, "マイグレーション失敗: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func connectTestDB(dsn string) (*pgxpool.Pool, error) {
	for i := range 5 {
		pool, err := pgxpool.New(context.Background(), dsn)
		if err == nil {
			if pingErr := pool.Ping(context.Background()); pingErr == nil {
				return pool, nil
			}
			pool.Close()
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, fmt.Errorf("テストDB接続タイムアウト")
}

// ─── テストデータ挿入ヘルパー ────────────────────────────────

func insertTestUser(t *testing.T, name, email, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)
	require.NoError(t, err)
	now := time.Now()
	_, err = testPool.Exec(context.Background(),
		`INSERT INTO users (id, association_id, name, email, password_hash, role, is_active, created_at, updated_at)
		 VALUES ($1, NULL, $2, $3, $4, $5, true, $6, $7)`,
		id, name, email, string(hash), role, now, now,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func strPtr(s string) *string { return &s }
