package auth_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════
// POST /api/v1/auth/login
// ═══════════════════════════════════════════════════════════

// TestLoginHandler_Success
// 正しい認証情報でログインすると200とトークン・ユーザー情報が返る
func TestLoginHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "ハンドラーログイン自治会", "HDL_LOGIN_1")
	_ = insertTestUser(t, &assocID, "ログインさん", "login@hdl-1.test", "handler!pass", "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "HDL_LOGIN_1",
		"email":            "login@hdl-1.test",
		"password":         "handler!pass",
	}, "")

	assert.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.Equal(t, "Bearer", data["token_type"])

	user := data["user"].(map[string]interface{})
	assert.Equal(t, "login@hdl-1.test", user["email"])
	assert.Equal(t, "user", user["role"])
	assert.Equal(t, "ログインさん", user["name"])

	// リフレッシュトークンをクリーンアップ
	t.Cleanup(func() {
		rawAccess := data["access_token"].(string)
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, rawAccess)
	})
}

// TestLoginHandler_Success_AssociationAdmin
// 自治会管理者がログインするとロールが正しく返る
func TestLoginHandler_Success_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者ハンドラー自治会", "HDL_ADMIN_1")
	_ = insertTestUser(t, &assocID, "管理者さん", "admin@hdl-admin.test", "adminPass!", "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "HDL_ADMIN_1",
		"email":            "admin@hdl-admin.test",
		"password":         "adminPass!",
	}, "")

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	assert.Equal(t, "association_admin", user["role"])

	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, data["access_token"].(string))
	})
}

// TestLoginHandler_MissingEmail
// メールアドレスが空の場合は400 INVALID_REQUEST
func TestLoginHandler_MissingEmail(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "ANY",
		"email":            "",
		"password":         "somepass",
	}, "")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST", errObj["code"])
}

// TestLoginHandler_MissingPassword
// パスワードが空の場合は400 INVALID_REQUEST
func TestLoginHandler_MissingPassword(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "ANY",
		"email":            "test@test.test",
		"password":         "",
	}, "")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST", errObj["code"])
}

// TestLoginHandler_InvalidJSON
// JSONが不正な場合は400 INVALID_REQUEST
func TestLoginHandler_InvalidJSON(t *testing.T) {
	deps := newTestDeps()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{not valid json}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	deps.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestLoginHandler_WrongCredentials
// パスワード不一致は401 INVALID_CREDENTIALS
func TestLoginHandler_WrongCredentials(t *testing.T) {
	assocID := insertTestAssociation(t, "認証失敗自治会", "HDL_BADCRED")
	_ = insertTestUser(t, &assocID, "パスワードさん", "badcred@hdl.test", "correctPass", "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "HDL_BADCRED",
		"email":            "badcred@hdl.test",
		"password":         "wrongPass",
	}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CREDENTIALS", errObj["code"])
}

// TestLoginHandler_WrongAssociationCode
// 自治会コード不一致は401 INVALID_CREDENTIALS（テナント境界の確認）
func TestLoginHandler_WrongAssociationCode(t *testing.T) {
	assocID := insertTestAssociation(t, "正規自治会HDL", "HDL_REAL_ASSOC")
	_ = insertTestUser(t, &assocID, "テナントさん", "tenant@hdl-real.test", "pass123", "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"association_code": "HDL_FAKE_ASSOC",
		"email":            "tenant@hdl-real.test",
		"password":         "pass123",
	}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CREDENTIALS", errObj["code"])
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/auth/me
// ═══════════════════════════════════════════════════════════

// TestMeHandler_Success
// 有効なトークンでユーザー情報が取得できる
func TestMeHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "Me自治会", "HDL_ME_1")
	_ = insertTestUser(t, &assocID, "Meさん", "me@hdl-me.test", "pass123", "user")

	deps := newTestDeps()
	accessToken, refreshToken := loginAndGetTokens(t, deps, "HDL_ME_1", "me@hdl-me.test", "pass123")
	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
		_ = refreshToken
	})

	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, accessToken)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "me@hdl-me.test", data["email"])
	assert.Equal(t, "Meさん", data["name"])
	assert.Equal(t, "user", data["role"])
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["association_id"])
}

