package account_test

// ハンドラーテスト。
// ルーティング・認証・ロールチェック・バリデーション・HTTP ステータスコードを検証する。
// ビジネスロジック（テナント境界・パスワードハッシュ化など）はサービス層テストでカバー済み。

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
// GET /api/v1/accounts（アカウント一覧）
// ═══════════════════════════════════════════════════════════

// TestListHandler_AssociationAdmin_Success
// association_admin が自治会内のアカウント一覧を取得できる
func TestListHandler_AssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧ハンドラー自治会", "HDL_LST_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-lst-adma.test", "pass123", "association_admin")
	_ = insertTestUser(t, &assocID, "メンバー1", "m1@hdl-lst-adma.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/accounts", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	accounts := data["accounts"].([]interface{})
	assert.GreaterOrEqual(t, len(accounts), 2, "自治会内のアカウントが返るべき")
}

// TestListHandler_SystemAdmin_AllAssociations
// system_admin は全自治会のアカウントを取得できる
func TestListHandler_SystemAdmin_AllAssociations(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "一覧SA自治会1", "HDL_LST_SA1")
	assoc2ID := insertTestAssociation(t, "一覧SA自治会2", "HDL_LST_SA2")
	_ = insertTestUser(t, &assoc1ID, "ユーザー1", "u1@hdl-lst-sa1.test", "pass123", "user")
	_ = insertTestUser(t, &assoc2ID, "ユーザー2", "u2@hdl-lst-sa2.test", "pass123", "user")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdl-lst-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/accounts", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	accounts := data["accounts"].([]interface{})
	assert.GreaterOrEqual(t, len(accounts), 2, "複数自治会のアカウントが含まれるべき")
}

// TestListHandler_SystemAdmin_FilterByAssociation
// system_admin は association_id クエリで絞り込みできる
func TestListHandler_SystemAdmin_FilterByAssociation(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "絞り込み自治会1", "HDL_LST_FLTR1")
	assoc2ID := insertTestAssociation(t, "絞り込み自治会2", "HDL_LST_FLTR2")
	_ = insertTestUser(t, &assoc1ID, "対象ユーザー", "target@hdl-lst-fltr1.test", "pass123", "user")
	_ = insertTestUser(t, &assoc2ID, "別自治会ユーザー", "other@hdl-lst-fltr2.test", "pass123", "user")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdl-lst-fltr.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	path := fmt.Sprintf("/api/v1/accounts?association_id=%s", assoc1ID.String())
	rr := doRequest(t, deps.router, http.MethodGet, path, nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	accounts := data["accounts"].([]interface{})
	for _, item := range accounts {
		acc := item.(map[string]interface{})
		assert.Equal(t, assoc1ID.String(), acc["association_id"], "絞り込みした自治会のアカウントのみ返るべき")
	}
}

// TestListHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestListHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/accounts", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestListHandler_Forbidden_UserRole
// 一般ユーザーロールは 403 が返る
func TestListHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403テスト自治会", "HDL_LST_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-lst-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/accounts", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/accounts（アカウント作成）
// ═══════════════════════════════════════════════════════════

// TestCreateHandler_AssociationAdmin_Success
// association_admin がアカウントを作成できる (201)
func TestCreateHandler_AssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "作成ハンドラー自治会", "HDL_CRT_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-crt-adma.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "新規メンバー",
		"email":    "newmember@hdl-crt-adma.test",
		"password": "password123",
		"role":     "user",
	}, token)

	require.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "新規メンバー", data["name"])
	assert.Equal(t, assocID.String(), data["association_id"])
	assert.Equal(t, "user", data["role"])
	assert.Equal(t, true, data["is_active"])

	t.Cleanup(func() {
		id, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
}

// TestCreateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestCreateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "テスト",
		"email":    "test@example.com",
		"password": "password123",
		"role":     "user",
	}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestCreateHandler_Forbidden_UserRole
