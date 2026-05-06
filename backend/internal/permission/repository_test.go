package permission_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/permission"
)

// ═══════════════════════════════════════════════════════════
// Repository.GetRoles（ロール一覧取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_GetRoles_ReturnsAll
// 全ロールが取得できる
func TestPermissionRepository_GetRoles_ReturnsAll(t *testing.T) {
	repo := permission.NewRepository(testPool)
	roles, err := repo.GetRoles(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, roles, "ロール一覧が空であってはならない")

	names := make(map[string]bool)
	for _, r := range roles {
		names[r.Name] = true
	}
	assert.True(t, names["system_admin"], "system_adminが含まれるべき")
	assert.True(t, names["association_admin"], "association_adminが含まれるべき")
	assert.True(t, names["vice_admin"], "vice_adminが含まれるべき")
	assert.True(t, names["user_admin"], "user_adminが含まれるべき")
	assert.True(t, names["user"], "userが含まれるべき")
}

// TestPermissionRepository_GetRoles_IsSystemFlag
// is_systemフラグが正しく返る
func TestPermissionRepository_GetRoles_IsSystemFlag(t *testing.T) {
	repo := permission.NewRepository(testPool)
	roles, err := repo.GetRoles(context.Background())
	require.NoError(t, err)

	for _, r := range roles {
		switch r.Name {
		case "system_admin":
			assert.True(t, r.IsSystem, "system_adminのis_systemはtrueであるべき")
		case "association_admin", "vice_admin", "user_admin", "user":
			assert.False(t, r.IsSystem, r.Name+"のis_systemはfalseであるべき")
		}
	}
}

// TestPermissionRepository_GetRoles_DisplayName
// display_nameが正しく返る
func TestPermissionRepository_GetRoles_DisplayName(t *testing.T) {
	repo := permission.NewRepository(testPool)
	roles, err := repo.GetRoles(context.Background())
	require.NoError(t, err)

	displayNames := make(map[string]string)
	for _, r := range roles {
		displayNames[r.Name] = r.DisplayName
	}
	assert.Equal(t, "システム管理者", displayNames["system_admin"])
	assert.Equal(t, "自治会長", displayNames["association_admin"])
	assert.Equal(t, "副会長", displayNames["vice_admin"])
	assert.Equal(t, "ユーザー管理者", displayNames["user_admin"])
	assert.Equal(t, "一般ユーザー", displayNames["user"])
}

// ═══════════════════════════════════════════════════════════
// Repository.GetFeatures（機能一覧取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_GetFeatures_SortOrder
// 全機能がsort_order順で取得できる
func TestPermissionRepository_GetFeatures_SortOrder(t *testing.T) {
	repo := permission.NewRepository(testPool)
	features, err := repo.GetFeatures(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, features)

	for i := 1; i < len(features); i++ {
		assert.LessOrEqual(t, features[i-1].SortOrder, features[i].SortOrder,
			"機能はsort_order順で並んでいるべき")
	}
}

// TestPermissionRepository_GetFeatures_DisplayName
// display_nameが正しく返る
func TestPermissionRepository_GetFeatures_DisplayName(t *testing.T) {
	repo := permission.NewRepository(testPool)
	features, err := repo.GetFeatures(context.Background())
	require.NoError(t, err)

	displayNames := make(map[string]string)
	for _, f := range features {
		displayNames[f.Name] = f.DisplayName
	}
	assert.Equal(t, "お知らせ", displayNames["notices"])
	assert.Equal(t, "回覧物", displayNames["files"])
	assert.Equal(t, "アンケート", displayNames["surveys"])
	assert.Equal(t, "アカウント管理", displayNames["accounts"])
	assert.Equal(t, "自治会管理", displayNames["associations"])
	assert.Equal(t, "権限管理", displayNames["permissions"])
}

// ═══════════════════════════════════════════════════════════
// Repository.GetAllPermissions（権限一覧取得）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_GetAllPermissions_ReturnsAll
// 全ロール×全機能の権限が取得できる
func TestPermissionRepository_GetAllPermissions_ReturnsAll(t *testing.T) {
	repo := permission.NewRepository(testPool)
	perms, err := repo.GetAllPermissions(context.Background())
	require.NoError(t, err)
	// 5ロール×6機能 = 30件
	assert.GreaterOrEqual(t, len(perms), 30, "少なくとも30件の権限レコードが必要")
}

