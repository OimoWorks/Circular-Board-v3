package notice_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════
// POST /api/v1/notices（お知らせ登録）
// ═══════════════════════════════════════════════════════════

// TestNoticeCreateHandler_Success_Admin
// association_adminが登録できる
func TestNoticeCreateHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "登録ハンドラー自治会", "HDL_NOT_C1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-c1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{
			"title":     "テスト告知",
			"body":      "テスト本文です。",
			"is_pinned": false,
		},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "テスト告知", data["title"])
	assert.Equal(t, "テスト本文です。", data["body"])
	assert.Equal(t, false, data["is_pinned"])
	assert.NotEmpty(t, data["id"])

	t.Cleanup(func() {
		noticeID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, noticeID)
	})
}

// TestNoticeCreateHandler_Success_WithPinned
// ピン留めフラグが正しく登録される
func TestNoticeCreateHandler_Success_WithPinned(t *testing.T) {
	assocID := insertTestAssociation(t, "ピン登録ハンドラー自治会", "HDL_NOT_CPIN")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-cpin.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{
			"title":     "重要告知",
			"body":      "重要な内容",
			"is_pinned": true,
		},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_pinned"])

	t.Cleanup(func() {
		noticeID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, noticeID)
	})
}

// TestNoticeCreateHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeCreateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "test", "body": "body"},
		"" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestNoticeCreateHandler_Forbidden_UserRole
// 一般ユーザーは不可（403）
func TestNoticeCreateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403登録自治会", "HDL_NOT_CUSR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-cusr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "test", "body": "body"},
		token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestNoticeCreateHandler_BadRequest_EmptyTitle
// タイトルなしはエラー（400）
func TestNoticeCreateHandler_BadRequest_EmptyTitle(t *testing.T) {
	assocID := insertTestAssociation(t, "タイトルなし自治会", "HDL_NOT_CNOTITLE")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-cnotitle.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "", "body": "本文あり"},
		token,
	)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

// TestNoticeCreateHandler_BadRequest_EmptyBody
// 本文なしはエラー（400）
func TestNoticeCreateHandler_BadRequest_EmptyBody(t *testing.T) {
	assocID := insertTestAssociation(t, "本文なし自治会", "HDL_NOT_CNOBODY")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-cnobody.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "タイトルあり", "body": ""},
		token,
	)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

// ═══════════════════════════════════════════════════════════
// DELETE /api/v1/notices/:id（お知らせ削除）
// ═══════════════════════════════════════════════════════════

// TestNoticeDeleteHandler_Success_Admin
// association_adminが削除できる
func TestNoticeDeleteHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "削除ハンドラー自治会", "HDL_NOT_D1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-d1.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "削除対象", "削除本文", false)
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "削除しました", data["message"])
}

// TestNoticeDeleteHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeDeleteHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "削除401自治会", "HDL_NOT_DUNAUTH")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-dunauth.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "対象", "本文", false)

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestNoticeDeleteHandler_Forbidden_UserRole
// 一般ユーザーは不可（403）
func TestNoticeDeleteHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "削除403自治会", "HDL_NOT_DUSR")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-dusr.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-dusr.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "対象", "本文", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestNoticeDeleteHandler_CrossTenant
// 他自治会のお知らせは不可（404）
func TestNoticeDeleteHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "HDL_NOT_DTA")
	assocB := insertTestAssociation(t, "削除テナントB", "HDL_NOT_DTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@hdl-not-dta.test", "pass123", "association_admin")
	adminB := insertTestUser(t, &assocB, "B管理者", "b@hdl-not-dtb.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)
	tokenB := makeTestToken(t, assocB.String(), adminB.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, tokenB,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestNoticeDeleteHandler_NotFound
// 存在しないIDはエラー（404）
func TestNoticeDeleteHandler_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "削除不在自治会", "HDL_NOT_DNF")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-dnf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", uuid.New()),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "NOTICE_NOT_FOUND", errObj["code"])
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/notices（お知らせ一覧取得）
// ═══════════════════════════════════════════════════════════

// TestNoticeListHandler_Success_User
// 全ロールが取得できる
func TestNoticeListHandler_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧ハンドラー自治会", "HDL_NOT_L1")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-l1.test", "pass123", "user")
	_ = insertTestNotice(t, assocID, userID, "お知らせ1", "本文1", false)
	_ = insertTestNotice(t, assocID, userID, "お知らせ2", "本文2", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	assert.GreaterOrEqual(t, len(notices), 2)
}