// 一般ユーザーロールは 403 が返る
func TestCreateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403作成自治会", "HDL_CRT_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-crt-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "テスト",
		"email":    "test@hdl-crt-usr.test",
		"password": "password123",
		"role":     "user",
	}, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestCreateHandler_EmailConflict
// メールアドレス重複は 409 が返る
func TestCreateHandler_EmailConflict(t *testing.T) {
	assocID := insertTestAssociation(t, "重複作成自治会", "HDL_CRT_DUP")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-crt-dup.test", "pass123", "association_admin")
	_ = insertTestUser(t, &assocID, "既存ユーザー", "exist@hdl-crt-dup.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "重複ユーザー",
		"email":    "exist@hdl-crt-dup.test",
		"password": "password123",
		"role":     "user",
	}, token)

	assert.Equal(t, http.StatusConflict, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "EMAIL_CONFLICT", errData["code"])
}

// TestCreateHandler_WeakPassword
// 8文字未満のパスワードは 400 が返る
func TestCreateHandler_WeakPassword(t *testing.T) {
	assocID := insertTestAssociation(t, "弱パス作成自治会", "HDL_CRT_WPWD")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-crt-wpwd.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "テスト",
		"email":    "weak@hdl-crt-wpwd.test",
		"password": "short",
		"role":     "user",
	}, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "WEAK_PASSWORD", errData["code"])
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/accounts/:id（アカウント編集）
// ═══════════════════════════════════════════════════════════

// TestUpdateHandler_AssociationAdmin_Success
// association_admin が自治会内のアカウントを編集できる (200)
func TestUpdateHandler_AssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "編集ハンドラー自治会", "HDL_UPD_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-upd-adma.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assocID, "編集対象", "target@hdl-upd-adma.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		map[string]interface{}{
			"name":  "編集後の名前",
			"email": "target@hdl-upd-adma.test",
			"role":  "user",
		}, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "編集後の名前", data["name"])
}

// TestUpdateHandler_NotFound
// 存在しない ID は 404 が返る
func TestUpdateHandler_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "404編集自治会", "HDL_UPD_NF")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-upd-nf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s", uuid.New()),
		map[string]interface{}{
			"name":  "名前",
			"email": "nf@hdl-upd-nf.test",
			"role":  "user",
		}, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestUpdateHandler_CrossTenant_NotFound
