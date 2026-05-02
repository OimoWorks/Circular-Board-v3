package files_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/files"
)

// ═══════════════════════════════════════════════════════════
// Repository.Create（ファイルメタデータの保存）
// ═══════════════════════════════════════════════════════════

// TestRepository_Create_Success
// 正しいデータでファイルレコードを保存できる
func TestRepository_Create_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "リポジトリ登録自治会", "REPO_FILE_C1")
	userID := insertTestUser(t, &assocID, "登録ユーザー", "create@repo-c1.test", "pass123", "association_admin")

	repo := files.NewRepository(testPool)
	id := uuid.New()
	storedName := id.String() + ".pdf"
	storagePath := os.TempDir() + "/" + storedName
	now := time.Now().Truncate(time.Millisecond)

	f := &files.File{
		ID:               id,
		AssociationID:    assocID,
		Year:             2024,
		Month:            6,
		Filename:         storedName,
		OriginalFilename: "テスト資料.pdf",
		StoragePath:      storagePath,
		UploadedBy:       userID,
		FileSize:         2048,
		MimeType:         "application/pdf",
		CreatedAt:        now,
	}

	err := repo.Create(context.Background(), f)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, id)
	})

	got, err := repo.FindByID(context.Background(), id, assocID)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, assocID, got.AssociationID)
	assert.Equal(t, 2024, got.Year)
	assert.Equal(t, 6, got.Month)
	assert.Equal(t, "テスト資料.pdf", got.OriginalFilename)
	assert.Equal(t, int64(2048), got.FileSize)
	assert.Equal(t, "application/pdf", got.MimeType)
	assert.Nil(t, got.DeletedAt)
}

// TestRepository_Create_MissingAssociationID
// 存在しない association_id での保存は外部キー制約でエラー
func TestRepository_Create_MissingAssociationID(t *testing.T) {
	assocID := insertTestAssociation(t, "存在確認自治会", "REPO_FILE_C2")
	userID := insertTestUser(t, &assocID, "制約ユーザー", "fk@repo-c2.test", "pass123", "user")

	repo := files.NewRepository(testPool)
	id := uuid.New()
	nonExistentAssocID := uuid.New() // DBに存在しない自治会ID

	f := &files.File{
		ID:               id,
		AssociationID:    nonExistentAssocID,
		Year:             2024,
		Month:            1,
		Filename:         id.String() + ".pdf",
		OriginalFilename: "制約テスト.pdf",
		StoragePath:      os.TempDir() + "/" + id.String() + ".pdf",
		UploadedBy:       userID,
		FileSize:         100,
		MimeType:         "application/pdf",
		CreatedAt:        time.Now(),
	}

	err := repo.Create(context.Background(), f)
	assert.Error(t, err, "存在しない自治会IDへの挿入は外部キー制約でエラーになるべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.FindByID（ファイル単件取得）
// ═══════════════════════════════════════════════════════════

// TestRepository_FindByID_Success
// IDと自治会IDでファイルを取得できる
func TestRepository_FindByID_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "単件取得自治会", "REPO_FILE_F1")
	userID := insertTestUser(t, &assocID, "取得ユーザー", "find@repo-f1.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, userID, 2024, 4, "取得テスト.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	got, err := repo.FindByID(context.Background(), fileID, assocID)

	require.NoError(t, err)
	assert.Equal(t, fileID, got.ID)
	assert.Equal(t, assocID, got.AssociationID)
	assert.Equal(t, "取得テスト.pdf", got.OriginalFilename)
	assert.Equal(t, 2024, got.Year)
	assert.Equal(t, 4, got.Month)
}

// TestRepository_FindByID_NotFound
// 存在しないIDは ErrNotFound が返る
func TestRepository_FindByID_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "不在取得自治会", "REPO_FILE_NF")

	repo := files.NewRepository(testPool)
	_, err := repo.FindByID(context.Background(), uuid.New(), assocID)

	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// TestRepository_FindByID_CrossTenant
// 異なる自治会IDでは取得できない（テナント境界の強制）
func TestRepository_FindByID_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "自治会A-単件テナント", "REPO_FILE_TA")
	assocB := insertTestAssociation(t, "自治会B-単件テナント", "REPO_FILE_TB")
	userA := insertTestUser(t, &assocA, "TAユーザー", "find@repo-ta.test", "pass123", "user")
	fileID := insertTestFile(t, assocA, userA, 2024, 1, "資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)

	// 正しい自治会IDで取得できる
	got, err := repo.FindByID(context.Background(), fileID, assocA)
	require.NoError(t, err)
	assert.Equal(t, fileID, got.ID)

	// 別の自治会IDでは取得できない
	_, errB := repo.FindByID(context.Background(), fileID, assocB)
	assert.True(t, errors.Is(errB, files.ErrNotFound),
		"他の自治会IDではファイルを取得できてはいけない")
}

// ═══════════════════════════════════════════════════════════
// Repository.List（ファイル一覧取得）
// ═══════════════════════════════════════════════════════════

