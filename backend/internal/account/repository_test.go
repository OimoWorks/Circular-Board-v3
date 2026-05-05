package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/account"
	"circular-board/internal/domain"
)

// ═══════════════════════════════════════════════════════════
// Repository.Create（アカウント作成）
// ═══════════════════════════════════════════════════════════

// TestAccountRepository_Create_Success
// 正しいデータでアカウントを作成できる
func TestAccountRepository_Create_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "作成テスト自治会", "ACC_REPO_C1")
	repo := account.NewRepository(testPool)

	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.MinCost)
	now := time.Now()
	id := uuid.New()
	a := &account.Account{
		ID:            id,
		AssociationID: &assocID,
		Name:          "田中太郎",
		Email:         "tanaka@acc-repo-c1.test",
		PasswordHash:  string(hash),
		Role:          domain.RoleUser,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.Create(context.Background(), a)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, assocID, *got.AssociationID)
	assert.Equal(t, "田中太郎", got.Name)
	assert.Equal(t, "tanaka@acc-repo-c1.test", got.Email)
	assert.Equal(t, domain.RoleUser, got.Role)
	assert.True(t, got.IsActive)
}

// TestAccountRepository_Create_PasswordHashStored
// パスワードはハッシュ化されて保存され、平文ではない
func TestAccountRepository_Create_PasswordHashStored(t *testing.T) {
	assocID := insertTestAssociation(t, "ハッシュ確認自治会", "ACC_REPO_HASH")
	repo := account.NewRepository(testPool)

	plainPassword := "secret1234"
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.MinCost)
	id := uuid.New()
	a := &account.Account{
		ID:            id,
		AssociationID: &assocID,
		Name:          "ハッシュユーザー",
		Email:         "hash@acc-repo-hash.test",
		PasswordHash:  string(hash),
		Role:          domain.RoleUser,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), a)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})

	got, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	// 保存されたハッシュは平文パスワードと異なる
	assert.NotEqual(t, plainPassword, got.PasswordHash, "平文パスワードが保存されてはいけない")
	// bcrypt で検証できる
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte(plainPassword)))
}

// TestAccountRepository_Create_EmailConflict
// 同一自治会内でメールアドレス重複するとエラー
func TestAccountRepository_Create_EmailConflict(t *testing.T) {
	assocID := insertTestAssociation(t, "重複確認自治会", "ACC_REPO_DUP")
	insertTestUser(t, &assocID, "既存ユーザー", "dup@acc-repo-dup.test", "pass123", "user")
	repo := account.NewRepository(testPool)

	id := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass456"), bcrypt.MinCost)
	a := &account.Account{
		ID:            id,
		AssociationID: &assocID,
		Name:          "重複ユーザー",
		Email:         "dup@acc-repo-dup.test", // 同じメール
		PasswordHash:  string(hash),
		Role:          domain.RoleUser,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), a)
	assert.True(t, errors.Is(err, account.ErrEmailConflict), "メール重複は ErrEmailConflict が返るべき")
}

