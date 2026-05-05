package home_test

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
	"circular-board/internal/home"
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

// insertTestNotice はお知らせを挿入する
func insertTestNotice(t *testing.T, assocID, createdBy uuid.UUID, title string, isPinned bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO notices (id, association_id, title, body, is_pinned, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, 'テスト本文', $4, $5, $6, $7)`,
		id, assocID, title, isPinned, createdBy, now, now,
	)
	require.NoError(t, err, "お知らせの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notice_reads WHERE notice_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, id)
	})
	return id
}

// insertTestNoticeDeleted は論理削除済みお知らせを挿入する
func insertTestNoticeDeleted(t *testing.T, assocID, createdBy uuid.UUID, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO notices (id, association_id, title, body, is_pinned, created_by, created_at, updated_at, deleted_at)
		 VALUES ($1, $2, $3, '本文', false, $4, $5, $6, $7)`,
		id, assocID, title, createdBy, now, now, now,
	)
	require.NoError(t, err, "削除済みお知らせの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, id)
	})
	return id
}

// markNoticeRead はお知らせを既読にする
func markNoticeRead(t *testing.T, noticeID, userID uuid.UUID) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO notice_reads (id, notice_id, user_id, read_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (notice_id, user_id) DO NOTHING`,
		uuid.New(), noticeID, userID, time.Now(),
	)
	require.NoError(t, err, "既読マークの挿入に失敗")
}

// insertTestFile はファイルレコードを挿入する
func insertTestFile(t *testing.T, assocID, uploadedBy uuid.UUID, year, month int, originalFilename string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	storedName := uuid.New().String() + ".pdf"
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO files (id, association_id, year, month, filename, original_filename,
		  storage_path, uploaded_by, file_size, mime_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, '/dev/null', $7, 1024, 'application/pdf', $8)`,
		id, assocID, year, month, storedName, originalFilename, uploadedBy, now,
	)
	require.NoError(t, err, "ファイルの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, id)
	})
	return id
}

// insertTestFileDeleted は論理削除済みファイルを挿入する
func insertTestFileDeleted(t *testing.T, assocID, uploadedBy uuid.UUID, year, month int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	storedName := uuid.New().String() + ".pdf"
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO files (id, association_id, year, month, filename, original_filename,
		  storage_path, uploaded_by, file_size, mime_type, created_at, deleted_at)
		 VALUES ($1, $2, $3, $4, $5, 'deleted.pdf', '/dev/null', $6, 1024, 'application/pdf', $7, $8)`,
		id, assocID, year, month, storedName, uploadedBy, now, now,
	)
	require.NoError(t, err, "削除済みファイルの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, id)
	})
	return id
}

// insertTestSurvey はアンケートを挿入する（質問・選択肢なし）
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
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_answers WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_choices WHERE question_id IN (SELECT id FROM survey_questions WHERE survey_id = $1)`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE survey_id = $1`, id)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, id)
	})
	return id
}

// insertTestSurveyDeleted は論理削除済みアンケートを挿入する
func insertTestSurveyDeleted(t *testing.T, assocID, createdBy uuid.UUID, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO surveys (id, association_id, title, description, expires_at, created_by,
		  created_at, updated_at, deleted_at)
		 VALUES ($1, $2, $3, '', NOW() + INTERVAL '1 day', $4, $5, $6, $7)`,
		id, assocID, title, createdBy, now, now, now,
	)
	require.NoError(t, err, "削除済みアンケートの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, id)
	})
	return id
}

// markSurveyAnswered はアンケートを回答済みにする（質問・選択肢・回答を最低限挿入）
func markSurveyAnswered(t *testing.T, surveyID, userID uuid.UUID) {
	t.Helper()
	qID := uuid.New()
	cID := uuid.New()
	aID := uuid.New()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO survey_questions (id, survey_id, question_text, question_type, sort_order)
		 VALUES ($1, $2, 'テスト質問', 'single', 1)`,
		qID, surveyID,
	)
	require.NoError(t, err, "質問の挿入に失敗")
	_, err = testPool.Exec(context.Background(),
		`INSERT INTO survey_choices (id, question_id, choice_text, sort_order)
		 VALUES ($1, $2, '選択肢A', 1)`,
		cID, qID,
	)
	require.NoError(t, err, "選択肢の挿入に失敗")
	_, err = testPool.Exec(context.Background(),
		`INSERT INTO survey_answers (id, survey_id, user_id, question_id, choice_id, answered_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		aID, surveyID, userID, qID, cID, time.Now(),
	)
	require.NoError(t, err, "回答の挿入に失敗")
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
	router  http.Handler
	homeSvc *home.Service
}

func newTestDeps() *testDeps {
	cfg := testConfig()
	homeRepo := home.NewRepository(testPool)
	homeSvc := home.NewService(homeRepo)
	homeHandler := home.NewHandler(homeSvc)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	r := chi.NewRouter()
	r.With(authMW.Authenticate).Get("/api/v1/home", homeHandler.GetHomeData)

	return &testDeps{router: r, homeSvc: homeSvc}
}

// ─── HTTPテストヘルパー ──────────────────────────────────────

func doRequest(t *testing.T, router http.Handler, method, path string, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
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

func doRequestWithBody(t *testing.T, router http.Handler, method, path string, body interface{}, bearerToken string) *httptest.ResponseRecorder {
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
