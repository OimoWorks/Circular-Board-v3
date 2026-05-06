package permission_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════
// GET /api/v1/permissions（権限一覧取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionGetMatrixHandler_Success_SystemAdmin
// system_adminが取得できる（200）
func TestPermissionGetMatrixHandler_Success_SystemAdmin(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_HDL1", "sa-hdl1@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "roles")
	assert.Contains(t, data, "features")
	assert.Contains(t, data, "permissions")
}

// TestPermissionGetMatrixHandler_ReturnsMatrix
// ロール×機能のマトリクスが返る
func TestPermissionGetMatrixHandler_ReturnsMatrix(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_HDL2", "sa-hdl2@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions", nil, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})

	roles := data["roles"].([]interface{})
	features := data["features"].([]interface{})
	permissions := data["permissions"].([]interface{})

	assert.GreaterOrEqual(t, len(roles), 5, "5ロール以上が返るべき")
	assert.GreaterOrEqual(t, len(features), 6, "6機能以上が返るべき")
	assert.GreaterOrEqual(t, len(permissions), 30, "30件以上の権限が返るべき")
}

// TestPermissionGetMatrixHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionGetMatrixHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestPermissionGetMatrixHandler_Forbidden_AssociationAdmin
// association_adminは不可（403）
func TestPermissionGetMatrixHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "権限403自治会A", "PERM_HDL_FA")
	userID := insertTestUser(t, &assocID, "会長", "admin@perm-hdl-fa.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions", nil, token)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestPermissionGetMatrixHandler_Forbidden_User
// 一般ユーザーは不可（403）
func TestPermissionGetMatrixHandler_Forbidden_User(t *testing.T) {
	assocID := insertTestAssociation(t, "権限403自治会B", "PERM_HDL_FU")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@perm-hdl-fu.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions", nil, token)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// PUT /api/v1/permissions（権限更新）
// ═══════════════════════════════════════════════════════════

// TestPermissionUpdateHandler_Success_SystemAdmin
// system_adminが更新できる（200）
func TestPermissionUpdateHandler_Success_SystemAdmin(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_UPD1", "sa-upd1@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	roleID := getRoleIDByName(t, "user_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, false, false, false, "own_association")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut, "/api/v1/permissions",
		map[string]interface{}{
			"permissions": []map[string]interface{}{
				{
					"role_id":    roleID.String(),
					"feature_id": featureID.String(),
					"can_view":   true,
					"can_create": true,
					"can_edit":   false,
					"can_delete": false,
					"scope":      "own_association",
				},
			},
		},
		token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "権限を更新しました", data["message"])
}

// TestPermissionUpdateHandler_BulkUpdate
// 一括更新が正しく反映される
func TestPermissionUpdateHandler_BulkUpdate(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_UPD2", "sa-upd2@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	roleID1 := getRoleIDByName(t, "vice_admin")
	featureID1 := getFeatureIDByName(t, "accounts")
	roleID2 := getRoleIDByName(t, "user_admin")
	featureID2 := getFeatureIDByName(t, "files")
	withPermission(t, roleID1, featureID1, false, false, false, false, "own_association")
	withPermission(t, roleID2, featureID2, true, false, false, false, "own_association")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut, "/api/v1/permissions",
		map[string]interface{}{
			"permissions": []map[string]interface{}{
				{
					"role_id": roleID1.String(), "feature_id": featureID1.String(),
					"can_view": true, "can_create": false, "can_edit": false, "can_delete": false,
					"scope": "own_association",
				},
				{
					"role_id": roleID2.String(), "feature_id": featureID2.String(),
					"can_view": true, "can_create": true, "can_edit": false, "can_delete": false,
					"scope": "own_association",
				},
			},
		},
		token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)

	// 反映確認
	permRepo := deps.permSvc
	pc1, err := permRepo.CheckPermission(context.Background(), "vice_admin", "accounts")
	require.NoError(t, err)
	assert.True(t, pc1.CanView, "vice_admin/accountsのcan_viewがtrueになるべき")

	pc2, err := permRepo.CheckPermission(context.Background(), "user_admin", "files")
	require.NoError(t, err)
	assert.True(t, pc2.CanCreate, "user_admin/filesのcan_createがtrueになるべき")
}

// TestPermissionUpdateHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionUpdateHandler_Unauthorized(t *testing.T) {
	roleID := getRoleIDByName(t, "user")
	featureID := getFeatureIDByName(t, "notices")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut, "/api/v1/permissions",
		map[string]interface{}{
			"permissions": []map[string]interface{}{
				{
					"role_id": roleID.String(), "feature_id": featureID.String(),
					"can_view": true, "scope": "own_association",
				},
			},
		},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestPermissionUpdateHandler_Forbidden_AssociationAdmin