// TestNoticeListHandler_PinnedFirst
// ピン留めが上部に来る
func TestNoticeListHandler_PinnedFirst(t *testing.T) {
	assocID := insertTestAssociation(t, "ピン順ハンドラー自治会", "HDL_NOT_LPIN")
	userID := insertTestUser(t, &assocID, "ユーザー", "user@hdl-not-lpin.test", "pass123", "user")
	_ = insertTestNotice(t, assocID, userID, "通常お知らせ", "本文", false)
	pinnedID := insertTestNotice(t, assocID, userID, "重要！", "重要本文", true)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices", nil, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	require.GreaterOrEqual(t, len(notices), 2)

	first := notices[0].(map[string]interface{})
	assert.Equal(t, pinnedID.String(), first["id"], "ピン留めされたお知らせが先頭に来るべき")
	assert.Equal(t, true, first["is_pinned"])
}

// TestNoticeListHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeListHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestNoticeListHandler_CrossTenant
// 他自治会のデータが混入しない
func TestNoticeListHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "一覧テナントA", "HDL_NOT_LTA")
	assocB := insertTestAssociation(t, "一覧テナントB", "HDL_NOT_LTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@hdl-not-lta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "Bユーザー", "b@hdl-not-ltb.test", "pass123", "user")
	_ = insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)
	_ = insertTestNotice(t, assocB, userB, "Bのお知らせ", "B本文", false)
	tokenA := makeTestToken(t, assocA.String(), userA.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices", nil, tokenA)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	for _, item := range notices {
		n := item.(map[string]interface{})
		assert.Equal(t, assocA.String(), n["association_id"],
			"自治会Aの一覧に自治会Bのお知らせが混入してはいけない")
	}
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/notices/:id（お知らせ詳細取得）
// ═══════════════════════════════════════════════════════════

