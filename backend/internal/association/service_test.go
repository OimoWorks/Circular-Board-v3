package association_test

// サービス層のテスト。
// ロールチェックはミドルウェアで強制されるためハンドラーテスト（handler_test.go）でカバーしている。
// サービス層ではバリデーション・コード形式・重複チェックをテストする。

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/association"
	"circular-board/internal/repository"
)

// ─── サービス初期化ヘルパー ──────────────────────────────────

func newAssociationService() *association.Service {
	return association.NewService(association.NewRepository(testPool))
}

// ═══════════════════════════════════════════════════════════
// Service.Create（自治会登録）
// ═══════════════════════════════════════════════════════════

// TestAssociationService_Create_Success
// 正常に自治会を登録できる
func TestAssociationService_Create_Success(t *testing.T) {
	svc := newAssociationService()

	a, err := svc.Create(context.Background(), association.CreateInput{
		Name: "新自治会",
		Code: "ASSOC_SVC_C1",
	})
	require.NoError(t, err)
	require.NotNil(t, a)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, a.ID)
	})

	assert.Equal(t, "新自治会", a.Name)
	assert.Equal(t, "ASSOC_SVC_C1", a.Code)
	assert.True(t, a.IsActive)
}

// TestAssociationService_Create_AutoUppercase
// コードは自動的に大文字に変換される
func TestAssociationService_Create_AutoUppercase(t *testing.T) {
	svc := newAssociationService()

	a, err := svc.Create(context.Background(), association.CreateInput{
		Name: "大文字変換自治会",
		Code: "assoc_svc_upper",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, a.ID)
	})

	assert.Equal(t, "ASSOC_SVC_UPPER", a.Code, "コードは大文字に変換されるべき")
}

// TestAssociationService_Create_CodeConflict
// コード重複は ErrCodeConflict
func TestAssociationService_Create_CodeConflict(t *testing.T) {
	_ = insertTestAssociation(t, "重複コード自治会", "ASSOC_SVC_DUP")
	svc := newAssociationService()

	_, err := svc.Create(context.Background(), association.CreateInput{
		Name: "重複コード2",
		Code: "ASSOC_SVC_DUP",
	})
	assert.True(t, errors.Is(err, association.ErrCodeConflict))
}

// TestAssociationService_Create_InvalidCode
// 無効なコード形式は ErrInvalidCode
func TestAssociationService_Create_InvalidCode(t *testing.T) {
	svc := newAssociationService()

	_, err := svc.Create(context.Background(), association.CreateInput{
		Name: "無効コード自治会",
		Code: "invalid code!", // スペースや記号は不可
	})
	assert.True(t, errors.Is(err, association.ErrInvalidCode))
}

// TestAssociationService_Create_EmptyCode
// 空のコードは ErrInvalidCode
func TestAssociationService_Create_EmptyCode(t *testing.T) {
	svc := newAssociationService()

	_, err := svc.Create(context.Background(), association.CreateInput{
		Name: "空コード自治会",
		Code: "",
	})
	assert.True(t, errors.Is(err, association.ErrInvalidCode))
}