// TestPermissionRepository_GetAllPermissions_ScopeCorrect
// スコープが正しく返る
func TestPermissionRepository_GetAllPermissions_ScopeCorrect(t *testing.T) {
	repo := permission.NewRepository(testPool)
	perms, err := repo.GetAllPermissions(context.Background())
	require.NoError(t, err)

	roleIDByName := make(map[uuid.UUID]string)
	roles, _ := repo.GetRoles(context.Background())
	for _, r := range roles {
		roleIDByName[r.ID] = r.Name
	}

	for _, p := range perms {
		assert.Contains(t, []string{"all", "own_association"}, p.Scope,
			"スコープはallまたはown_associationのみ")
		roleName := roleIDByName[p.RoleID]
		if roleName == "system_admin" {
			assert.Equal(t, "all", p.Scope, "system_adminのスコープはallであるべき")
		}
	}
}

// TestPermissionRepository_GetAllPermissions_DynamicRole
// ロールが増えた場合も動的に取得できる（新ロール追加テスト）
func TestPermissionRepository_GetAllPermissions_DynamicRole(t *testing.T) {
	// 新しいテスト用ロールを追加
	newRoleID := uuid.New()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO roles (id, name, display_name, is_system) VALUES ($1, $2, $3, false)`,
		newRoleID, "test_role_"+newRoleID.String()[:8], "テスト用ロール",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM roles WHERE id = $1`, newRoleID)
	})

	// 新ロールに権限を付与
	featureID := getFeatureIDByName(t, "notices")
	_, err = testPool.Exec(context.Background(),
		`INSERT INTO role_permissions (role_id, feature_id, can_view, can_create, can_edit, can_delete, scope)
		 VALUES ($1, $2, true, false, false, false, 'own_association')`,
		newRoleID, featureID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM role_permissions WHERE role_id = $1`, newRoleID)
	})

	repo := permission.NewRepository(testPool)
	perms, err := repo.GetAllPermissions(context.Background())
	require.NoError(t, err)

	found := false
	for _, p := range perms {
		if p.RoleID == newRoleID && p.FeatureID == featureID {
			found = true
			assert.True(t, p.CanView)
			assert.False(t, p.CanCreate)
		}
	}
	assert.True(t, found, "新ロールの権限が一覧に含まれるべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.GetPermissionByRoleAndFeature（権限チェック）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_GetPermissionByRoleAndFeature_Success
// ロール名・機能名で権限チェック用データを取得できる
func TestPermissionRepository_GetPermissionByRoleAndFeature_Success(t *testing.T) {
	repo := permission.NewRepository(testPool)
	pc, err := repo.GetPermissionByRoleAndFeature(context.Background(), "association_admin", "notices")
	require.NoError(t, err)
	assert.True(t, pc.CanView, "association_adminはnoticesを閲覧できるべき")
	assert.True(t, pc.CanCreate, "association_adminはnoticesを作成できるべき")
	assert.Equal(t, "own_association", pc.Scope)
}

// TestPermissionRepository_GetPermissionByRoleAndFeature_UserViewOnly
// 一般ユーザーは閲覧のみ
func TestPermissionRepository_GetPermissionByRoleAndFeature_UserViewOnly(t *testing.T) {
	repo := permission.NewRepository(testPool)
	pc, err := repo.GetPermissionByRoleAndFeature(context.Background(), "user", "notices")
	require.NoError(t, err)
	assert.True(t, pc.CanView, "userはnoticesを閲覧できるべき")
	assert.False(t, pc.CanCreate, "userはnoticesを作成できてはならない")
	assert.False(t, pc.CanEdit, "userはnoticesを編集できてはならない")
	assert.False(t, pc.CanDelete, "userはnoticesを削除できてはならない")
}

// TestPermissionRepository_GetPermissionByRoleAndFeature_NotFound
// 存在しないロールIDはエラー
func TestPermissionRepository_GetPermissionByRoleAndFeature_NotFound(t *testing.T) {
	repo := permission.NewRepository(testPool)
	_, err := repo.GetPermissionByRoleAndFeature(context.Background(), "nonexistent_role", "notices")
	assert.Error(t, err, "存在しないロールはエラーが返るべき")
}

// TestPermissionRepository_GetPermissionByRoleAndFeature_SystemAdmin
// system_adminはスコープ=all
func TestPermissionRepository_GetPermissionByRoleAndFeature_SystemAdmin(t *testing.T) {
	repo := permission.NewRepository(testPool)
	pc, err := repo.GetPermissionByRoleAndFeature(context.Background(), "system_admin", "permissions")
	require.NoError(t, err)
	assert.True(t, pc.CanView)
	assert.True(t, pc.CanCreate)
	assert.True(t, pc.CanEdit)
	assert.True(t, pc.CanDelete)
	assert.Equal(t, "all", pc.Scope)
}

// ═══════════════════════════════════════════════════════════
// Repository.UpdatePermissions（権限更新）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_UpdatePermissions_CanViewUpdate
// can_viewが正しく更新される
func TestPermissionRepository_UpdatePermissions_CanViewUpdate(t *testing.T) {
	assocID := insertTestAssociation(t, "権限更新自治会", "PERM_REPO_UPD1")
	operatorID := insertTestUser(t, nil, "管理者", "admin@perm-repo-upd1.test", "pass123", "system_admin")

	roleID := getRoleIDByName(t, "user_admin")
	featureID := getFeatureIDByName(t, "accounts")

	// 現在の値を保存して後で復元
	withPermission(t, roleID, featureID, true, true, false, false, "own_association")

	// can_editをtrueに更新
	repo := permission.NewRepository(testPool)
	err := repo.UpdatePermissions(context.Background(), operatorID, []permission.UpdateInput{
		{
			RoleID:    roleID,
			FeatureID: featureID,
			CanView:   true,
			CanCreate: true,
			CanEdit:   true, // falseからtrueへ
			CanDelete: false,
			Scope:     "own_association",
		},
	})
	require.NoError(t, err)

	pc, err := repo.GetPermissionByRoleAndFeature(context.Background(), "user_admin", "accounts")
	require.NoError(t, err)
	assert.True(t, pc.CanEdit, "can_editがtrueに更新されているべき")

	_ = assocID
}

// TestPermissionRepository_UpdatePermissions_ScopeUpdate
// スコープが正しく更新される
func TestPermissionRepository_UpdatePermissions_ScopeUpdate(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA", "sa@perm-repo-scope.test", "pass123", "system_admin")

	roleID := getRoleIDByName(t, "vice_admin")
	featureID := getFeatureIDByName(t, "notices")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	repo := permission.NewRepository(testPool)
	err := repo.UpdatePermissions(context.Background(), operatorID, []permission.UpdateInput{
		{
			RoleID:    roleID,
			FeatureID: featureID,
			CanView:   true,
			CanCreate: true,
			CanEdit:   true,
			CanDelete: true,
			Scope:     "all", // own_association → all
		},
	})
	require.NoError(t, err)

	pc, err := repo.GetPermissionByRoleAndFeature(context.Background(), "vice_admin", "notices")
	require.NoError(t, err)
	assert.Equal(t, "all", pc.Scope, "スコープがallに更新されているべき")
}

// TestPermissionRepository_UpdatePermissions_BulkUpdate
// 複数権限の一括更新ができる
func TestPermissionRepository_UpdatePermissions_BulkUpdate(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA2", "sa2@perm-repo-bulk.test", "pass123", "system_admin")

	userAdminRoleID := getRoleIDByName(t, "user_admin")
	viceAdminRoleID := getRoleIDByName(t, "vice_admin")
	noticesFeatureID := getFeatureIDByName(t, "notices")
	filesFeatureID := getFeatureIDByName(t, "files")

	withPermission(t, userAdminRoleID, noticesFeatureID, true, false, false, false, "own_association")
	withPermission(t, viceAdminRoleID, filesFeatureID, true, true, true, true, "own_association")

	repo := permission.NewRepository(testPool)
	err := repo.UpdatePermissions(context.Background(), operatorID, []permission.UpdateInput{
		{
			RoleID: userAdminRoleID, FeatureID: noticesFeatureID,
			CanView: true, CanCreate: true, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
		{
			RoleID: viceAdminRoleID, FeatureID: filesFeatureID,
			CanView: true, CanCreate: false, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
	})
	require.NoError(t, err)

	pc1, err := repo.GetPermissionByRoleAndFeature(context.Background(), "user_admin", "notices")
	require.NoError(t, err)
	assert.True(t, pc1.CanCreate, "user_admin/noticesのcan_createがtrueに更新されているべき")

	pc2, err := repo.GetPermissionByRoleAndFeature(context.Background(), "vice_admin", "files")
	require.NoError(t, err)
	assert.False(t, pc2.CanCreate, "vice_admin/filesのcan_createがfalseに更新されているべき")
}

// TestPermissionRepository_UpdatePermissions_SystemAdminSkipped
// system_adminの権限は変更不可
func TestPermissionRepository_UpdatePermissions_SystemAdminSkipped(t *testing.T) {
	operatorID := insertTestUser(t, nil, "SA3", "sa3@perm-repo-sa.test", "pass123", "system_admin")

	sysAdminRoleID := getRoleIDByName(t, "system_admin")
	noticesFeatureID := getFeatureIDByName(t, "notices")

	// system_adminの現在値を確認（全trueのはず）
	repo := permission.NewRepository(testPool)
	before, err := repo.GetPermissionByRoleAndFeature(context.Background(), "system_admin", "notices")
	require.NoError(t, err)
	require.True(t, before.CanView)

	// system_adminの権限をfalseに変更しようとする
	err = repo.UpdatePermissions(context.Background(), operatorID, []permission.UpdateInput{
		{
			RoleID: sysAdminRoleID, FeatureID: noticesFeatureID,
			CanView: false, CanCreate: false, CanEdit: false, CanDelete: false,
			Scope: "own_association",
		},
	})
	require.NoError(t, err) // エラーにはならず、スキップされる

	// system_adminの値が変わっていないことを確認
	after, err := repo.GetPermissionByRoleAndFeature(context.Background(), "system_admin", "notices")
	require.NoError(t, err)
	assert.Equal(t, before.CanView, after.CanView, "system_adminの権限は変更されていないべき")
	assert.Equal(t, before.CanCreate, after.CanCreate)
}

// ═══════════════════════════════════════════════════════════
// Repository.EmergencyAppointment（緊急任命）
// ═══════════════════════════════════════════════════════════

// TestPermissionRepository_EmergencyAppointment_RoleChanged
// 緊急任命でロールが変更される
func TestPermissionRepository_EmergencyAppointment_RoleChanged(t *testing.T) {
	assocID := insertTestAssociation(t, "緊急任命自治会", "PERM_REPO_EA1")
	operatorID := insertTestUser(t, nil, "SA4", "sa4@perm-repo-ea1.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "対象ユーザー", "target@perm-repo-ea1.test", "pass123", "user")

	repo := permission.NewRepository(testPool)
	err := repo.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "vice_admin",
	})
	require.NoError(t, err)

	// ロールが変更されたか確認
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

// TestPermissionRepository_EmergencyAppointment_OperationLogCreated
// 操作ログが記録される
func TestPermissionRepository_EmergencyAppointment_OperationLogCreated(t *testing.T) {
	assocID := insertTestAssociation(t, "緊急任命ログ自治会", "PERM_REPO_EA2")
	operatorID := insertTestUser(t, nil, "SA5", "sa5@perm-repo-ea2.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "ログ対象ユーザー", "target@perm-repo-ea2.test", "pass123", "user")

	repo := permission.NewRepository(testPool)
	err := repo.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "user_admin",
	})
	require.NoError(t, err)

	// 操作ログが作成されたか確認
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM operation_logs
		 WHERE operator_id = $1 AND operation_type = 'emergency_appointment'
		   AND target_type = 'user' AND target_id = $2`,
		operatorID, targetID.String(),
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "操作ログが1件作成されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}

// TestPermissionRepository_EmergencyAppointment_NoticeCreated
// 対象自治会の全アカウントに通知が作成される
func TestPermissionRepository_EmergencyAppointment_NoticeCreated(t *testing.T) {
	assocID := insertTestAssociation(t, "緊急任命通知自治会", "PERM_REPO_EA3")
	operatorID := insertTestUser(t, nil, "SA6", "sa6@perm-repo-ea3.test", "pass123", "system_admin")
	targetID := insertTestUser(t, &assocID, "通知対象ユーザー", "target@perm-repo-ea3.test", "pass123", "user")

	repo := permission.NewRepository(testPool)
	err := repo.EmergencyAppointment(context.Background(), operatorID, permission.EmergencyAppointmentInput{
		UserID:  targetID,
		NewRole: "association_admin",
	})
	require.NoError(t, err)

	// お知らせが作成されたか確認
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notices
		 WHERE association_id = $1 AND title = '【緊急任命】ロール変更のお知らせ'`,
		assocID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "対象自治会にお知らせが1件作成されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM operation_logs WHERE target_id = $1`, targetID.String())
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notices WHERE association_id = $1`, assocID)
	})
}
