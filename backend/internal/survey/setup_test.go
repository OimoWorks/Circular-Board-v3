package survey_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	"circular-board/internal/survey"
)

// ─── グローバル ──────────────────────────────────────────────
var (
	testPool      *pgxpool.Pool
	testUploadDir string
)

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

	testUploadDir, err = os.MkdirTemp("", "survey_test_uploads_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "テスト用アップロードディレクトリ作成失敗: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(testUploadDir)

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
		UploadDir:          testUploadDir,
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

func insertTestSurvey(t *testing.T, assocID, createdBy uuid.UUID, title string, expiresAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO surveys (id, association_id, title, description, expires_at, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, '', $4, $5, $6, $7)`,
		id, assocID, title, expiresAt, createdBy, now, now,
	)
	require.NoError(t, err, "アンケートの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_answers WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, id)
	})
	return id
}

func insertTestSurveyImage(t *testing.T, surveyID, assocID uuid.UUID, filename, storagePath string, sortOrder int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO survey_images (id, survey_id, association_id, filename, storage_path, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, surveyID, assocID, filename, storagePath, sortOrder, now,
	)
	require.NoError(t, err, "画像レコードの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE id = $1`, id)
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

// ─── 権限ヘルパー ─────────────────────────────────────────────

func getRoleIDByName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM roles WHERE name = $1`, name).Scan(&id)
	require.NoError(t, err, "ロールID取得失敗: "+name)
	return id
}

func getFeatureIDByName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM features WHERE name = $1`, name).Scan(&id)
	require.NoError(t, err, "機能ID取得失敗: "+name)
	return id
}

func withPermission(t *testing.T, roleID, featureID uuid.UUID, canView, canCreate, canEdit, canDelete bool, scope string) {
	t.Helper()
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

	_, err2 := testPool.Exec(context.Background(), `
		INSERT INTO role_permissions (role_id, feature_id, can_view, can_create, can_edit, can_delete, scope)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (role_id, feature_id) DO UPDATE SET
			can_view=$3, can_create=$4, can_edit=$5, can_delete=$6, scope=$7, updated_at=NOW()`,
		roleID, featureID, canView, canCreate, canEdit, canDelete, scope,
	)
	require.NoError(t, err2, "権限設定失敗")

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

// ─── テスト用ルーター ─────────────────────────────────────────

type testDeps struct {
	router    http.Handler
	surveySvc *survey.Service
}

func newTestDeps() *testDeps {
	cfg := testConfig()
	surveyRepo := survey.NewRepository(testPool)
	surveySvc := survey.NewService(surveyRepo)
	surveyHandler := survey.NewHandler(surveySvc, cfg)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/surveys", func(r chi.Router) {
		r.Use(authMW.Authenticate)

		r.Get("/unanswered-count", surveyHandler.UnansweredCount)
		r.Get("/", surveyHandler.List)
		r.Get("/{id}", surveyHandler.Get)
		r.Post("/{id}/answer", surveyHandler.Answer)

		r.Group(func(r chi.Router) {
			r.Use(authMW.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
			r.Post("/", surveyHandler.Create)
			r.Delete("/{id}", surveyHandler.Delete)
			r.Get("/{id}/results", surveyHandler.Results)
			r.Post("/{id}/images", surveyHandler.UploadImage)
			r.Delete("/{id}/images/{image_id}", surveyHandler.DeleteImage)
		})

		r.Get("/{id}/images/{image_id}", surveyHandler.GetImage)
	})

	return &testDeps{router: r, surveySvc: surveySvc}
}

// newPermAwareTestDeps は RequireFeature ミドルウェアを使うルーターを返す（DB権限チェックテスト用）
func newPermAwareTestDeps() *testDeps {
	cfg := testConfig()
	surveyRepo := survey.NewRepository(testPool)
	surveySvc := survey.NewService(surveyRepo)
	surveyHandler := survey.NewHandler(surveySvc, cfg)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	permRepo := permission.NewRepository(testPool)
	permSvc := permission.NewService(permRepo)
	permMW := middleware.NewPermissionMiddleware(permSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/surveys", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.With(permMW.RequireFeature("surveys", "view")).Get("/unanswered-count", surveyHandler.UnansweredCount)
		r.With(permMW.RequireFeature("surveys", "view")).Get("/", surveyHandler.List)
		r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}", surveyHandler.Get)
		r.With(permMW.RequireFeature("surveys", "view")).Post("/{id}/answer", surveyHandler.Answer)
		r.With(permMW.RequireFeature("surveys", "create")).Post("/", surveyHandler.Create)
		r.With(permMW.RequireFeature("surveys", "delete")).Delete("/{id}", surveyHandler.Delete)
		r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}/results", surveyHandler.Results)
		r.With(permMW.RequireFeature("surveys", "create")).Post("/{id}/images", surveyHandler.UploadImage)
		r.With(permMW.RequireFeature("surveys", "delete")).Delete("/{id}/images/{image_id}", surveyHandler.DeleteImage)
		r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}/images/{image_id}", surveyHandler.GetImage)
	})

	return &testDeps{router: r, surveySvc: surveySvc}
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

func doMultipartRequest(t *testing.T, router http.Handler, path string, imageData []byte, filename, mimeType, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)
	fw, err := mw.CreatePart(h)
	require.NoError(t, err)
	_, err = fw.Write(imageData)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
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

// minimalPNGBytes は 1x1 の最小 PNG バイト列
var minimalPNGBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR length + type
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // width=1, height=1
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, // bit depth, color type, ...
	0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, // IDAT length + type
	0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01,
	0xe2, 0x21, 0xbc, 0x33, // IDAT CRC
	0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82, // IEND
}
