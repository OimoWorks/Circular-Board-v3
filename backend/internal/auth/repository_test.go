package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/repository"
)

// ═══════════════════════════════════════════════════════════
// UserRepository
// ═══════════════════════════════════════════════════════════

// TestUserRepository_FindByEmailAndAssociationCode_Found
// 正しいメールアドレスと自治会コードでユーザーが取得できる
func TestUserRepository_FindByEmailAndAssociationCode_Found(t *testing.T) {
	assocID := insertTestAssociation(t, "テスト自治会A", "REPO_TEST_A")
	_ = insertTestUser(t, &assocID, "山田太郎", "yamada@repo-a.test", "pass123", "user")

	repo := repository.NewUserRepository(testPool)
	got, err := repo.FindByEmailAndAssociationCode(context.Background(), "yamada@repo-a.test", "REPO_TEST_A")

	require.NoError(t, err)
	assert.Equal(t, "yamada@repo-a.test", got.Email)
	assert.Equal(t, "山田太郎", got.Name)
	assert.Equal(t, "user", string(got.Role))
	assert.True(t, got.IsActive)
	require.NotNil(t, got.AssociationID)
	assert.Equal(t, assocID, *got.AssociationID)
	assert.Equal(t, "テスト自治会A", *got.AssociationName)
	assert.Equal(t, "REPO_TEST_A", *got.AssociationCode)
}

// TestUserRepository_FindByEmailAndAssociationCode_NotFound
// 存在しないメールアドレスで ErrNotFound が返る
func TestUserRepository_FindByEmailAndAssociationCode_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "テスト自治会B", "REPO_TEST_B")
	_ = assocID // 自治会だけ存在しユーザーは存在しない

	repo := repository.NewUserRepository(testPool)
	_, err := repo.FindByEmailAndAssociationCode(context.Background(), "no-user@repo-b.test", "REPO_TEST_B")

	assert.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestUserRepository_FindByEmailAndAssociationCode_WrongAssocCode
// 存在するユーザーでも自治会コードが違えば取得できない
func TestUserRepository_FindByEmailAndAssociationCode_WrongAssocCode(t *testing.T) {
	assocID := insertTestAssociation(t, "テスト自治会C", "REPO_TEST_C")
	_ = insertTestUser(t, &assocID, "鈴木花子", "suzuki@repo-c.test", "pass123", "user")

	repo := repository.NewUserRepository(testPool)
	_, err := repo.FindByEmailAndAssociationCode(context.Background(), "suzuki@repo-c.test", "WRONG_CODE")

	assert.True(t, errors.Is(err, repository.ErrNotFound),
		"異なる自治会コードでは ErrNotFound が返るべき")
}

// TestUserRepository_FindByEmailForSystemAdmin_Found
// system_admin はメールアドレスのみで取得できる
func TestUserRepository_FindByEmailForSystemAdmin_Found(t *testing.T) {
	_ = insertTestUser(t, nil, "スーパー管理者", "sysadmin@repo.test", "pass123", "system_admin")

	repo := repository.NewUserRepository(testPool)
	got, err := repo.FindByEmailForSystemAdmin(context.Background(), "sysadmin@repo.test")

	require.NoError(t, err)
	assert.Equal(t, "system_admin", string(got.Role))
	assert.Nil(t, got.AssociationID, "system_admin は自治会に属さない")
}

// TestUserRepository_FindByEmailForSystemAdmin_NotSystemAdmin
// system_admin 以外はこのメソッドで取得できない
func TestUserRepository_FindByEmailForSystemAdmin_NotSystemAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "テスト自治会D", "REPO_TEST_D")
	_ = insertTestUser(t, &assocID, "一般さん", "user@repo-d.test", "pass123", "user")

	repo := repository.NewUserRepository(testPool)
	_, err := repo.FindByEmailForSystemAdmin(context.Background(), "user@repo-d.test")

	assert.True(t, errors.Is(err, repository.ErrNotFound),
		"user ロールは FindByEmailForSystemAdmin で取得できない")
}