// TestRepository_List_ByAssociationID
// association_id で絞り込める
func TestRepository_List_ByAssociationID(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧テスト自治会", "REPO_LIST_1")
	userID := insertTestUser(t, &assocID, "一覧ユーザー", "list@repo-1.test", "pass123", "user")

	id1 := insertTestFile(t, assocID, userID, 2024, 1, "1月資料.pdf", "application/pdf")
	id2 := insertTestFile(t, assocID, userID, 2024, 3, "3月資料.pdf", "application/pdf")
	id3 := insertTestFile(t, assocID, userID, 2023, 12, "12月資料.jpg", "image/jpeg")

	repo := files.NewRepository(testPool)
	list, err := repo.List(context.Background(), assocID, files.ListFilter{Year: 0, Month: 0})
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)
	for _, f := range list {
		ids[f.ID] = true
		assert.Equal(t, assocID, f.AssociationID)
	}
	assert.True(t, ids[id1], "id1 が一覧に含まれるべき")
	assert.True(t, ids[id2], "id2 が一覧に含まれるべき")
	assert.True(t, ids[id3], "id3 が一覧に含まれるべき")
}

// TestRepository_List_FilterByYear
// 年フィルタで絞り込める
func TestRepository_List_FilterByYear(t *testing.T) {
	assocID := insertTestAssociation(t, "年フィルタ自治会", "REPO_LIST_YR")
	userID := insertTestUser(t, &assocID, "年フィルタユーザー", "yr@repo-yr.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2024, 5, "2024資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2023, 5, "2023資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	list, err := repo.List(context.Background(), assocID, files.ListFilter{Year: 2024, Month: 0})
	require.NoError(t, err)
	require.NotEmpty(t, list)

	for _, f := range list {
		assert.Equal(t, 2024, f.Year, "2024年のみが返るべき")
	}
}

// TestRepository_List_FilterByYearAndMonth
// 年月フィルタで絞り込める
func TestRepository_List_FilterByYearAndMonth(t *testing.T) {
	assocID := insertTestAssociation(t, "年月フィルタ自治会", "REPO_LIST_YM")
	userID := insertTestUser(t, &assocID, "年月ユーザー", "ym@repo-ym.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2024, 6, "6月資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 7, "7月資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	list, err := repo.List(context.Background(), assocID, files.ListFilter{Year: 2024, Month: 6})
	require.NoError(t, err)
	require.NotEmpty(t, list)

	for _, f := range list {
		assert.Equal(t, 2024, f.Year)
		assert.Equal(t, 6, f.Month, "6月のみが返るべき")
	}
}

// TestRepository_List_Empty
// データがない場合は空スライスが返る（nil ではなく長さ0）
func TestRepository_List_Empty(t *testing.T) {
	assocID := insertTestAssociation(t, "空一覧自治会", "REPO_LIST_EMPTY")

	repo := files.NewRepository(testPool)
	list, err := repo.List(context.Background(), assocID, files.ListFilter{})
	require.NoError(t, err)
	assert.Empty(t, list)
}

// TestRepository_List_CrossTenant
// 別の自治会のファイルが混入しない
func TestRepository_List_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "自治会Aリスト", "REPO_LIST_TA")
	assocB := insertTestAssociation(t, "自治会Bリスト", "REPO_LIST_TB")
	userA := insertTestUser(t, &assocA, "ユーザーAL", "al@repo-list-a.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "ユーザーBL", "bl@repo-list-b.test", "pass123", "user")

	_ = insertTestFile(t, assocA, userA, 2024, 4, "自治会A資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocB, userB, 2024, 4, "自治会B資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	listA, err := repo.List(context.Background(), assocA, files.ListFilter{})
	require.NoError(t, err)

	for _, f := range listA {
		assert.Equal(t, assocA, f.AssociationID,
			"自治会Aの一覧に自治会Bのファイルが含まれてはいけない")
	}
}

// TestRepository_List_OrderByYearMonthDesc
// 年月降順・作成日時降順で返る
func TestRepository_List_OrderByYearMonthDesc(t *testing.T) {
	assocID := insertTestAssociation(t, "ソート自治会", "REPO_LIST_SORT")
	userID := insertTestUser(t, &assocID, "ソートユーザー", "sort@repo-sort.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2023, 1, "古い資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 6, "新しい資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	list, err := repo.List(context.Background(), assocID, files.ListFilter{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list), 2)

	// 先頭が最新（年が大きい）であることを確認
	first := list[0]
	last := list[len(list)-1]
	assert.GreaterOrEqual(t, first.Year*100+first.Month, last.Year*100+last.Month,
		"年月降順で返るべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.SoftDelete（論理削除）
// ═══════════════════════════════════════════════════════════

// TestRepository_SoftDelete_SetsDeletedAt
// 論理削除後は FindByID で取得できず、deleted_at が設定される
func TestRepository_SoftDelete_SetsDeletedAt(t *testing.T) {
	assocID := insertTestAssociation(t, "削除テスト自治会", "REPO_DEL_1")
	userID := insertTestUser(t, &assocID, "削除ユーザー", "del@repo-del-1.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, userID, 2024, 2, "削除資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	err := repo.SoftDelete(context.Background(), fileID, assocID)
	require.NoError(t, err)

	// FindByID では取得できない（deleted_at IS NULL 条件）
	_, errFind := repo.FindByID(context.Background(), fileID, assocID)
	assert.True(t, errors.Is(errFind, files.ErrNotFound),
		"論理削除後はFindByIDで取得できない")

	// DBに deleted_at が実際にセットされていることを直接確認
	var deletedAt *time.Time
	err = testPool.QueryRow(context.Background(),
		`SELECT deleted_at FROM files WHERE id = $1`, fileID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "deleted_at がセットされているべき")
}

// TestRepository_SoftDelete_ExcludedFromList
// 削除済みファイルは一覧に出ない
func TestRepository_SoftDelete_ExcludedFromList(t *testing.T) {
	assocID := insertTestAssociation(t, "削除一覧自治会", "REPO_DEL_LIST")
	userID := insertTestUser(t, &assocID, "削除一覧ユーザー", "dellist@repo-del-list.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, userID, 2024, 3, "削除対象.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	require.NoError(t, repo.SoftDelete(context.Background(), fileID, assocID))

	list, err := repo.List(context.Background(), assocID, files.ListFilter{})
	require.NoError(t, err)

	for _, f := range list {
		assert.NotEqual(t, fileID, f.ID, "削除済みファイルは一覧に含まれてはいけない")
	}
}

// TestRepository_SoftDelete_NotFound
// 存在しないIDの削除は ErrNotFound が返る
func TestRepository_SoftDelete_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "削除不在自治会", "REPO_DEL_NF")

	repo := files.NewRepository(testPool)
	err := repo.SoftDelete(context.Background(), uuid.New(), assocID)

	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// TestRepository_SoftDelete_CrossTenant
// 他の自治会のファイルは削除できない
func TestRepository_SoftDelete_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "REPO_DEL_TA")
	assocB := insertTestAssociation(t, "削除テナントB", "REPO_DEL_TB")
	userA := insertTestUser(t, &assocA, "削除TA", "del@repo-del-ta.test", "pass123", "user")
	fileID := insertTestFile(t, assocA, userA, 2024, 3, "テナントA資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	// 自治会Bの権限で自治会Aのファイルを削除しようとすると ErrNotFound
	err := repo.SoftDelete(context.Background(), fileID, assocB)
	assert.True(t, errors.Is(err, files.ErrNotFound),
		"他の自治会IDでは削除できない")

	// 自治会Aのファイルは削除されていない
	got, err := repo.FindByID(context.Background(), fileID, assocA)
	require.NoError(t, err)
	assert.Nil(t, got.DeletedAt, "他テナントからの削除操作でファイルが消えてはいけない")
}

// ═══════════════════════════════════════════════════════════
// Repository.AvailableYears（利用可能な年一覧）
// ═══════════════════════════════════════════════════════════

// TestRepository_AvailableYears_ReturnsDescending
// ファイルが存在する年の一覧が降順で返る
func TestRepository_AvailableYears_ReturnsDescending(t *testing.T) {
	assocID := insertTestAssociation(t, "年一覧自治会", "REPO_YEARS_1")
	userID := insertTestUser(t, &assocID, "年一覧ユーザー", "yrs@repo-years.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2022, 1, "2022資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2023, 6, "2023資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 11, "2024資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	years, err := repo.AvailableYears(context.Background(), assocID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(years), 3)

	for i := 1; i < len(years); i++ {
		assert.Greater(t, years[i-1], years[i], "年は降順で返るべき")
	}
	assert.Contains(t, years, 2022)
	assert.Contains(t, years, 2023)
	assert.Contains(t, years, 2024)
}

// TestRepository_AvailableYears_Empty
// ファイルがない場合は空スライスが返る
func TestRepository_AvailableYears_Empty(t *testing.T) {
	assocID := insertTestAssociation(t, "空の年一覧自治会", "REPO_YEARS_EMPTY")

	repo := files.NewRepository(testPool)
	years, err := repo.AvailableYears(context.Background(), assocID)
	require.NoError(t, err)
	assert.Empty(t, years)
}

// TestRepository_AvailableYears_ExcludeDeleted
// 論理削除済みファイルの年は一覧に含まれない
func TestRepository_AvailableYears_ExcludeDeleted(t *testing.T) {
	assocID := insertTestAssociation(t, "削除除外年自治会", "REPO_YEARS_DEL")
	userID := insertTestUser(t, &assocID, "削除除外ユーザー", "del@repo-years-del.test", "pass123", "user")

	fileID := insertTestFile(t, assocID, userID, 2098, 1, "未来資料.pdf", "application/pdf")

	repo := files.NewRepository(testPool)
	// 削除前は 2098 年が含まれる
	yearsBefore, err := repo.AvailableYears(context.Background(), assocID)
	require.NoError(t, err)
	assert.Contains(t, yearsBefore, 2098)

	// 論理削除後は 2098 年が含まれない
	require.NoError(t, repo.SoftDelete(context.Background(), fileID, assocID))
	yearsAfter, err := repo.AvailableYears(context.Background(), assocID)
	require.NoError(t, err)
	assert.NotContains(t, yearsAfter, 2098, "削除済みファイルの年は含まれない")
}
