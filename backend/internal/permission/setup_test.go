package permission_test

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

	"circular-board/internal/config"
	"circular-board/internal/db"
	"circular-board/internal/domain"
	"circular-board/internal/middleware"
	"circular-board/internal/permission"
	"circular-board/internal/repository"
	"circular-board/internal/service"
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

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:          "test-jwt-secret-32bytes-padding!!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Env:                "test",
	}
}

// ─── テストデータ挿入ヘルパー ────────────────────────────────

func insertTestAssociation(t *testing.T, name, code string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO associations (id, name, code, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		id, name, code, now, now,
	)
	require.NoError(t, err, "自治会の挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, id)
	})
	return id
}

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

// ─── 権限ヘルパー ─────────────────────────────────────────────

// getRoleIDByName はロール名からIDを取得する
func getRoleIDByName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM roles WHERE name = $1`, name).Scan(&id)
	require.NoError(t, err, "ロールID取得失敗: "+name)
	return id
}

// getFeatureIDByName は機能名からIDを取得する
func getFeatureIDByName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM features WHERE name = $1`, name).Scan(&id)
	require.NoError(t, err, "機能ID取得失敗: "+name)
	return id
}

// upsertPermission は権限を直接DBに設定する（テスト専用）
func upsertPermission(t *testing.T, roleID, featureID uuid.UUID, canView, canCreate, canEdit, canDelete bool, scope string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO role_permissions (role_id, feature_id, can_view, can_create, can_edit, can_delete, scope)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (role_id, feature_id) DO UPDATE SET
			can_view   = EXCLUDED.can_view,
			can_create = EXCLUDED.can_create,
			can_edit   = EXCLUDED.can_edit,
			can_delete = EXCLUDED.can_delete,
			scope      = EXCLUDED.scope,
			updated_at = NOW()`,
		roleID, featureID, canView, canCreate, canEdit, canDelete, scope,
	)
	require.NoError(t, err, "権限設定失敗")
}

// withPermission は権限を一時的に変更しテスト後に復元する
func withPermission(t *testing.T, roleID, featureID uuid.UUID, canView, canCreate, canEdit, canDelete bool, scope string) {
	t.Helper()

	// 現在値を保存
	var orig struct {
		canView, canCreate, canEdit, canDelete bool
		scope                                  string
		exists                                 bool
	}
	err := testPool.QueryRow(context.Background(),
		`SELECT can_view, can_create, can_edit, can_delete, scope
		 FROM role_permissions WHERE role_id = $1 AND feature_id = $2`,
		roleID, featureID,
	).Scan(&orig.canView, &orig.canCreate, &orig.canEdit, &orig.canDelete, &orig.scope)
	orig.exists = (err == nil)

	// 新しい値を設定
	upsertPermission(t, roleID, featureID, canView, canCreate, canEdit, canDelete, scope)

	// テスト後に復元
	t.Cleanup(func() {
		if orig.exists {
			_, _ = testPool.Exec(context.Background(), `
				UPDATE role_permissions SET
					can_view=$1, can_create=$2, can_edit=$3, can_delete=$4, scope=$5, updated_at=NOW()
				WHERE role_id=$6 AND feature_id=$7`,
				orig.canView, orig.canCreate, orig.canEdit, orig.canDelete, orig.scope, roleID, featureID,
			)
		} else {
			_, _ = testPool.Exec(context.Background(),
				`DELETE FROM role_permissions WHERE role_id=$1 AND feature_id=$2`, roleID, featureID)
		}
	})
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
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(cfg.JWTSecret))
	require.NoError(t, err)
	return signed
}

// ─── テスト用ルーター ─────────────────────────────────────────

type testDeps struct {
	router  http.Handler
	permSvc *permission.Service
}

func newTestDeps() *testDeps {
	cfg := testConfig()
	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	permRepo := permission.NewRepository(testPool)
	permSvc := permission.NewService(permRepo)
	permHandler := permission.NewHandler(permSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/permissions", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole(domain.RoleSystemAdmin))
		r.Get("/", permHandler.GetMatrix)
		r.Put("/", permHandler.UpdatePermissions)
		r.Get("/roles", permHandler.GetRoles)
		r.Get("/features", permHandler.GetFeatures)
		r.Post("/emergency-appointment", permHandler.EmergencyAppointment)
		r.Get("/logs", permHandler.GetOperationLogs)
	})

	return &testDeps{router: r, permSvc: permSvc}
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