// TestMeHandler_Unauthorized_NoToken
// トークンなしは401 UNAUTHORIZED
func TestMeHandler_Unauthorized_NoToken(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errObj["code"])
}

// TestMeHandler_Unauthorized_InvalidToken
// 不正なトークン文字列は401 UNAUTHORIZED
func TestMeHandler_Unauthorized_InvalidToken(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, "invalid.token.here")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestMeHandler_Unauthorized_ExpiredToken
// 期限切れトークンは401 UNAUTHORIZED
func TestMeHandler_Unauthorized_ExpiredToken(t *testing.T) {
	deps := newTestDeps()
	expiredToken := createExpiredAccessToken(t, deps.cfg, uuid.New().String(), uuid.New().String(), "user")
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, expiredToken)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/auth/logout
// ═══════════════════════════════════════════════════════════

// TestLogoutHandler_Success
// ログアウト後はリフレッシュトークンが無効になる
func TestLogoutHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "ログアウトHDL自治会", "HDL_LOGOUT_1")
	_ = insertTestUser(t, &assocID, "ログアウトさん", "logout@hdl-logout.test", "pass123", "user")

	deps := newTestDeps()
	accessToken, refreshToken := loginAndGetTokens(t, deps, "HDL_LOGOUT_1", "logout@hdl-logout.test", "pass123")

	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// ログアウト後はリフレッシュが失敗する
	rrRefresh := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": refreshToken}, "")
	assert.Equal(t, http.StatusUnauthorized, rrRefresh.Code,
		"ログアウト後はリフレッシュトークンが無効になる")
}

// TestLogoutHandler_Unauthorized
// 未認証でのログアウトは401
func TestLogoutHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/auth/refresh
// ═══════════════════════════════════════════════════════════

// TestRefreshHandler_Success
// 有効なリフレッシュトークンで新しいアクセストークンが取得できる
func TestRefreshHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "リフレッシュHDL自治会", "HDL_REFRESH_1")
	_ = insertTestUser(t, &assocID, "リフレッシュさん", "refresh@hdl-refresh.test", "pass123", "user")

	deps := newTestDeps()
	firstAccess, firstRefresh := loginAndGetTokens(t, deps, "HDL_REFRESH_1", "refresh@hdl-refresh.test", "pass123")

	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": firstRefresh}, "")

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	newAccess := data["access_token"].(string)
	newRefresh := data["refresh_token"].(string)

	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	assert.NotEqual(t, firstAccess, newAccess, "新しいアクセストークンが発行される")
	assert.NotEqual(t, firstRefresh, newRefresh, "リフレッシュトークンがローテーションされる")

	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, newAccess)
	})
}

// TestRefreshHandler_InvalidToken
// 存在しないリフレッシュトークンは401 INVALID_TOKEN
func TestRefreshHandler_InvalidToken(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": "this-token-does-not-exist"}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_TOKEN", errObj["code"])
}

// TestRefreshHandler_ExpiredToken
// DBに期限切れで保存されたリフレッシュトークンは401 INVALID_TOKEN
func TestRefreshHandler_ExpiredToken(t *testing.T) {
	assocID := insertTestAssociation(t, "期限切れHDL自治会", "HDL_REFEXP")
	userID := insertTestUser(t, &assocID, "期限切れさん", "exp@hdl-refexp.test", "pass123", "user")

	rawToken := "handler-expired-" + uuid.New().String()
	insertExpiredRefreshToken(t, userID, rawToken)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": rawToken}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_TOKEN", errObj["code"])
}

