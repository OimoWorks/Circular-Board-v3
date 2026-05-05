package association_test

// ハンドラーテスト。
// ルーティング・認証・ロールチェック（system_admin 専用）・バリデーション・HTTP ステータスコードを検証する。

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
// GET /api/v1/associations（自治会一覧）
// ═══════════════════════════════════════════════════════════

// TestAssocListHandler_SystemAdmin_Success
// system_admin が全自治会一覧を取得できる (200)
func TestAssocListHandler_SystemAdmin_Success(t *testing.T) {
	id1 := insertTestAssociation(t, "一覧ハンドラー自治会1", "HDLA_LST_SA1")
	id2 := insertTestAssociation(t, "一覧ハンドラー自治会2", "HDLA_LST_SA2")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-lst-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assocs := data["associations"].([]interface{})

	ids := make(map[string]bool)
	for _, item := range assocs {
		a := item.(map[string]interface{})
		ids[a["id"].(string)] = true
	}
	assert.True(t, ids[id1.String()], "自治会1が含まれるべき")
	assert.True(t, ids[id2.String()], "自治会2が含まれるべき")
}

// TestAssocListHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocListHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestAssocListHandler_Forbidden_AssociationAdmin
// association_admin は 403 が返る（system_admin 専用）
func TestAssocListHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "403自治会管理者自治会", "HDLA_LST_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-lst-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/associations（自治会作成）
// ═══════════════════════════════════════════════════════════

// TestAssocCreateHandler_SystemAdmin_Success
// system_admin が自治会を作成できる (201)
func TestAssocCreateHandler_SystemAdmin_Success(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-crt-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "新規自治会",
		"code": "HDLA_CRT_NEW",
	}, token)

	require.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "新規自治会", data["name"])
	assert.Equal(t, "HDLA_CRT_NEW", data["code"])
	assert.Equal(t, true, data["is_active"])

	t.Cleanup(func() {
		id, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, id)
	})
}

// TestAssocCreateHandler_CodeConflict
// コード重複は 409 が返る
func TestAssocCreateHandler_CodeConflict(t *testing.T) {
	_ = insertTestAssociation(t, "重複コード自治会", "HDLA_CRT_DUP")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-crt-dup.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "別名",
		"code": "HDLA_CRT_DUP",
	}, token)

	assert.Equal(t, http.StatusConflict, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "CODE_CONFLICT", errData["code"])
}

// TestAssocCreateHandler_InvalidCode
// 無効なコード形式は 400 が返る
func TestAssocCreateHandler_InvalidCode(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-crt-inv.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "無効コード自治会",
		"code": "invalid code!", // スペース・記号は不可
	}, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CODE", errData["code"])
}

// TestAssocCreateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocCreateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "テスト",
		"code": "TEST_CODE",
	}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestAssocCreateHandler_Forbidden_AssociationAdmin
// association_admin は 403 が返る
func TestAssocCreateHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "403作成自治会", "HDLA_CRT_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-crt-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "新規自治会",
		"code": "HDLA_CRT_FADM",
	}, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/associations/:id（自治会編集）
// ═══════════════════════════════════════════════════════════

