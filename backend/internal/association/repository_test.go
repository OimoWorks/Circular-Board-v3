package association_test

// リポジトリ層のテスト。
// DB の CRUD 操作・ユニーク制約・NOT NULL 制約を直接検証する。

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/association"
	"circular-board/internal/repository"
)

// ─── リポジトリ初期化ヘルパー ────────────────────────────────

func newAssociationRepo() *association.Repository {
	return association.NewRepository(testPool)
}

// ═══════════════════════════════════════════════════════════
// Repository.Create（自治会登録）
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_Create_Success
// 正常に自治会を登録できる
func TestAssociationRepo_Create_Success(t *testing.T) {
	repo := newAssociationRepo()
	now := time.Now()
	a := &association.Association{
		ID:        uuid.New(),
		Name:      "テスト自治会",
		Code:      "ASSOC_REPO_C1",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, a.ID)
	})

	err := repo.Create(context.Background(), a)
	require.NoError(t, err)

	got, err := repo.FindByID(context.Background(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, "テスト自治会", got.Name)
	assert.Equal(t, "ASSOC_REPO_C1", got.Code)
	assert.True(t, got.IsActive)
}

// TestAssociationRepo_Create_CodeConflict
// コード重複は ErrCodeConflict
func TestAssociationRepo_Create_CodeConflict(t *testing.T) {
	_ = insertTestAssociation(t, "重複コード自治会", "ASSOC_REPO_DUP")
	repo := newAssociationRepo()

	now := time.Now()
	a := &association.Association{
		ID:        uuid.New(),
		Name:      "重複コード2",
		Code:      "ASSOC_REPO_DUP",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, a.ID)
	})

	err := repo.Create(context.Background(), a)
	assert.ErrorIs(t, err, association.ErrCodeConflict)
}

// ═══════════════════════════════════════════════════════════
// Repository.List（自治会一覧）
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_List_ReturnsAll
// 複数自治会が全件返る
func TestAssociationRepo_List_ReturnsAll(t *testing.T) {
	id1 := insertTestAssociation(t, "一覧自治会1", "ASSOC_REPO_LST1")
	id2 := insertTestAssociation(t, "一覧自治会2", "ASSOC_REPO_LST2")
	repo := newAssociationRepo()

	items, err := repo.List(context.Background())
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)
	for _, a := range items {
		ids[a.ID] = true
	}
	assert.True(t, ids[id1], "自治会1が含まれるべき")
	assert.True(t, ids[id2], "自治会2が含まれるべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.FindByID（自治会取得）
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_FindByID_Success
// ID で自治会を取得できる
func TestAssociationRepo_FindByID_Success(t *testing.T) {
	id := insertTestAssociation(t, "取得テスト自治会", "ASSOC_REPO_FIND")
	repo := newAssociationRepo()

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "ASSOC_REPO_FIND", got.Code)
}

// TestAssociationRepo_FindByID_NotFound
// 存在しない ID は ErrNotFound
func TestAssociationRepo_FindByID_NotFound(t *testing.T) {
	repo := newAssociationRepo()

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, association.ErrNotFound)
}

