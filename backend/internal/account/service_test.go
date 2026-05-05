package account_test

// サービス層のテスト。
// 「一般ユーザーは登録/編集/削除不可」などのロールチェックはミドルウェアで強制されるため
// ハンドラーテスト（handler_test.go）でカバーしている。
// サービス層ではバリデーション・テナント境界・パスワードハッシュ化をテストする。

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/account"
	"circular-board/internal/domain"
	"circular-board/internal/repository"
)

// ─── サービス初期化ヘルパー ──────────────────────────────────

func newAccountService() *account.Service {
	return account.NewService(account.NewRepository(testPool))
}

// ═══════════════════════════════════════════════════════════
// Service.Create（アカウント登録）
// ═══════════════════════════════════════════════════════════

// TestAccountService_Create_ByAssociationAdmin_Success
// association_admin が自治会内にアカウントを登録できる
func TestAccountService_Create_ByAssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者登録自治会", "ACC_SVC_ADMIN")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@acc-svc-admin.test", "pass123", "association_admin")
	_ = adminID
	svc := newAccountService()

	a, err := svc.Create(context.Background(),
		domain.RoleAssociationAdmin, &assocID,
		account.CreateInput{
			Name:     "新メンバー",
			Email:    "new@acc-svc-admin.test",
			Password: "password123",
			Role:     domain.RoleUser,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, a)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, a.ID)
	})

	assert.Equal(t, assocID, *a.AssociationID, "association_admin の自治会に作成されるべき")
	assert.Equal(t, "新メンバー", a.Name)
	assert.True(t, a.IsActive)
	// パスワードはハッシュ化されている
	assert.NotEqual(t, "password123", a.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte("password123")))
}

// TestAccountService_Create_BySystemAdmin_ToAnyAssociation
// system_admin は任意の自治会にアカウントを登録できる
func TestAccountService_Create_BySystemAdmin_ToAnyAssociation(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "送信先自治会", "ACC_SVC_SYSADM1")
	svc := newAccountService()

	a, err := svc.Create(context.Background(),
		domain.RoleSystemAdmin, nil, // system_admin は assocID=nil
		account.CreateInput{
			Name:          "他自治会メンバー",
			Email:         "other@acc-svc-sysadm1.test",
			Password:      "password123",
			Role:          domain.RoleUser,
			AssociationID: &assoc1ID, // 明示的に自治会を指定
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, a.ID)
	})

	assert.Equal(t, assoc1ID, *a.AssociationID, "指定した自治会に作成されるべき")
}

// TestAccountService_Create_EmailConflict
// メールアドレス重複は ErrEmailConflict
func TestAccountService_Create_EmailConflict(t *testing.T) {
	assocID := insertTestAssociation(t, "重複テスト自治会", "ACC_SVC_CONFLICT")
	insertTestUser(t, &assocID, "既存", "exist@acc-svc-conflict.test", "pass", "user")
	svc := newAccountService()

	_, err := svc.Create(context.Background(),
		domain.RoleAssociationAdmin, &assocID,
		account.CreateInput{
			Name:     "重複ユーザー",
			Email:    "exist@acc-svc-conflict.test",
			Password: "password123",
			Role:     domain.RoleUser,
		},
	)
	assert.True(t, errors.Is(err, account.ErrEmailConflict))
}

// TestAccountService_Create_WeakPassword
// 8文字未満のパスワードは ErrWeakPassword
func TestAccountService_Create_WeakPassword(t *testing.T) {
	assocID := insertTestAssociation(t, "弱パスワード自治会", "ACC_SVC_WEAKPW")
	svc := newAccountService()

	_, err := svc.Create(context.Background(),
		domain.RoleAssociationAdmin, &assocID,
		account.CreateInput{
			Name:     "テスト",
			Email:    "weak@acc-svc-weakpw.test",
			Password: "short", // 8文字未満
			Role:     domain.RoleUser,
		},
	)
	assert.True(t, errors.Is(err, account.ErrWeakPassword))
}