// TestAccountRepository_Create_InvalidAssociationFK
// 存在しない association_id への挿入は FK 制約エラー
func TestAccountRepository_Create_InvalidAssociationFK(t *testing.T) {
	repo := account.NewRepository(testPool)
	nonExistentAssocID := uuid.New() // DBに存在しない自治会ID

	id := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)
	a := &account.Account{
		ID:            id,
		AssociationID: &nonExistentAssocID,
		Name:          "不正ユーザー",
		Email:         "invalid@acc-repo-fk.test",
		PasswordHash:  string(hash),
		Role:          domain.RoleUser,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), a)
	assert.Error(t, err, "存在しない自治会への挿入はエラーが返るべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.List（アカウント一覧）
// ═══════════════════════════════════════════════════════════

// TestAccountRepository_List_ByAssociation
// association_id で絞り込みができる
func TestAccountRepository_List_ByAssociation(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧絞込自治会", "ACC_REPO_LST1")
	insertTestUser(t, &assocID, "メンバー1", "m1@acc-repo-lst1.test", "pass", "user")
	insertTestUser(t, &assocID, "メンバー2", "m2@acc-repo-lst1.test", "pass", "user")
	repo := account.NewRepository(testPool)

	accounts, err := repo.List(context.Background(), &assocID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(accounts), 2, "自治会のメンバーが2件以上返るべき")
	for _, a := range accounts {
		assert.NotNil(t, a.AssociationID)
		assert.Equal(t, assocID, *a.AssociationID, "他自治会のアカウントが混入してはいけない")
	}
}

// TestAccountRepository_List_AllAssociations
// nil を渡すと全自治会のアカウントを取得できる（system_admin 用）
func TestAccountRepository_List_AllAssociations(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "全件自治会A", "ACC_REPO_ALL1")
	assoc2ID := insertTestAssociation(t, "全件自治会B", "ACC_REPO_ALL2")
	insertTestUser(t, &assoc1ID, "A会員", "a@acc-repo-all1.test", "pass", "user")
	insertTestUser(t, &assoc2ID, "B会員", "b@acc-repo-all2.test", "pass", "user")
	repo := account.NewRepository(testPool)

	accounts, err := repo.List(context.Background(), nil) // nil = 全件
	require.NoError(t, err)

	// 両自治会のIDが結果に含まれることを確認
	var foundAssoc1, foundAssoc2 bool
	for _, a := range accounts {
		if a.AssociationID != nil && *a.AssociationID == assoc1ID {
			foundAssoc1 = true
		}
		if a.AssociationID != nil && *a.AssociationID == assoc2ID {
			foundAssoc2 = true
		}
	}
	assert.True(t, foundAssoc1, "自治会Aのアカウントが含まれるべき")
	assert.True(t, foundAssoc2, "自治会Bのアカウントが含まれるべき")
}

// TestAccountRepository_List_TenantIsolation
// 自治会Aで絞り込むと自治会Bのデータは混入しない
func TestAccountRepository_List_TenantIsolation(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "分離自治会A", "ACC_REPO_ISO1")
	assoc2ID := insertTestAssociation(t, "分離自治会B", "ACC_REPO_ISO2")
	insertTestUser(t, &assoc1ID, "A会員", "a@acc-repo-iso1.test", "pass", "user")
	insertTestUser(t, &assoc2ID, "B会員", "b@acc-repo-iso2.test", "pass", "user")
	repo := account.NewRepository(testPool)

	accounts, err := repo.List(context.Background(), &assoc1ID)
	require.NoError(t, err)
	for _, a := range accounts {
		require.NotNil(t, a.AssociationID)
		assert.Equal(t, assoc1ID, *a.AssociationID, "自治会Bのデータが混入してはいけない")
	}
}

// ═══════════════════════════════════════════════════════════
// Repository.FindByID（単件取得）
// ═══════════════════════════════════════════════════════════

// TestAccountRepository_FindByID_Success
// ID でアカウントを取得できる
func TestAccountRepository_FindByID_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "単件自治会", "ACC_REPO_FIND")
	userID := insertTestUser(t, &assocID, "検索対象", "find@acc-repo-find.test", "pass", "user")
	repo := account.NewRepository(testPool)

	got, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
	assert.Equal(t, "検索対象", got.Name)
}

// TestAccountRepository_FindByID_NotFound
// 存在しない ID は ErrNotFound
func TestAccountRepository_FindByID_NotFound(t *testing.T) {
	repo := account.NewRepository(testPool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, account.ErrNotFound), "存在しないIDはErrNotFoundが返るべき")
}

// TestAccountRepository_FindByID_AnyTenant
// FindByID はテナント境界を持たず他自治会のアカウントも取得できる（境界はサービス層で強制）
func TestAccountRepository_FindByID_AnyTenant(t *testing.T) {
	assoc1ID := insertTestAssociation(t, "テナントA", "ACC_REPO_TENA")
	assoc2ID := insertTestAssociation(t, "テナントB", "ACC_REPO_TENB")
	userOfAssoc2 := insertTestUser(t, &assoc2ID, "B自治会ユーザー", "b@acc-repo-tenb.test", "pass", "user")
	_ = assoc1ID
	repo := account.NewRepository(testPool)

	// リポジトリは assocID を持たないため他自治会のアカウントも取得できる
	got, err := repo.FindByID(context.Background(), userOfAssoc2)
	require.NoError(t, err)
	assert.Equal(t, userOfAssoc2, got.ID, "FindByIDはテナント境界なしで取得できるべき（境界はサービス層）")
}