// ═══════════════════════════════════════════════════════════
// Repository.Update（自治会更新）
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_Update_Success
// 自治会情報を更新できる
func TestAssociationRepo_Update_Success(t *testing.T) {
	id := insertTestAssociation(t, "更新前自治会", "ASSOC_REPO_UPD")
	repo := newAssociationRepo()

	a, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)

	a.Name = "更新後自治会"
	a.Code = "ASSOC_REPO_UPDNEW"
	a.UpdatedAt = time.Now()

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM associations WHERE code = 'ASSOC_REPO_UPDNEW'`)
	})

	err = repo.Update(context.Background(), a)
	require.NoError(t, err)

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "更新後自治会", got.Name)
	assert.Equal(t, "ASSOC_REPO_UPDNEW", got.Code)
}

// TestAssociationRepo_Update_CodeConflict
// 更新時のコード重複は ErrCodeConflict
func TestAssociationRepo_Update_CodeConflict(t *testing.T) {
	_ = insertTestAssociation(t, "既存コード自治会", "ASSOC_REPO_UDPCF")
	id2 := insertTestAssociation(t, "更新対象自治会", "ASSOC_REPO_UPD2")
	repo := newAssociationRepo()

	a, err := repo.FindByID(context.Background(), id2)
	require.NoError(t, err)
	a.Code = "ASSOC_REPO_UDPCF" // 既存コードと衝突
	a.UpdatedAt = time.Now()

	err = repo.Update(context.Background(), a)
	assert.ErrorIs(t, err, association.ErrCodeConflict)
}

// TestAssociationRepo_Update_NotFound
// 存在しない ID の更新は ErrNotFound
func TestAssociationRepo_Update_NotFound(t *testing.T) {
	repo := newAssociationRepo()

	a := &association.Association{
		ID:        uuid.New(),
		Name:      "ゴースト",
		Code:      "ASSOC_REPO_GHOST",
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), a)
	assert.ErrorIs(t, err, association.ErrNotFound)
}

// ═══════════════════════════════════════════════════════════
// Repository.SetActive（有効化・無効化）
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_SetActive_Deactivate
// 有効な自治会を無効化できる
func TestAssociationRepo_SetActive_Deactivate(t *testing.T) {
	id := insertTestAssociation(t, "無効化テスト自治会", "ASSOC_REPO_DACT")
	repo := newAssociationRepo()

	err := repo.SetActive(context.Background(), id, false)
	require.NoError(t, err)

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
}

// TestAssociationRepo_SetActive_Activate
// 無効化された自治会を有効化できる
func TestAssociationRepo_SetActive_Activate(t *testing.T) {
	id := insertTestAssociation(t, "有効化テスト自治会", "ASSOC_REPO_ACT")
	repo := newAssociationRepo()

	// まず無効化
	_, _ = testPool.Exec(context.Background(),
		`UPDATE associations SET is_active = false WHERE id = $1`, id)

	err := repo.SetActive(context.Background(), id, true)
	require.NoError(t, err)

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.True(t, got.IsActive)
}

// TestAssociationRepo_SetActive_NotFound
// 存在しない ID の状態変更は ErrNotFound
func TestAssociationRepo_SetActive_NotFound(t *testing.T) {
	repo := newAssociationRepo()

	err := repo.SetActive(context.Background(), uuid.New(), false)
	assert.ErrorIs(t, err, association.ErrNotFound)
}

// ═══════════════════════════════════════════════════════════
// 追加テスト：一覧・ログイン遮断
// ═══════════════════════════════════════════════════════════

// TestAssociationRepo_List_IncludesInactive
// 一覧には無効化された自治会も含まれる（管理用途）
func TestAssociationRepo_List_IncludesInactive(t *testing.T) {
	activeID := insertTestAssociation(t, "有効自治会", "ASSOC_REPO_ACTIVE")
	inactiveID := insertTestAssociation(t, "無効自治会", "ASSOC_REPO_INACT")
	repo := newAssociationRepo()

	// inactiveID を無効化
	_, err := testPool.Exec(context.Background(),
		`UPDATE associations SET is_active = false WHERE id = $1`, inactiveID)
	require.NoError(t, err)

	items, err := repo.List(context.Background())
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)
	for _, a := range items {
		ids[a.ID] = true
	}
	assert.True(t, ids[activeID], "有効な自治会が含まれるべき")
	assert.True(t, ids[inactiveID], "無効化された自治会も管理用に含まれるべき")
}

// TestAssociationRepo_Deactivate_BlocksLogin
// 無効化した自治会のユーザーはログインできなくなる
func TestAssociationRepo_Deactivate_BlocksLogin(t *testing.T) {
	id := insertTestAssociation(t, "ログイン遮断自治会", "ASSOC_REPO_LGBLK")
	_ = insertTestUser(t, &id, "ユーザー", "user@assoc-repo-lgblk.test", "pass123", "user")
	repo := newAssociationRepo()

	// 自治会を無効化
	err := repo.SetActive(context.Background(), id, false)
	require.NoError(t, err)

	// FindByEmailAndAssociationCode はアクティブな自治会のユーザーのみ返す
	userRepo := repository.NewUserRepository(testPool)
	_, err = userRepo.FindByEmailAndAssociationCode(
		context.Background(), "user@assoc-repo-lgblk.test", "ASSOC_REPO_LGBLK")
	assert.ErrorIs(t, err, repository.ErrNotFound, "無効化した自治会のユーザーはログイン不可になるべき")
}