// TestNoticeGetHandler_Success_User
// 全ロールが取得できる
func TestNoticeGetHandler_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "詳細ハンドラー自治会", "HDL_NOT_G1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-g1.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-g1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "詳細テスト", "詳細本文", false)
	tokenUser := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, tokenUser,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, noticeID.String(), data["id"])
	assert.Equal(t, "詳細テスト", data["title"])
	assert.Equal(t, "詳細本文", data["body"])

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeGetHandler_AutoMarkAsRead
// 取得時に既読になる
func TestNoticeGetHandler_AutoMarkAsRead(t *testing.T) {
	assocID := insertTestAssociation(t, "自動既読自治会", "HDL_NOT_GAUTO")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-gauto.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-gauto.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "自動既読テスト", "本文", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()

	// GET前はnotice_readsに記録なし
	var countBefore int
	require.NoError(t, testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notice_reads WHERE notice_id = $1 AND user_id = $2`,
		noticeID, userID).Scan(&countBefore))
	assert.Equal(t, 0, countBefore, "GET前はnotice_readsが0件")

	// GETリクエスト
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, token,
	)
	require.Equal(t, http.StatusOK, rr.Code)

	// GET後はnotice_readsに記録あり
	var countAfter int
	require.NoError(t, testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notice_reads WHERE notice_id = $1 AND user_id = $2`,
		noticeID, userID).Scan(&countAfter))
	assert.Equal(t, 1, countAfter, "GET後はnotice_readsに1件記録されるべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeGetHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeGetHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "詳細401自治会", "HDL_NOT_GUNAUTH")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-gunauth.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "対象", "本文", false)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestNoticeGetHandler_CrossTenant
// 他自治会のお知らせは不可（404）
func TestNoticeGetHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "詳細テナントA", "HDL_NOT_GTA")
	assocB := insertTestAssociation(t, "詳細テナントB", "HDL_NOT_GTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@hdl-not-gta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "Bユーザー", "b@hdl-not-gtb.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)
	tokenB := makeTestToken(t, assocB.String(), userB.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, tokenB,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/notices/:id/read（既読）
// ═══════════════════════════════════════════════════════════

// TestNoticeMarkAsReadHandler_Success_User
// 全ロールが既読にできる
func TestNoticeMarkAsReadHandler_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "既読ハンドラー自治会", "HDL_NOT_MAR1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-mar1.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-not-mar1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "既読対象", "本文", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost,
		fmt.Sprintf("/api/v1/notices/%s/read", noticeID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "既読にしました", data["message"])

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeMarkAsReadHandler_Idempotent
// 2回実行しても重複しない
func TestNoticeMarkAsReadHandler_Idempotent(t *testing.T) {
	assocID := insertTestAssociation(t, "重複既読ハンドラー自治会", "HDL_NOT_MARIO")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-mario.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "ユーザー", "user@hdl-not-mario.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "重複既読対象", "本文", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	path := fmt.Sprintf("/api/v1/notices/%s/read", noticeID)

	// 1回目
	rr1 := doRequest(t, deps.router, http.MethodPost, path, nil, token)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// 2回目（重複していても200が返るべき）
	rr2 := doRequest(t, deps.router, http.MethodPost, path, nil, token)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// DBのレコードは1件のみ
	var count int
	require.NoError(t, testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notice_reads WHERE notice_id = $1 AND user_id = $2`,
		noticeID, userID).Scan(&count))
	assert.Equal(t, 1, count, "2回既読にしてもnotice_readsは1件のまま")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeMarkAsReadHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeMarkAsReadHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "既読401自治会", "HDL_NOT_MARUNAUTH")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-marunauth.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "対象", "本文", false)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost,
		fmt.Sprintf("/api/v1/notices/%s/read", noticeID),
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/notices/unread-count（未読件数取得）
// ═══════════════════════════════════════════════════════════

// TestNoticeUnreadCountHandler_Success
// 全ロールが未読件数を取得できる
func TestNoticeUnreadCountHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "未読件数ハンドラー自治会", "HDL_NOT_UNRD1")
	userID := insertTestUser(t, &assocID, "ユーザー", "user@hdl-not-unrd1.test", "pass123", "user")
	_ = insertTestNotice(t, assocID, userID, "未読1", "本文1", false)
	_ = insertTestNotice(t, assocID, userID, "未読2", "本文2", false)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices/unread-count", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	count := data["unread_count"].(float64)
	assert.GreaterOrEqual(t, count, float64(2), "2件以上の未読件数が返るべき")
}

// TestNoticeUnreadCountHandler_Unauthorized
// 未認証は不可（401）
func TestNoticeUnreadCountHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices/unread-count", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DB権限チェック（RequireFeature ミドルウェア）
// ═══════════════════════════════════════════════════════════

// TestNoticeCreateHandler_PermAware_WithPermission
// DB権限あり（notices:create=true）のassociation_adminは登録できる（201）
func TestNoticeCreateHandler_PermAware_WithPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "権限あり登録自治会", "HDL_NOT_PA_C1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-pa-c1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "権限あり登録", "body": "本文"},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)

	t.Cleanup(func() {
		body := decodeBody(t, rr)
		if data, ok := body["data"].(map[string]interface{}); ok {
			if id, ok := data["id"].(string); ok {
				noticeID, _ := uuid.Parse(id)
				_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, noticeID)
			}
		}
	})
}

// TestNoticeCreateHandler_PermAware_NoPermission
// DB権限なし（notices:create=false）のassociation_adminは拒否される（403）
func TestNoticeCreateHandler_PermAware_NoPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "権限なし登録自治会", "HDL_NOT_PA_C2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-pa-c2.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, false, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/notices",
		map[string]interface{}{"title": "権限なし登録", "body": "本文"},
		token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestNoticeDeleteHandler_PermAware_WithPermission
// DB権限あり（notices:delete=true）のassociation_adminは削除できる（200）
func TestNoticeDeleteHandler_PermAware_WithPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "権限あり削除自治会", "HDL_NOT_PA_D1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-pa-d1.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "削除対象", "本文", false)
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestNoticeDeleteHandler_PermAware_NoPermission
// DB権限なし（notices:delete=false）のassociation_adminは拒否される（403）
func TestNoticeDeleteHandler_PermAware_NoPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "権限なし削除自治会", "HDL_NOT_PA_D2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-not-pa-d2.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "削除対象", "本文", false)
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, false, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/notices/%s", noticeID),
		nil, token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestNoticeListHandler_PermAware_NoViewPermission
// DB権限なし（notices:view=false）の場合は一覧取得を拒否される（403）
func TestNoticeListHandler_PermAware_NoViewPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "閲覧権限なし自治会", "HDL_NOT_PA_L1")
	userID := insertTestUser(t, &assocID, "ユーザー", "user@hdl-not-pa-l1.test", "pass123", "user_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "user_admin")

	roleID := getRoleIDByName(t, "user_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, false, false, false, false, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/notices", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