// ═══════════════════════════════════════════════════════════
// Repository.Update（アカウント更新）
// ═══════════════════════════════════════════════════════════

// TestAccountRepository_Update_Success
// 名前・ロールを更新できる
func TestAccountRepository_Update_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "更新テスト自治会", "ACC_REPO_UPD")
	userID := insertTestUser(t, &assocID, "更新前", "upd@acc-repo-upd.test", "pass", "user")
	repo := account.NewRepository(testPool)

	got, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)

	got.Name = "更新後"
	got.Role = domain.RoleAssociationAdmin
	got.UpdatedAt = time.Now()

	err = repo.Update(context.Background(), got)
	require.NoError(t, err)

	updated, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "更新後", updated.Name)
	assert.Equal(t, domain.RoleAssociationAdmin, updated.Role)
}

// TestAccountRepository_Update_PasswordChange
// パスワードを変更するとハッシュが更新される
func TestAccountRepository_Update_PasswordChange(t *testing.T) {
	assocID := insertTestAssociation(t, "パスワード更新自治会", "ACC_REPO_UPDHASH")
	userID := insertTestUser(t, &assocID, "PW変更ユーザー", "pwchg@acc-repo-updhash.test", "oldpass", "user")
	repo := account.NewRepository(testPool)

	got, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	oldHash := got.PasswordHash

	newHash, _ := bcrypt.GenerateFromPassword([]byte("newpass123"), bcrypt.MinCost)
	got.PasswordHash = string(newHash)
	got.UpdatedAt = time.Now()

	require.NoError(t, repo.Update(context.Background(), got))

	updated, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	assert.NotEqual(t, oldHash, updated.PasswordHash, "ハッシュが更新されるべき")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("newpass123")))
}

// TestAccountRepository_Update_NotFound
// 存在しない ID の更新は ErrNotFound
func TestAccountRepository_Update_NotFound(t *testing.T) {
	repo := account.NewRepository(testPool)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	a := &account.Account{
		ID:           uuid.New(),
		Name:         "存在しない",
		Email:        "notfound@test.test",
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
		UpdatedAt:    time.Now(),
	}
	err := repo.Update(context.Background(), a)
	assert.True(t, errors.Is(err, account.ErrNotFound))
}

// ═══════════════════════════════════════════════════════════
// Repository.SetActive（有効/無効化）
// ═══════════════════════════════════════════════════════════

// TestAccountRepository_SetActive_Deactivate
// is_active を false に切り替えられる
func TestAccountRepository_SetActive_Deactivate(t *testing.T) {
	assocID := insertTestAssociation(t, "無効化自治会", "ACC_REPO_DACT")
	userID := insertTestUser(t, &assocID, "無効化対象", "dact@acc-repo-dact.test", "pass", "user")
	repo := account.NewRepository(testPool)

	require.NoError(t, repo.SetActive(context.Background(), userID, false))

	got, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, got.IsActive, "is_active が false になるべき")
}

// TestAccountRepository_SetActive_Activate
// is_active を true に切り替えられる
func TestAccountRepository_SetActive_Activate(t *testing.T) {
	assocID := insertTestAssociation(t, "有効化自治会", "ACC_REPO_ACT")
	userID := insertTestUser(t, &assocID, "有効化対象", "act@acc-repo-act.test", "pass", "user")
	repo := account.NewRepository(testPool)

	// まず無効化
	require.NoError(t, repo.SetActive(context.Background(), userID, false))
	// 再度有効化
	require.NoError(t, repo.SetActive(context.Background(), userID, true))

	got, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, got.IsActive, "is_active が true に戻るべき")
}

// TestAccountRepository_SetActive_NotFound
// 存在しない ID の操作は ErrNotFound
func TestAccountRepository_SetActive_NotFound(t *testing.T) {
	repo := account.NewRepository(testPool)
	err := repo.SetActive(context.Background(), uuid.New(), false)
	assert.True(t, errors.Is(err, account.ErrNotFound))
}