// TestAccountService_Create_EmptyName
// 名前が空はバリデーションエラー
func TestAccountService_Create_EmptyName(t *testing.T) {
	assocID := insertTestAssociation(t, "空名前自治会", "ACC_SVC_EMNAME")
	svc := newAccountService()

	_, err := svc.Create(context.Background(),
		domain.RoleAssociationAdmin, &assocID,
		account.CreateInput{
			Name:     "   ", // 空白のみ
			Email:    "noname@acc-svc-emname.test",
			Password: "password123",
			Role:     domain.RoleUser,
		},
	)
	assert.Error(t, err, "名前が空のときはエラーが返るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.Update（アカウント編集）
// ═══════════════════════════════════════════════════════════

// TestAccountService_Update_ByAssociationAdmin_Success
// association_admin が自治会内のアカウントを編集できる
func TestAccountService_Update_ByAssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "編集テスト自治会", "ACC_SVC_UPDADM")
	targetID := insertTestUser(t, &assocID, "編集前", "before@acc-svc-updadm.test", "pass123", "user")
	svc := newAccountService()

	updated, err := svc.Update(context.Background(),
		domain.RoleAssociationAdmin, &assocID, targetID,
		account.UpdateInput{
			Name:  "編集後",
			Email: "after@acc-svc-updadm.test",
			Role:  domain.RoleAssociationAdmin,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "編集後", updated.Name)
	assert.Equal(t, domain.RoleAssociationAdmin, updated.Role)
}

// TestAccountService_Update_BySystemAdmin_CrossTenant
// system_admin は別自治会のアカウントも編集できる
func TestAccountService_Update_BySystemAdmin_CrossTenant(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "編集元自治会", "ACC_SVC_UPDALL1")
	assoc2ID := insertTestAssociation(t, "編集先自治会", "ACC_SVC_UPDALL2")
	targetID := insertTestUser(t, &assoc2ID, "別自治会ユーザー", "cross@acc-svc-updall2.test", "pass123", "user")
	_ = assoc1ID
	svc := newAccountService()

	updated, err := svc.Update(context.Background(),
		domain.RoleSystemAdmin, nil, targetID, // system_admin は assocID=nil
		account.UpdateInput{
			Name:  "更新済み",
			Email: "cross-updated@acc-svc-updall2.test",
			Role:  domain.RoleUser,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "更新済み", updated.Name)
}

// TestAccountService_Update_CrossTenant_Blocked
// 他自治会のアカウントは ErrNotFound（テナント境界）
func TestAccountService_Update_CrossTenant_Blocked(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "自治会A", "ACC_SVC_UPDCRS1")
	assoc2ID := insertTestAssociation(t, "自治会B", "ACC_SVC_UPDCRS2")
	targetID := insertTestUser(t, &assoc2ID, "B自治会ユーザー", "b@acc-svc-updcrs2.test", "pass123", "user")
	svc := newAccountService()

	_, err := svc.Update(context.Background(),
		domain.RoleAssociationAdmin, &assoc1ID, targetID, // A自治会の管理者がBのユーザーを操作
		account.UpdateInput{Name: "変更", Email: "b@acc-svc-updcrs2.test", Role: domain.RoleUser},
	)
	assert.True(t, errors.Is(err, account.ErrNotFound), "他自治会のアカウントには ErrNotFound が返るべき")
}

// TestAccountService_Update_PasswordUnchanged
// パスワードが空のときはパスワードが変更されない
func TestAccountService_Update_PasswordUnchanged(t *testing.T) {
	assocID := insertTestAssociation(t, "PW不変自治会", "ACC_SVC_NOCHPW")
	targetID := insertTestUser(t, &assocID, "PW不変ユーザー", "nochpw@acc-svc-nochpw.test", "originalpass", "user")
	svc := newAccountService()

	original, err := account.NewRepository(testPool).FindByID(context.Background(), targetID)
	require.NoError(t, err)
	originalHash := original.PasswordHash

	_, err = svc.Update(context.Background(),
		domain.RoleAssociationAdmin, &assocID, targetID,
		account.UpdateInput{
			Name:     "名前変更",
			Email:    "nochpw@acc-svc-nochpw.test",
			Password: "", // 空 = 変更なし
			Role:     domain.RoleUser,
		},
	)
	require.NoError(t, err)

	after, err := account.NewRepository(testPool).FindByID(context.Background(), targetID)
	require.NoError(t, err)
	assert.Equal(t, originalHash, after.PasswordHash, "パスワードが空のとき変更されるべきでない")
}

// ═══════════════════════════════════════════════════════════
// Service.Deactivate（アカウント削除・無効化）
// ═══════════════════════════════════════════════════════════

// TestAccountService_Deactivate_ByAssociationAdmin_Success
// association_admin が自治会内のアカウントを無効化できる
func TestAccountService_Deactivate_ByAssociationAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "無効化自治会", "ACC_SVC_DCTADM")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@acc-svc-dctadm.test", "pass", "association_admin")
	targetID := insertTestUser(t, &assocID, "無効化対象", "target@acc-svc-dctadm.test", "pass", "user")
	svc := newAccountService()

	err := svc.Deactivate(context.Background(),
		domain.RoleAssociationAdmin, &assocID, adminID, targetID)
	require.NoError(t, err)

	got, err := account.NewRepository(testPool).FindByID(context.Background(), targetID)
	require.NoError(t, err)
	assert.False(t, got.IsActive, "無効化後は is_active=false になるべき")
}