// TestRefreshHandler_MissingBody
// refresh_token が空のボディは400 INVALID_REQUEST
func TestRefreshHandler_MissingBody(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": ""}, "")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// RequireRole ミドルウェア（/admin-only テストルート）
// ═══════════════════════════════════════════════════════════

// TestRequireRoleMiddleware_AssociationAdmin_Allowed
// association_admin は管理者専用ルートにアクセスできる
func TestRequireRoleMiddleware_AssociationAdmin_Allowed(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者ミドル自治会", "HDL_ROLE_ADM")
	_ = insertTestUser(t, &assocID, "管理者ミドルさん", "admin@hdl-role.test", "pass123", "association_admin")

	deps := newTestDeps()
	accessToken, _ := loginAndGetTokens(t, deps, "HDL_ROLE_ADM", "admin@hdl-role.test", "pass123")
	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	})

	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/admin-only", nil, accessToken)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRequireRoleMiddleware_SystemAdmin_Allowed
// system_admin は管理者専用ルートにアクセスできる
func TestRequireRoleMiddleware_SystemAdmin_Allowed(t *testing.T) {
	_ = insertTestUser(t, nil, "システム管理者HDL", "sysadmin@hdl-role.test", "pass123", "system_admin")

	deps := newTestDeps()
	accessToken, _ := loginAndGetTokens(t, deps, "", "sysadmin@hdl-role.test", "pass123")
	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	})

	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/admin-only", nil, accessToken)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRequireRoleMiddleware_User_Forbidden
// user は管理者専用ルートに403 FORBIDDEN でアクセス拒否される
func TestRequireRoleMiddleware_User_Forbidden(t *testing.T) {
	assocID := insertTestAssociation(t, "一般ユーザーミドル自治会", "HDL_ROLE_USR")
	_ = insertTestUser(t, &assocID, "一般さん", "user@hdl-role.test", "pass123", "user")

	deps := newTestDeps()
	accessToken, _ := loginAndGetTokens(t, deps, "HDL_ROLE_USR", "user@hdl-role.test", "pass123")
	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	})

	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/admin-only", nil, accessToken)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errObj["code"])
}

// TestRequireRoleMiddleware_NoToken_Unauthorized
// トークンなしは401 UNAUTHORIZED
func TestRequireRoleMiddleware_NoToken_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/admin-only", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// マルチテナント境界の確認
// ═══════════════════════════════════════════════════════════

// TestCrossAssociation_Tokens_NoMixup
// 異なる自治会のユーザーが互いの情報にアクセスできないこと
func TestCrossAssociation_Tokens_NoMixup(t *testing.T) {
	assocA := insertTestAssociation(t, "クロステナントA", "HDL_CROSS_A")
	assocB := insertTestAssociation(t, "クロステナントB", "HDL_CROSS_B")
	_ = insertTestUser(t, &assocA, "テナントAさん", "a@hdl-cross.test", "pass123", "user")
	_ = insertTestUser(t, &assocB, "テナントBさん", "b@hdl-cross.test", "pass123", "user")

	deps := newTestDeps()
	tokenA, _ := loginAndGetTokens(t, deps, "HDL_CROSS_A", "a@hdl-cross.test", "pass123")
	tokenB, _ := loginAndGetTokens(t, deps, "HDL_CROSS_B", "b@hdl-cross.test", "pass123")

	t.Cleanup(func() {
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, tokenA)
		_ = doRequest(t, deps.router, http.MethodPost, "/api/v1/auth/logout", nil, tokenB)
	})

	// トークンAで /me → テナントAの情報が返る
	rrA := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, tokenA)
	require.Equal(t, http.StatusOK, rrA.Code)
	dataA := decodeBody(t, rrA)["data"].(map[string]interface{})
	assert.Equal(t, "a@hdl-cross.test", dataA["email"])

	// トークンBで /me → テナントBの情報が返る
	rrB := doRequest(t, deps.router, http.MethodGet, "/api/v1/auth/me", nil, tokenB)
	require.Equal(t, http.StatusOK, rrB.Code)
	dataB := decodeBody(t, rrB)["data"].(map[string]interface{})
	assert.Equal(t, "b@hdl-cross.test", dataB["email"])

	// 互いの association_id が異なる
	assert.NotEqual(t, dataA["association_id"], dataB["association_id"],
		"異なる自治会のユーザーは異なる association_id を持つ")
}
