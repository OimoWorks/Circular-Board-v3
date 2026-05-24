package permission_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/permission"
)

func newPermissionService() *permission.Service {
	return permission.NewService(permission.NewRepository(testPool))
}

// ═══════════════════════════════════════════════════════════
// Service.GetMatrix（権限マトリクス取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionService_GetMatrix_ReturnsAll
// ロール・機能・権限が全て取得できる
func TestPermissionService_GetMatrix_ReturnsAll(t *testing.T) {
	svc := newPermissionService()
	matrix, err := svc.GetMatrix(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, matrix)

	assert.GreaterOrEqual(t, len(matrix.Roles), 5, "5ロール以上が含まれるべき")
	assert.GreaterOrEqual(t, len(matrix.Features), 6, "6機能以上が含まれるべき")
	assert.GreaterOrEqual(t, len(matrix.Permissions), 30, "30件以上の権限が含まれるべき")
}

// TestPermissionService_GetMatrix_NeverNil
// データがない場合も空スライスが返る
func TestPermissionService_GetMatrix_NeverNil(t *testing.T) {
	svc := newPermissionService()
	matrix, err := svc.GetMatrix(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, matrix.Roles)
	assert.NotNil(t, matrix.Features)
	assert.NotNil(t, matrix.Permissions)
}

// ═══════════════════════════════════════════════════════════
// Service.CheckPermission（権限チェック）
// ═══════════════════════════════════════════════════════════

// TestPermissionService_CheckPermission_CanView
// 閲覧権限があるユーザーはアクセス可
func TestPermissionService_CheckPermission_CanView(t *testing.T) {
	roleID := getRoleIDByName(t, "user")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, false, false, false, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "user", "notices", nil)
	require.NoError(t, err)
	assert.True(t, pc.CanView, "userはnoticesを閲覧できるべき")
}

// TestPermissionService_CheckPermission_CanCreate
// 作成権限があるユーザーは作成可
func TestPermissionService_CheckPermission_CanCreate(t *testing.T) {
	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "association_admin", "notices", nil)
	require.NoError(t, err)
	assert.True(t, pc.CanCreate, "association_adminはnoticesを作成できるべき")
}

// TestPermissionService_CheckPermission_CanEdit
// 編集権限があるユーザーは編集可
func TestPermissionService_CheckPermission_CanEdit(t *testing.T) {
	roleID := getRoleIDByName(t, "vice_admin")
	featureID := getFeatureIDByName(t, "files")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "vice_admin", "files", nil)
	require.NoError(t, err)
	assert.True(t, pc.CanEdit, "vice_adminはfilesを編集できるべき")
}

// TestPermissionService_CheckPermission_CanDelete
// 削除権限があるユーザーは削除可
func TestPermissionService_CheckPermission_CanDelete(t *testing.T) {
	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "association_admin", "surveys", nil)
	require.NoError(t, err)
	assert.True(t, pc.CanDelete, "association_adminはsurveysを削除できるべき")
}

// TestPermissionService_CheckPermission_NoAccess
// 権限がない機能はアクセス不可
func TestPermissionService_CheckPermission_NoAccess(t *testing.T) {
	roleID := getRoleIDByName(t, "user")
	featureID := getFeatureIDByName(t, "accounts")
	withPermission(t, roleID, featureID, false, false, false, false, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "user", "accounts", nil)
	require.NoError(t, err)
	assert.False(t, pc.CanView, "userはaccountsを閲覧できてはならない")
	assert.False(t, pc.CanCreate)
	assert.False(t, pc.CanEdit)
	assert.False(t, pc.CanDelete)
}

// TestPermissionService_CheckPermission_ScopeAll
// スコープallのユーザーはスコープ値がallで返る
func TestPermissionService_CheckPermission_ScopeAll(t *testing.T) {
	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "system_admin", "notices", nil)
	require.NoError(t, err)
	assert.Equal(t, "all", pc.Scope, "system_adminのスコープはallであるべき")
}

// TestPermissionService_CheckPermission_ScopeOwnAssociation
// スコープown_associationのロールは正しい値が返る
func TestPermissionService_CheckPermission_ScopeOwnAssociation(t *testing.T) {
	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	svc := newPermissionService()
	pc, err := svc.CheckPermission(context.Background(), "association_admin", "notices", nil)
	require.NoError(t, err)
	assert.Equal(t, "own_association", pc.Scope, "association_adminのスコープはown_associationであるべき")
}

