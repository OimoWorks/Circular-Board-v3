package files_test

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
	"circular-board/internal/files"
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
			// setup_test.go が backend/internal/files/ にある
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
		UploadDir:          os.TempDir(),
		MaxUploadBytes:     10 << 20, // 10MB
	}
}

// ─── テストダミーデータ ──────────────────────────────────────

// dummyPDF は最小限のPDFバイト列（マジックバイトで始まる）
var dummyPDF = func() []byte {
	b := []byte("%PDF-1.4\n%EOF\n")
	b = append(b, make([]byte, 64)...)
	return b
}()

// dummyJPEG はJPEGマジックバイト
var dummyJPEG = []byte{
	0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
}

// dummyPNG はPNGマジックバイト
var dummyPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A,
}

// ─── テストデータ挿入ヘルパー ────────────────────────────────

// insertTestAssociation は自治会を挿入しテスト後に削除する
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

// insertTestUser はユーザーを挿入しテスト後に削除する
func insertTestUser(t *testing.T, assocID *uuid.UUID, name, email, password, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // テスト高速化のため MinCost
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

// insertTestFile はファイルレコードをDBに直接挿入しテスト後に削除する。
// 物理ファイルは作成しない（ダウンロードテストには insertTestFileOnDisk を使うこと）。
func insertTestFile(t *testing.T, assocID, uploadedBy uuid.UUID, year, month int, originalFilename, mimeType string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ext := files.ExtensionForMIME(mimeType)
	storedName := id.String() + ext
	storagePath := filepath.Join(os.TempDir(), storedName)
	now := time.Now()

	_, err := testPool.Exec(context.Background(),
		`INSERT INTO files
			(id, association_id, year, month, filename, original_filename,
			 storage_path, uploaded_by, file_size, mime_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, assocID, year, month, storedName, originalFilename,
		storagePath, uploadedBy, int64(len(dummyPDF)), mimeType, now,
	)
	require.NoError(t, err, "ファイルレコードの挿入に失敗")
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, id)
	})
	return id
}

// insertTestFileOnDisk はDBレコードの挿入＋物理ファイルの作成を行い、
// テスト後に両方をクリーンアップする。ダウンロードハンドラーのテストに使う。
func insertTestFileOnDisk(t *testing.T, assocID, uploadedBy uuid.UUID, year, month int, originalFilename, mimeType string, content []byte) (fileID uuid.UUID, storagePath string) {
	t.Helper()
	id := uuid.New()
	ext := files.ExtensionForMIME(mimeType)
	storedName := id.String() + ext
	path := filepath.Join(os.TempDir(), storedName)
	now := time.Now()

	// 物理ファイルを作成
	require.NoError(t, os.WriteFile(path, content, 0o644), "テストファイルの書き込みに失敗")

	_, err := testPool.Exec(context.Background(),
		`INSERT INTO files
			(id, association_id, year, month, filename, original_filename,
			 storage_path, uploaded_by, file_size, mime_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, assocID, year, month, storedName, originalFilename,
		path, uploadedBy, int64(len(content)), mimeType, now,
	)
	require.NoError(t, err, "ファイルレコードの挿入に失敗")

	t.Cleanup(func() {
		_ = os.Remove(path)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, id)
	})
	return id, path
}

// ─── テスト用ルーター ─────────────────────────────────────────

type testDeps struct {
	router   http.Handler
	filesSvc *files.Service
	cfg      *config.Config
}

func newTestDeps() *testDeps {
	cfg := testConfig()
	fileRepo := files.NewRepository(testPool)
	filesSvc := files.NewService(fileRepo, cfg)
	filesHandler := files.NewHandler(filesSvc)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Get("/", filesHandler.List)
		r.Get("/years", filesHandler.AvailableYears)
		r.Get("/{id}/download", filesHandler.Download)

		r.Group(func(r chi.Router) {
			r.Use(authMW.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
			r.Post("/", filesHandler.Upload)
			r.Delete("/{id}", filesHandler.Delete)
		})
	})

	return &testDeps{router: r, filesSvc: filesSvc, cfg: cfg}
}

// ─── JWTトークン生成ヘルパー ─────────────────────────────────

// makeTestToken はテスト用のアクセストークンを生成する
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

// doMultipartUpload はマルチパートフォームのPOSTリクエストを構築してルーターに送る
func doMultipartUpload(
	t *testing.T,
	router http.Handler,
	path string,
	fileContent []byte,
	filename, mimeType string,
	extraFields map[string]string,
	bearerToken string,
) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// ファイルパート
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)
	fw, err := mw.CreatePart(h)
	require.NoError(t, err)
	_, err = fw.Write(fileContent)
	require.NoError(t, err)

	// 追加フィールド（year, month など）
	for k, v := range extraFields {
		require.NoError(t, mw.WriteField(k, v))
	}
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