// TestAssociationService_Create_EmptyName
// 空の名前はバリデーションエラー
func TestAssociationService_Create_EmptyName(t *testing.T) {
	svc := newAssociationService()

	_, err := svc.Create(context.Background(), association.CreateInput{
		Name: "   ", // 空白のみ
		Code: "ASSOC_SVC_ENAME",
	})
	assert.Error(t, err, "名前が空のときはエラーが返るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.Update（自治会編集）
// ═══════════════════════════════════════════════════════════

// TestAssociationService_Update_Success
// 自治会情報を更新できる
func TestAssociationService_Update_Success(t *testing.T) {
	id := insertTestAssociation(t, "更新前自治会", "ASSOC_SVC_UPD")
	svc := newAssociationService()

	updated, err := svc.Update(context.Background(), id, association.UpdateInput{
		Name: "更新後自治会",
		Code: "ASSOC_SVC_UPDNEW",
	})
	require.NoError(t, err)
	assert.Equal(t, "更新後自治会", updated.Name)
	assert.Equal(t, "ASSOC_SVC_UPDNEW", updated.Code)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE code = 'ASSOC_SVC_UPDNEW'`)
	})
}

// TestAssociationService_Update_NotFound
// 存在しない ID は ErrNotFound
func TestAssociationService_Update_NotFound(t *testing.T) {
	svc := newAssociationService()

	_, err := svc.Update(context.Background(), nonExistentID(), association.UpdateInput{
		Name: "名前",
		Code: "ASSOC_SVC_UPDNF",
	})
	assert.True(t, errors.Is(err, association.ErrNotFound))
}

// TestAssociationService_Update_CodeConflict
// 更新時のコード重複は ErrCodeConflict
func TestAssociationService_Update_CodeConflict(t *testing.T) {
	_ = insertTestAssociation(t, "既存コード自治会", "ASSOC_SVC_CF1")
	id2 := insertTestAssociation(t, "更新対象自治会", "ASSOC_SVC_CF2")
	svc := newAssociationService()

	_, err := svc.Update(context.Background(), id2, association.UpdateInput{
		Name: "名前",
		Code: "ASSOC_SVC_CF1",
	})
	assert.True(t, errors.Is(err, association.ErrCodeConflict))
}

// ═══════════════════════════════════════════════════════════
// Service.Activate / Service.Deactivate
// ═══════════════════════════════════════════════════════════

// TestAssociationService_Deactivate_Success
// 有効な自治会を無効化できる
func TestAssociationService_Deactivate_Success(t *testing.T) {
	id := insertTestAssociation(t, "無効化テスト自治会", "ASSOC_SVC_DACT")
	svc := newAssociationService()

	err := svc.Deactivate(context.Background(), id)
	require.NoError(t, err)

	got, err := association.NewRepository(testPool).FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, got.IsActive, "無効化後は is_active=false になるべき")
}

// TestAssociationService_Deactivate_NotFound
// 存在しない ID の無効化は ErrNotFound
func TestAssociationService_Deactivate_NotFound(t *testing.T) {
	svc := newAssociationService()

	err := svc.Deactivate(context.Background(), nonExistentID())
	assert.True(t, errors.Is(err, association.ErrNotFound))
}

// TestAssociationService_Activate_Success
// 無効化された自治会を有効化できる
func TestAssociationService_Activate_Success(t *testing.T) {
	id := insertTestAssociation(t, "有効化テスト自治会", "ASSOC_SVC_ACT")
	svc := newAssociationService()

	// まず無効化
	_, _ = testPool.Exec(context.Background(),
		`UPDATE associations SET is_active = false WHERE id = $1`, id)

	err := svc.Activate(context.Background(), id)
	require.NoError(t, err)

	got, err := association.NewRepository(testPool).FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.True(t, got.IsActive, "有効化後は is_active=true になるべき")
}

// TestAssociationService_Activate_NotFound
// 存在しない ID の有効化は ErrNotFound
func TestAssociationService_Activate_NotFound(t *testing.T) {
	svc := newAssociationService()

	err := svc.Activate(context.Background(), nonExistentID())
	assert.True(t, errors.Is(err, association.ErrNotFound))
}

// nonExistentID は DB に存在しない UUID を返す
func nonExistentID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000000")
}

// ═══════════════════════════════════════════════════════════
// 追加テスト：無効化後の副作用
// ═══════════════════════════════════════════════════════════

// TestAssociationService_Deactivate_UsersStillExist
// 自治会を無効化してもユーザーデータは削除されない
func TestAssociationService_Deactivate_UsersStillExist(t *testing.T) {
	assocID := insertTestAssociation(t, "ユーザー残存自治会", "ASSOC_SVC_USREXIST")
	userID := insertTestUser(t, &assocID, "残存ユーザー", "user@assoc-svc-usrexist.test", "pass123", "user")
	svc := newAssociationService()

	err := svc.Deactivate(context.Background(), assocID)
	require.NoError(t, err)

	// ユーザーレコードはDBに残っている（論理削除）
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE id = $1`, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "自治会無効化後もユーザーレコードは存在するべき")
}

// TestAssociationService_Deactivate_LoginBlocked
// 無効化した自治会のユーザーはログインできなくなる
func TestAssociationService_Deactivate_LoginBlocked(t *testing.T) {
	assocID := insertTestAssociation(t, "ログイン遮断自治会", "ASSOC_SVC_LGBLK")
	_ = insertTestUser(t, &assocID, "ユーザー", "user@assoc-svc-lgblk.test", "pass123", "user")
	svc := newAssociationService()

	err := svc.Deactivate(context.Background(), assocID)
	require.NoError(t, err)

	// ログイン試行 - 無効化された自治会のユーザーは見つからない
	userRepo := repository.NewUserRepository(testPool)
	_, err = userRepo.FindByEmailAndAssociationCode(
		context.Background(), "user@assoc-svc-lgblk.test", "ASSOC_SVC_LGBLK")
	assert.ErrorIs(t, err, repository.ErrNotFound, "無効化後はその自治会のユーザーはログイン不可になるべき")
}