// TestUserRepository_FindByID_Found
// IDでユーザーが取得できる
func TestUserRepository_FindByID_Found(t *testing.T) {
	assocID := insertTestAssociation(t, "テスト自治会E", "REPO_TEST_E")
	userID := insertTestUser(t, &assocID, "田中次郎", "tanaka@repo-e.test", "pass123", "association_admin")

	repo := repository.NewUserRepository(testPool)
	got, err := repo.FindByID(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
	assert.Equal(t, "田中次郎", got.Name)
	assert.Equal(t, "association_admin", string(got.Role))
}

// TestUserRepository_FindByID_NotFound
// 存在しないIDで ErrNotFound が返る
func TestUserRepository_FindByID_NotFound(t *testing.T) {
	repo := repository.NewUserRepository(testPool)
	_, err := repo.FindByID(context.Background(), uuid.New())

	assert.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestUserRepository_MultiTenant_AssociationIsolation
// あるユーザーを別の自治会コードでは取得できないこと（テナント境界の確認）
func TestUserRepository_MultiTenant_AssociationIsolation(t *testing.T) {
	assocA := insertTestAssociation(t, "自治会A-テナント", "TENANT_A")
	assocB := insertTestAssociation(t, "自治会B-テナント", "TENANT_B")
	_ = insertTestUser(t, &assocA, "テナントAユーザー", "member@tenant-a.test", "pass123", "user")
	_ = insertTestUser(t, &assocB, "テナントBユーザー", "member@tenant-b.test", "pass123", "user")

	repo := repository.NewUserRepository(testPool)

	// 自治会Aのユーザーを自治会Bのコードで引くと取得できない
	_, errAB := repo.FindByEmailAndAssociationCode(context.Background(), "member@tenant-a.test", "TENANT_B")
	assert.True(t, errors.Is(errAB, repository.ErrNotFound),
		"テナントAユーザーをテナントBコードで取得できてはいけない")

	// 自治会Bのユーザーを自治会Aのコードで引くと取得できない
	_, errBA := repo.FindByEmailAndAssociationCode(context.Background(), "member@tenant-b.test", "TENANT_A")
	assert.True(t, errors.Is(errBA, repository.ErrNotFound),
		"テナントBユーザーをテナントAコードで取得できてはいけない")

	// 正しいコードで引けばそれぞれ取得できる
	gotA, err := repo.FindByEmailAndAssociationCode(context.Background(), "member@tenant-a.test", "TENANT_A")
	require.NoError(t, err)
	assert.Equal(t, "テナントAユーザー", gotA.Name)

	gotB, err := repo.FindByEmailAndAssociationCode(context.Background(), "member@tenant-b.test", "TENANT_B")
	require.NoError(t, err)
	assert.Equal(t, "テナントBユーザー", gotB.Name)
}

// ═══════════════════════════════════════════════════════════
// RefreshTokenRepository
// ═══════════════════════════════════════════════════════════

// TestRefreshTokenRepository_CreateAndFindByHash
// トークンを保存してハッシュで取得できる
func TestRefreshTokenRepository_CreateAndFindByHash(t *testing.T) {
	assocID := insertTestAssociation(t, "リフレッシュ自治会", "RTREPO_1")
	userID := insertTestUser(t, &assocID, "トークンユーザー", "token@rtrepo-1.test", "pass123", "user")

	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	rawToken := "raw-test-token-" + uuid.New().String()
	tokenHash := sha256Hex(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	err := tokenRepo.Create(context.Background(), userID, tokenHash, expiresAt)
	require.NoError(t, err)

	got, err := tokenRepo.FindByTokenHash(context.Background(), tokenHash)
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, tokenHash, got.TokenHash)
	assert.WithinDuration(t, expiresAt, got.ExpiresAt, time.Second)
}

// TestRefreshTokenRepository_FindByHash_NotFound
// 存在しないハッシュで ErrNotFound が返る
func TestRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	_, err := tokenRepo.FindByTokenHash(context.Background(), "nonexistent-hash")

	assert.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestRefreshTokenRepository_FindByHash_Expired
// 期限切れトークンは取得できない
func TestRefreshTokenRepository_FindByHash_Expired(t *testing.T) {
	assocID := insertTestAssociation(t, "期限切れ自治会", "RTREPO_EXP")
	userID := insertTestUser(t, &assocID, "期限切れユーザー", "exp@rtrepo-exp.test", "pass123", "user")

	rawToken := "expired-token-" + uuid.New().String()
	insertExpiredRefreshToken(t, userID, rawToken)

	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	_, err := tokenRepo.FindByTokenHash(context.Background(), sha256Hex(rawToken))

	assert.True(t, errors.Is(err, repository.ErrNotFound),
		"期限切れトークンは ErrNotFound が返るべき")
}

// TestRefreshTokenRepository_DeleteByUserID
// ユーザーIDでトークンを一括削除できる
func TestRefreshTokenRepository_DeleteByUserID(t *testing.T) {
	assocID := insertTestAssociation(t, "削除テスト自治会", "RTREPO_DEL")
	userID := insertTestUser(t, &assocID, "削除ユーザー", "del@rtrepo-del.test", "pass123", "user")

	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	hash1 := sha256Hex("token1-" + uuid.New().String())
	hash2 := sha256Hex("token2-" + uuid.New().String())
	exp := time.Now().Add(7 * 24 * time.Hour)
	require.NoError(t, tokenRepo.Create(context.Background(), userID, hash1, exp))
	require.NoError(t, tokenRepo.Create(context.Background(), userID, hash2, exp))

	err := tokenRepo.DeleteByUserID(context.Background(), userID)
	require.NoError(t, err)

	_, err1 := tokenRepo.FindByTokenHash(context.Background(), hash1)
	_, err2 := tokenRepo.FindByTokenHash(context.Background(), hash2)
	assert.True(t, errors.Is(err1, repository.ErrNotFound), "削除後はトークン1が見つからないはず")
	assert.True(t, errors.Is(err2, repository.ErrNotFound), "削除後はトークン2が見つからないはず")
}

// TestRefreshTokenRepository_DeleteByTokenHash
// 特定ハッシュのトークンだけを削除できる
func TestRefreshTokenRepository_DeleteByTokenHash(t *testing.T) {
	assocID := insertTestAssociation(t, "選択削除自治会", "RTREPO_SEL")
	userID := insertTestUser(t, &assocID, "選択削除ユーザー", "sel@rtrepo-sel.test", "pass123", "user")

	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	hashToDelete := sha256Hex("delete-this-" + uuid.New().String())
	hashToKeep := sha256Hex("keep-this-" + uuid.New().String())
	exp := time.Now().Add(7 * 24 * time.Hour)
	require.NoError(t, tokenRepo.Create(context.Background(), userID, hashToDelete, exp))
	require.NoError(t, tokenRepo.Create(context.Background(), userID, hashToKeep, exp))

	err := tokenRepo.DeleteByTokenHash(context.Background(), hashToDelete)
	require.NoError(t, err)

	_, errDeleted := tokenRepo.FindByTokenHash(context.Background(), hashToDelete)
	assert.True(t, errors.Is(errDeleted, repository.ErrNotFound), "削除したトークンは見つからないはず")

	kept, errKept := tokenRepo.FindByTokenHash(context.Background(), hashToKeep)
	require.NoError(t, errKept, "削除していないトークンは残っているはず")
	assert.Equal(t, hashToKeep, kept.TokenHash)

	// 残ったトークンをクリーンアップ
	_ = tokenRepo.DeleteByUserID(context.Background(), userID)
}

// TestRefreshTokenRepository_DeleteExpired
// 期限切れトークンの一括削除
func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	assocID := insertTestAssociation(t, "期限切れ一括削除自治会", "RTREPO_DELEXP")
	userID := insertTestUser(t, &assocID, "一括削除ユーザー", "bulkdel@rtrepo.test", "pass123", "user")

	rawExpired := "bulk-expired-" + uuid.New().String()
	insertExpiredRefreshToken(t, userID, rawExpired)

	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	err := tokenRepo.DeleteExpired(context.Background())
	require.NoError(t, err)

	_, errFind := tokenRepo.FindByTokenHash(context.Background(), sha256Hex(rawExpired))
	assert.True(t, errors.Is(errFind, repository.ErrNotFound), "DeleteExpired後は期限切れトークンが消えているはず")
}