// association_adminは不可（403）
func TestPermissionUpdateHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "更新403自治会", "PERM_HDL_UPDF")
	userID := insertTestUser(t, &assocID, "会長", "admin@perm-hdl-updf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	roleID := getRoleIDByName(t, "user")
	featureID := getFeatureIDByName(t, "notices")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut, "/api/v1/permissions",
		map[string]interface{}{
			"permissions": []map[string]interface{}{
				{
					"role_id": roleID.String(), "feature_id": featureID.String(),
					"can_view": true, "scope": "own_association",
				},
			},
		},
		token,
	)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestPermissionUpdateHandler_InvalidRequest
// 必須項目なしはエラー（400）
func TestPermissionUpdateHandler_InvalidRequest(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_UPD3", "sa-upd3@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPut, "/api/v1/permissions",
		map[string]interface{}{
			"permissions": []map[string]interface{}{
				{
					"role_id":    "not-a-uuid",
					"feature_id": uuid.New().String(),
					"can_view":   true,
					"scope":      "own_association",
				},
			},
		},
		token,
	)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/permissions/roles（ロール一覧）
// ═══════════════════════════════════════════════════════════

// TestPermissionGetRolesHandler_Success_SystemAdmin
// system_adminが取得できる（200）
func TestPermissionGetRolesHandler_Success_SystemAdmin(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_ROLES1", "sa-roles1@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/roles", nil, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	roles := data["roles"].([]interface{})
	assert.GreaterOrEqual(t, len(roles), 5, "5ロール以上が返るべき")
}

// TestPermissionGetRolesHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionGetRolesHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/roles", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestPermissionGetRolesHandler_Forbidden_AssociationAdmin
// association_adminは不可（403）
func TestPermissionGetRolesHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "ロール403自治会", "PERM_HDL_RF")
	userID := insertTestUser(t, &assocID, "会長", "admin@perm-hdl-rf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/roles", nil, token)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/permissions/features（機能一覧）
// ═══════════════════════════════════════════════════════════

// TestPermissionGetFeaturesHandler_Success_SystemAdmin
// system_adminが取得できる（200）
func TestPermissionGetFeaturesHandler_Success_SystemAdmin(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_FEAT1", "sa-feat1@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/features", nil, token)

	require.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	features := data["features"].([]interface{})
	assert.GreaterOrEqual(t, len(features), 6, "6機能以上が返るべき")
}

// TestPermissionGetFeaturesHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionGetFeaturesHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/features", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestPermissionGetFeaturesHandler_Forbidden_AssociationAdmin
// association_adminは不可（403）
func TestPermissionGetFeaturesHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "機能403自治会", "PERM_HDL_FF")
	userID := insertTestUser(t, &assocID, "会長", "admin@perm-hdl-ff.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/features", nil, token)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/permissions/emergency-appointment（緊急任命）
// ═══════════════════════════════════════════════════════════

// TestPermissionEmergencyAppointmentHandler_Success
// system_adminが緊急任命できる（200）
func TestPermissionEmergencyAppointmentHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "緊急任命ハンドラー自治会", "PERM_HDL_EA1")
	adminID := insertTestUser(t, nil, "SA_EA_HDL1", "sa-ea-hdl1@perm-hdl.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "任命対象", "target@perm-hdl-ea1.test", "pass123", "user")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/permissions/emergency-appointment",
		map[string]interface{}{
			"user_id":  targetID.String(),
			"new_role": "vice_admin",
		},
		token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "緊急任命を実行しました", data["message"])

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}

// TestPermissionEmergencyAppointmentHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionEmergencyAppointmentHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/permissions/emergency-appointment",
		map[string]interface{}{"user_id": uuid.New().String(), "new_role": "vice_admin"},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestPermissionEmergencyAppointmentHandler_Forbidden
// association_adminは不可（403）
func TestPermissionEmergencyAppointmentHandler_Forbidden(t *testing.T) {
	assocID := insertTestAssociation(t, "EA403自治会", "PERM_HDL_EAF")
	adminID := insertTestUser(t, &assocID, "会長", "admin@perm-hdl-eaf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/permissions/emergency-appointment",
		map[string]interface{}{"user_id": uuid.New().String(), "new_role": "vice_admin"},
		token,
	)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestPermissionEmergencyAppointmentHandler_InvalidRole
// 不正なロール名はエラー（400）
func TestPermissionEmergencyAppointmentHandler_InvalidRole(t *testing.T) {
	assocID := insertTestAssociation(t, "EA400自治会", "PERM_HDL_EA400")
	adminID := insertTestUser(t, nil, "SA_EA400", "sa-ea400@perm-hdl.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "対象", "target@perm-hdl-ea400.test", "pass123", "user")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/permissions/emergency-appointment",
		map[string]interface{}{
			"user_id":  targetID.String(),
			"new_role": "invalid_role_name",
		},
		token,
	)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/permissions/logs（操作ログ取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionGetLogsHandler_Success_SystemAdmin
// system_adminが取得できる（200）
func TestPermissionGetLogsHandler_Success_SystemAdmin(t *testing.T) {
	adminID := insertTestUser(t, nil, "SA_LOG1", "sa-log1@perm-hdl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", adminID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/logs", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "logs")
}

// TestPermissionGetLogsHandler_Unauthorized
// 未認証は不可（401）
func TestPermissionGetLogsHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/permissions/logs", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
