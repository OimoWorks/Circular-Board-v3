package account_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/account"
	"circular-board/internal/config"
	"circular-board/internal/db"
	"circular-board/internal/domain"
	"circular-board/internal/middleware"
	"circular-board/internal/repository"
	"circular-board/internal/service"
)

// ─── グローバル ──────────────────────────────────────────────
var testPool *pgxpool.Pool

// ─── TestMain：DB接続・マイグレーション ─────────────────────
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

// ─── テスト用設定 ────────────────────────────────────────────
func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:          "test-jwt-secret-32bytes-padding!!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Env:                "test",
	}
}

// ─── テストデータ挿入ヘルパー ────────────────────────────────

// insertTestAssociation は自治会を挿入しテスト後に削除する
func insertTestAssociation(t *testing.T, name, code string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO associations (id, name, code, is_active, created_at, updated_at) VALUES ($1, $2, $3, true, $4, $5)`,
		id, name, code, now, now,
	)
	require.NoError(t, err, "自治会の挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, id)
	})
	return id
}

// insertTestUser はユーザーを挿入しテスト後に削除する
func insertTestUser(t *testing.T, assocID *uuid.UUID, name, email, password, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	now := time.Now()
	_, err = testPool.Exec(context.Background(),
		`INSERT INTO users (id, association_id, name, email, password_hash, role, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8)`,
		id, assocID, name, email, string(hash), role, now, now,
	)
	require.NoError(t, err, "ユーザーの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM refresh_tokens WHERE user_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// ─── JWTトークン生成ヘルパー ─────────────────────────────────

func makeTestToken(t *testing.T, assocID, userID, role string) string {
	t.Helper()
	cfg := testConfig()
	claims := service.Claims{
		UserID:        userID,
		AssociationID: assocID,
		Role:          role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	require.NoError(t, err)
	return signed
}

// ─── テスト用ルーター ─────────────────────────────────────────

type testDeps struct {
	router     http.Handler
	accountSvc *account.Service
}

func newTestDeps() *testDeps {
	cfg := testConfig()
	accountRepo := account.NewRepository(testPool)
	accountSvc := account.NewService(accountRepo)
	accountHandler := account.NewHandler(accountSvc)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/accounts", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
		r.Get("/", accountHandler.List)
		r.Post("/", accountHandler.Create)
		r.Put("/{id}", accountHandler.Update)
		r.Delete("/{id}", accountHandler.Delete)
		r.Put("/{id}/activate", accountHandler.Activate)
		r.Put("/{id}/deactivate", accountHandler.Deactivate)
	})

	return &testDeps{router: r, accountSvc: accountSvc}
}

// ─── HTTPテストヘルパー ──────────────────────────────────────

func doRequest(t *testing.T, router http.Handler, method, path string, body interface{}, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	return result
}