// TestAssocUpdateHandler_SystemAdmin_Success
// system_admin が自治会を更新できる (200)
func TestAssocUpdateHandler_SystemAdmin_Success(t *testing.T) {
	id := insertTestAssociation(t, "更新前自治会", "HDLA_UPD_SA")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-upd-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s", id),
		map[string]interface{}{
			"name": "更新後自治会",
			"code": "HDLA_UPD_SANEW",
		}, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "更新後自治会", data["name"])
	assert.Equal(t, "HDLA_UPD_SANEW", data["code"])

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE code = 'HDLA_UPD_SANEW'`)
	})
}

// TestAssocUpdateHandler_NotFound
// 存在しない ID は 404 が返る
func TestAssocUpdateHandler_NotFound(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-upd-nf.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s", uuid.New()),
		map[string]interface{}{
			"name": "名前",
			"code": "HDLA_UPD_NF",
		}, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestAssocUpdateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocUpdateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s", uuid.New()),
		map[string]interface{}{
			"name": "名前",
			"code": "HDLA_UPD_UNAUTH",
		}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DELETE /api/v1/associations/:id（自治会無効化）
// ═══════════════════════════════════════════════════════════

// TestAssocDeleteHandler_SystemAdmin_Success
// system_admin が自治会を無効化できる (200)
func TestAssocDeleteHandler_SystemAdmin_Success(t *testing.T) {
	id := insertTestAssociation(t, "削除ハンドラー自治会", "HDLA_DEL_SA")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-del-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/associations/%s", id),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestAssocDeleteHandler_NotFound
// 存在しない ID は 404 が返る
func TestAssocDeleteHandler_NotFound(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-del-nf.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/associations/%s", uuid.New()),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestAssocDeleteHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocDeleteHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/associations/%s", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/associations/:id/activate（有効化）
// ═══════════════════════════════════════════════════════════

// TestAssocActivateHandler_Success
// 無効化された自治会を有効化できる (200)
func TestAssocActivateHandler_Success(t *testing.T) {
	id := insertTestAssociation(t, "有効化ハンドラー自治会", "HDLA_ACT_SA")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-act-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	// まず無効化
	_, err := testPool.Exec(context.Background(),
		`UPDATE associations SET is_active = false WHERE id = $1`, id)
	require.NoError(t, err)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/activate", id),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestAssocActivateHandler_NotFound
// 存在しない ID は 404 が返る
func TestAssocActivateHandler_NotFound(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-act-nf.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/activate", uuid.New()),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestAssocActivateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocActivateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/activate", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/associations/:id/deactivate（無効化）
// ═══════════════════════════════════════════════════════════

// TestAssocDeactivateHandler_Success
// 有効な自治会を無効化できる (200)
func TestAssocDeactivateHandler_Success(t *testing.T) {
	id := insertTestAssociation(t, "無効化ハンドラー自治会", "HDLA_DACT_SA")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-dact-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/deactivate", id),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestAssocDeactivateHandler_NotFound
// 存在しない ID は 404 が返る
func TestAssocDeactivateHandler_NotFound(t *testing.T) {
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdla-dact-nf.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/deactivate", uuid.New()),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestAssocDeactivateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestAssocDeactivateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/deactivate", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// 追加テスト：ロール別アクセス制御
// ═══════════════════════════════════════════════════════════

// TestAssocListHandler_Forbidden_UserRole
// 一般ユーザーロールは 403 が返る
func TestAssocListHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "一般403一覧自治会", "HDLA_LST_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdla-lst-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestAssocCreateHandler_Forbidden_UserRole
// 一般ユーザーロールは 403 が返る
func TestAssocCreateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "一般403作成自治会", "HDLA_CRT_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdla-crt-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/associations", map[string]interface{}{
		"name": "新規自治会",
		"code": "HDLA_CRT_USR_NEW",
	}, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestAssocUpdateHandler_Forbidden_AssociationAdmin
// association_admin は 403 が返る
func TestAssocUpdateHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "更新403自治会", "HDLA_UPD_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-upd-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s", assocID),
		map[string]interface{}{
			"name": "更新後",
			"code": "HDLA_UPD_ADM2",
		}, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestAssocDeleteHandler_Forbidden_AssociationAdmin
// association_admin は 403 が返る
func TestAssocDeleteHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "削除403自治会", "HDLA_DEL_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-del-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/associations/%s", assocID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestAssocActivateHandler_Forbidden_AssociationAdmin
// association_admin は有効化で 403 が返る
func TestAssocActivateHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "有効化403自治会", "HDLA_ACT_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-act-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/activate", assocID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestAssocDeactivateHandler_Forbidden_AssociationAdmin
// association_admin は無効化で 403 が返る
func TestAssocDeactivateHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "無効化403自治会", "HDLA_DACT_ADM")
	adminID := insertTestUser(t, &assocID, "自治会管理者", "admin@hdla-dact-adm.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/associations/%s/deactivate", assocID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