// 他自治会のアカウント編集は 404 が返る（テナント境界）
func TestUpdateHandler_CrossTenant_NotFound(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "編集元自治会", "HDL_UPD_CRS1")
	assoc2ID := insertTestAssociation(t, "編集先自治会", "HDL_UPD_CRS2")
	adminID := insertTestUser(t, &assoc1ID, "A管理者", "admin@hdl-upd-crs1.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assoc2ID, "B一般ユーザー", "target@hdl-upd-crs2.test", "pass123", "user")
	token := makeTestToken(t, assoc1ID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		map[string]interface{}{
			"name":  "変更",
			"email": "target@hdl-upd-crs2.test",
			"role":  "user",
		}, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestUpdateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestUpdateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s", uuid.New()),
		map[string]interface{}{
			"name":  "名前",
			"email": "test@example.com",
			"role":  "user",
		}, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DELETE /api/v1/accounts/:id（アカウント無効化）
// ═══════════════════════════════════════════════════════════

// TestDeleteHandler_AssociationAdmin_Success
// association_admin が自治会内のアカウントを無効化できる (200)
func TestDeleteHandler_AssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "削除ハンドラー自治会", "HDL_DEL_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-del-adma.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assocID, "削除対象", "target@hdl-del-adma.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestDeleteHandler_SelfDeletion_Blocked
// 自分自身の削除は 400 CANNOT_DELETE_SELF が返る
func TestDeleteHandler_SelfDeletion_Blocked(t *testing.T) {
	assocID := insertTestAssociation(t, "自己削除ハンドラー自治会", "HDL_DEL_SELF")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-del-self.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/accounts/%s", adminID),
		nil, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "CANNOT_DELETE_SELF", errData["code"])
}

// TestDeleteHandler_CrossTenant_NotFound
// 他自治会のアカウント削除は 404 が返る
func TestDeleteHandler_CrossTenant_NotFound(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "削除元自治会", "HDL_DEL_CRS1")
	assoc2ID := insertTestAssociation(t, "削除先自治会", "HDL_DEL_CRS2")
	adminID := insertTestUser(t, &assoc1ID, "A管理者", "admin@hdl-del-crs1.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assoc2ID, "B一般ユーザー", "target@hdl-del-crs2.test", "pass123", "user")
	token := makeTestToken(t, assoc1ID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestDeleteHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestDeleteHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/accounts/%s", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/accounts/:id/activate（アカウント有効化）
// ═══════════════════════════════════════════════════════════

// TestActivateHandler_Success
// 無効化されたアカウントを有効化できる (200)
func TestActivateHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "有効化ハンドラー自治会", "HDL_ACT_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-act-adma.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assocID, "有効化対象", "target@hdl-act-adma.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	// まず無効化
	_, err := testPool.Exec(context.Background(),
		`UPDATE users SET is_active = false WHERE id = $1`, targetID)
	require.NoError(t, err)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/activate", targetID),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestActivateHandler_CrossTenant_NotFound
// 他自治会のアカウント有効化は 404 が返る
func TestActivateHandler_CrossTenant_NotFound(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "有効化元自治会", "HDL_ACT_CRS1")
	assoc2ID := insertTestAssociation(t, "有効化先自治会", "HDL_ACT_CRS2")
	adminID := insertTestUser(t, &assoc1ID, "A管理者", "admin@hdl-act-crs1.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assoc2ID, "B一般ユーザー", "target@hdl-act-crs2.test", "pass123", "user")
	token := makeTestToken(t, assoc1ID.String(), adminID.String(), "association_admin")

	// まず無効化
	_, _ = testPool.Exec(context.Background(),
		`UPDATE users SET is_active = false WHERE id = $1`, targetID)

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/activate", targetID),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestActivateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestActivateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/activate", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/accounts/:id/deactivate（アカウント無効化）
// ═══════════════════════════════════════════════════════════

// TestDeactivateHandler_Success
// association_admin がアカウントを無効化できる (200)
func TestDeactivateHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "無効化ハンドラー自治会", "HDL_DACT_ADMA")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-dact-adma.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assocID, "無効化対象", "target@hdl-dact-adma.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/deactivate", targetID),
		nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestDeactivateHandler_SelfDeactivation_Blocked
// 自分自身の無効化は 400 CANNOT_DELETE_SELF が返る
func TestDeactivateHandler_SelfDeactivation_Blocked(t *testing.T) {
	assocID := insertTestAssociation(t, "自己無効化ハンドラー自治会", "HDL_DACT_SELF")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-dact-self.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/deactivate", adminID),
		nil, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errData := body["error"].(map[string]interface{})
	assert.Equal(t, "CANNOT_DELETE_SELF", errData["code"])
}

// TestDeactivateHandler_CrossTenant_NotFound
// 他自治会のアカウント無効化は 404 が返る
func TestDeactivateHandler_CrossTenant_NotFound(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "無効化元自治会", "HDL_DACT_CRS1")
	assoc2ID := insertTestAssociation(t, "無効化先自治会", "HDL_DACT_CRS2")
	adminID := insertTestUser(t, &assoc1ID, "A管理者", "admin@hdl-dact-crs1.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assoc2ID, "B一般ユーザー", "target@hdl-dact-crs2.test", "pass123", "user")
	token := makeTestToken(t, assoc1ID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/deactivate", targetID),
		nil, token)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestDeactivateHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestDeactivateHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/deactivate", uuid.New()),
		nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// 追加テスト：スペック網羅
// ═══════════════════════════════════════════════════════════

// TestListHandler_AssociationAdmin_TenantIsolation
// association_admin は自治会内のアカウントのみ取得できる（他自治会が混入しない）
func TestListHandler_AssociationAdmin_TenantIsolation(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "テナント隔離自治会A", "HDL_ISOA")
	assoc2ID := insertTestAssociation(t, "テナント隔離自治会B", "HDL_ISOB")
	adminID := insertTestUser(t, &assoc1ID, "A管理者", "admin@hdl-isoa.test", "pass123", "association_admin")
	_ = insertTestUser(t, &assoc2ID, "B一般ユーザー", "b@hdl-isob.test", "pass123", "user")
	token := makeTestToken(t, assoc1ID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/accounts", nil, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	accounts := data["accounts"].([]interface{})
	for _, item := range accounts {
		acc := item.(map[string]interface{})
		assert.Equal(t, assoc1ID.String(), acc["association_id"],
			"他自治会のアカウントが混入するべきでない")
	}
}

// TestCreateHandler_SystemAdmin_AnyAssociation
// system_admin は association_id を指定して任意の自治会にアカウントを作成できる
func TestCreateHandler_SystemAdmin_AnyAssociation(t *testing.T) {
	targetAssocID := insertTestAssociation(t, "SA作成先自治会", "HDL_CRT_SA_ASSOC")
	sysAdminID := insertTestUser(t, nil, "システム管理者", "sysadmin@hdl-crt-sa.test", "pass123", "system_admin")
	token := makeTestToken(t, "", sysAdminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":           "SA作成ユーザー",
		"email":          "sauser@hdl-crt-sa-assoc.test",
		"password":       "password123",
		"role":           "user",
		"association_id": targetAssocID.String(),
	}, token)

	require.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, targetAssocID.String(), data["association_id"],
		"指定した自治会に作成されるべき")

	t.Cleanup(func() {
		id, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
}

// TestCreateHandler_MissingRequiredFields
// 必須項目（名前）が空のとき 400 が返る
func TestCreateHandler_MissingRequiredFields(t *testing.T) {
	assocID := insertTestAssociation(t, "必須項目自治会", "HDL_CRT_REQ")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-crt-req.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/accounts", map[string]interface{}{
		"name":     "   ", // 空白のみ
		"email":    "req@hdl-crt-req.test",
		"password": "password123",
		"role":     "user",
	}, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestUpdateHandler_Forbidden_UserRole
// 一般ユーザーロールはアカウント編集で 403 が返る
func TestUpdateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403更新自治会", "HDL_UPD_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-upd-usr.test", "pass123", "user")
	targetID := insertTestUser(t, &assocID, "更新対象", "target@hdl-upd-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		map[string]interface{}{
			"name":  "変更後",
			"email": "target@hdl-upd-usr.test",
			"role":  "user",
		}, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestDeleteHandler_Forbidden_UserRole
// 一般ユーザーロールはアカウント削除で 403 が返る
func TestDeleteHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403削除自治会", "HDL_DEL_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-del-usr.test", "pass123", "user")
	targetID := insertTestUser(t, &assocID, "削除対象", "target@hdl-del-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodDelete,
		fmt.Sprintf("/api/v1/accounts/%s", targetID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestActivateHandler_Forbidden_UserRole
// 一般ユーザーロールはアカウント有効化で 403 が返る
func TestActivateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403有効化自治会", "HDL_ACT_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-act-usr.test", "pass123", "user")
	targetID := insertTestUser(t, &assocID, "対象", "target@hdl-act-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/activate", targetID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestDeactivateHandler_Forbidden_UserRole
// 一般ユーザーロールはアカウント無効化で 403 が返る
func TestDeactivateHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403無効化自治会", "HDL_DACT_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-dact-usr.test", "pass123", "user")
	targetID := insertTestUser(t, &assocID, "対象", "target@hdl-dact-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut,
		fmt.Sprintf("/api/v1/accounts/%s/deactivate", targetID),
		nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