// TestPermissionService_CheckPermission_NotFound
// 存在しないロールはエラー
func TestPermissionService_CheckPermission_NotFound(t *testing.T) {
	svc := newPermissionService()
	_, err := svc.CheckPermission(context.Background(), "nonexistent_role", "notices", nil)
	assert.Error(t, err, "存在しないロールはエラーが返るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.UpdatePermissions（権限更新）
// ═══════════════════════════════════════════════════════════

// TestPermissionService_UpdatePermissions_Success
// 権限を変更できる
func TestPermissionService_UpdatePermissions_Success(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA_SVC1", "sa-svc1@perm-svc.test", "pass123", "system_admin")

	roleID := getRoleIDByName(t, "user_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, false, false, false, "own_association")

	svc := newPermissionService()
	err := svc.UpdatePermissions(context.Background(), operatorID, nil, []permission.UpdateInput{
		{
			RoleID: roleID, FeatureID: featureID,
			CanView: true, CanCreate: true, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
	})
	require.NoError(t, err)

	// 変更後即時反映される
	pc, err := svc.CheckPermission(context.Background(), "user_admin", "surveys", nil)
	require.NoError(t, err)
	assert.True(t, pc.CanCreate, "更新後即時にcan_createがtrueになるべき")
}

// TestPermissionService_UpdatePermissions_SystemAdminNotChangeable
// system_adminの権限は変更不可（スキップされる）
func TestPermissionService_UpdatePermissions_SystemAdminNotChangeable(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA_SVC2", "sa-svc2@perm-svc.test", "pass123", "system_admin")

	sysAdminRoleID := getRoleIDByName(t, "system_admin")
	featureID := getFeatureIDByName(t, "permissions")

	svc := newPermissionService()
	before, err := svc.CheckPermission(context.Background(), "system_admin", "permissions", nil)
	require.NoError(t, err)
	require.True(t, before.CanView)

	err = svc.UpdatePermissions(context.Background(), operatorID, nil, []permission.UpdateInput{
		{
			RoleID: sysAdminRoleID, FeatureID: featureID,
			CanView: false, CanCreate: false, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
	})
	require.NoError(t, err)

	after, err := svc.CheckPermission(context.Background(), "system_admin", "permissions", nil)
	require.NoError(t, err)
	assert.Equal(t, before.CanView, after.CanView, "system_adminの権限は変更されていないべき")
}

// TestPermissionService_UpdatePermissions_ImmediateReflection
// 変更後即時反映される（CheckPermissionで確認）
func TestPermissionService_UpdatePermissions_ImmediateReflection(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA_SVC3", "sa-svc3@perm-svc.test", "pass123", "system_admin")

	roleID := getRoleIDByName(t, "vice_admin")
	featureID := getFeatureIDByName(t, "accounts")
	withPermission(t, roleID, featureID, false, false, false, false, "own_association")

	svc := newPermissionService()

	before, err := svc.CheckPermission(context.Background(), "vice_admin", "accounts", nil)
	require.NoError(t, err)
	assert.False(t, before.CanView)

	err = svc.UpdatePermissions(context.Background(), operatorID, nil, []permission.UpdateInput{
		{
			RoleID: roleID, FeatureID: featureID,
			CanView: true, CanCreate: false, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
	})
	require.NoError(t, err)

	after, err := svc.CheckPermission(context.Background(), "vice_admin", "accounts", nil)
	require.NoError(t, err)
	assert.True(t, after.CanView, "権限変更が即時に反映されているべき")
}

// ═══════════════════════════════════════════════════════════
// Service.EmergencyAppointment（緊急任命）
// ═══════════════════════════════════════════════════════════

// TestPermissionService_EmergencyAppointment_Success
// system_adminが緊急任命できる
func TestPermissionService_EmergencyAppointment_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "緊急任命サービス自治会", "PERM_SVC_EA1")
	operatorID := insertTestUser(t, nil, "SA_EA1", "sa-ea1@perm-svc.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "EA対象1", "target-ea1@perm-svc.test", "pass123", "user")

	svc := newPermissionService()
	err := svc.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "vice_admin",
	})
	require.NoError(t, err)

	var newRole string
	err = testPool.QueryRow(context.Background(),
		`SELECT role FROM users WHERE id = $1`, targetID).Scan(&newRole)
	require.NoError(t, err)
	assert.Equal(t, "vice_admin", newRole, "ロールがvice_adminに変更されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}

// TestPermissionService_EmergencyAppointment_OperationLogRecorded
// 操作ログに記録される
func TestPermissionService_EmergencyAppointment_OperationLogRecorded(t *testing.T) {
	assocID := insertTestAssociation(t, "ログ記録サービス自治会", "PERM_SVC_EA2")
	operatorID := insertTestUser(t, nil, "SA_EA2", "sa-ea2@perm-svc.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "EA対象2", "target-ea2@perm-svc.test", "pass123", "user")

	svc := newPermissionService()
	err := svc.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "user_admin",
	})
	require.NoError(t, err)

	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM operation_logs
		 WHERE operator_id = $1 AND operation_type = 'emergency_appointment'`,
		operatorID,
	).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "操作ログが作成されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}

// TestPermissionService_EmergencyAppointment_NoticeCreated
// 対象自治会の全アカウントに通知される
func TestPermissionService_EmergencyAppointment_NoticeCreated(t *testing.T) {
	assocID := insertTestAssociation(t, "通知サービス自治会", "PERM_SVC_EA3")
	operatorID := insertTestUser(t, nil, "SA_EA3", "sa-ea3@perm-svc.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "EA対象3", "target-ea3@perm-svc.test", "pass123", "user")

	svc := newPermissionService()
	err := svc.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "association_admin",
	})
	require.NoError(t, err)

	var noticeCount int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notices WHERE association_id = $1`, assocID,
	).Scan(&noticeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, noticeCount, "自治会にお知らせが作成されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}

// TestPermissionService_EmergencyAppointment_NotFoundUser
// 存在しないユーザーへの任命はエラー
func TestPermissionService_EmergencyAppointment_NotFoundUser(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA_EA4", "sa-ea4@perm-svc.test", "pass123", "system_admin")

	svc := newPermissionService()
	err := svc.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  uuid.New(), // 存在しないユーザーID
		NewRole: "vice_admin",
	})
	assert.Error(t, err, "存在しないユーザーへの任命はエラーが返るべき")
}