// TestAccountService_Deactivate_IsActiveFalse
// 無効化後は is_active が false になる
func TestAccountService_Deactivate_IsActiveFalse(t *testing.T) {
	assocID := insertTestAssociation(t, "状態確認自治会", "ACC_SVC_DCTAFTER")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@acc-svc-dctafter.test", "pass", "association_admin")
	targetID := insertTestUser(t, &assocID, "対象ユーザー", "target@acc-svc-dctafter.test", "pass", "user")
	svc := newAccountService()

	require.NoError(t, svc.Deactivate(context.Background(),
		domain.RoleAssociationAdmin, &assocID, adminID, targetID))

	got, err := account.NewRepository(testPool).FindByID(context.Background(), targetID)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
}

// TestAccountService_Deactivate_CrossTenant_Blocked
// 他自治会のアカウントの無効化は ErrNotFound
func TestAccountService_Deactivate_CrossTenant_Blocked(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "自治会C", "ACC_SVC_DCTCRS1")
	assoc2ID := insertTestAssociation(t, "自治会D", "ACC_SVC_DCTCRS2")
	adminID := insertTestUser(t, &assoc1ID, "C管理者", "admin@acc-svc-dctcrs1.test", "pass", "association_admin")
	targetID := insertTestUser(t, &assoc2ID, "D一般ユーザー", "target@acc-svc-dctcrs2.test", "pass", "user")
	svc := newAccountService()

	err := svc.Deactivate(context.Background(),
		domain.RoleAssociationAdmin, &assoc1ID, adminID, targetID)
	assert.True(t, errors.Is(err, account.ErrNotFound), "他自治会のアカウントはErrNotFoundが返るべき")
}

// TestAccountService_Deactivate_Self_Blocked
// 自分自身の無効化は ErrSelfDeactivation
func TestAccountService_Deactivate_Self_Blocked(t *testing.T) {
	assocID := insertTestAssociation(t, "自己削除自治会", "ACC_SVC_SELF")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@acc-svc-self.test", "pass", "association_admin")
	svc := newAccountService()

	err := svc.Deactivate(context.Background(),
		domain.RoleAssociationAdmin, &assocID,
		adminID, adminID, // callerUserID == targetID
	)
	assert.True(t, errors.Is(err, account.ErrSelfDeactivation), "自分自身の無効化は ErrSelfDeactivation が返るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.Activate（アカウント有効化）
// ═══════════════════════════════════════════════════════════

// TestAccountService_Activate_Success
// 無効化されたアカウントを有効化できる
func TestAccountService_Activate_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "有効化テスト自治会", "ACC_SVC_ACTADM")
	targetID := insertTestUser(t, &assocID, "有効化対象", "act@acc-svc-actadm.test", "pass", "user")
	svc := newAccountService()

	// まず無効化
	_, _ = testPool.Exec(context.Background(),
		`UPDATE users SET is_active = false WHERE id = $1`, targetID)

	err := svc.Activate(context.Background(),
		domain.RoleAssociationAdmin, &assocID, targetID)
	require.NoError(t, err)

	got, err := account.NewRepository(testPool).FindByID(context.Background(), targetID)
	require.NoError(t, err)
	assert.True(t, got.IsActive, "有効化後は is_active=true になるべき")
}

// TestAccountService_Activate_CrossTenant_Blocked
// 他自治会のアカウントの有効化は ErrNotFound
func TestAccountService_Activate_CrossTenant_Blocked(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "自治会E", "ACC_SVC_ACTCRS1")
	assoc2ID := insertTestAssociation(t, "自治会F", "ACC_SVC_ACTCRS2")
	targetID := insertTestUser(t, &assoc2ID, "F一般ユーザー", "f@acc-svc-actcrs2.test", "pass", "user")
	svc := newAccountService()

	err := svc.Activate(context.Background(),
		domain.RoleAssociationAdmin, &assoc1ID, targetID)
	assert.True(t, errors.Is(err, account.ErrNotFound))
}

// TestAccountService_Deactivate_LoginBlocked
// 無効化されたアカウントはログインできなくなる
func TestAccountService_Deactivate_LoginBlocked(t *testing.T) {
	assocID := insertTestAssociation(t, "ログイン不可自治会", "ACC_SVC_LGBLK")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@acc-svc-lgblk.test", "pass123", "association_admin")
	targetID := insertTestUser(t, &assocID, "無効化対象", "target@acc-svc-lgblk.test", "pass123", "user")
	svc := newAccountService()

	err := svc.Deactivate(context.Background(),
		domain.RoleAssociationAdmin, &assocID, adminID, targetID)
	require.NoError(t, err)

	// FindByEmailAndAssociationCode は is_active=true のユーザーのみ返す
	userRepo := repository.NewUserRepository(testPool)
	_, err = userRepo.FindByEmailAndAssociationCode(
		context.Background(), "target@acc-svc-lgblk.test", "ACC_SVC_LGBLK")
	assert.ErrorIs(t, err, repository.ErrNotFound, "無効化後はログイン不可になるべき")
}
