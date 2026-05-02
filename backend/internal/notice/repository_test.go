package notice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/notice"
)

// ═══════════════════════════════════════════════════════════
// Repository.Create（お知らせ作成）
// ═══════════════════════════════════════════════════════════

// TestNoticeRepository_Create_Success
// 正しいデータでお知らせを作成できる
func TestNoticeRepository_Create_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "お知らせ作成自治会", "NOT_REPO_C1")
	userID := insertTestUser(t, &assocID, "作成ユーザー", "create@not-repo-c1.test", "pass123", "association_admin")

	repo := notice.NewRepository(testPool)
	now := time.Now().Truncate(time.Millisecond)
	id := uuid.New()
	n := &notice.Notice{
		ID:            id,
		AssociationID: assocID,
		Title:         "テスト告知",
		Body:          "テスト本文です。",
		IsPinned:      false,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.Create(context.Background(), n)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, id)
	})

	got, err := repo.FindByID(context.Background(), id, assocID)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, assocID, got.AssociationID)
	assert.Equal(t, "テスト告知", got.Title)
	assert.Equal(t, "テスト本文です。", got.Body)
	assert.False(t, got.IsPinned)
	assert.Equal(t, userID, got.CreatedBy)
	assert.Nil(t, got.DeletedAt)
}

// TestNoticeRepository_Create_WithPinned
// ピン留めフラグが正しく保存される
func TestNoticeRepository_Create_WithPinned(t *testing.T) {
	assocID := insertTestAssociation(t, "ピン留め作成自治会", "NOT_REPO_PIN")
	userID := insertTestUser(t, &assocID, "ピン管理者", "pin@not-repo-pin.test", "pass123", "association_admin")

	repo := notice.NewRepository(testPool)
	now := time.Now()
	id := uuid.New()
	n := &notice.Notice{
		ID:            id,
		AssociationID: assocID,
		Title:         "重要なお知らせ",
		Body:          "重要な内容です。",
		IsPinned:      true,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.Create(context.Background(), n)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, id)
	})

	got, err := repo.FindByID(context.Background(), id, assocID)
	require.NoError(t, err)
	assert.True(t, got.IsPinned, "IsPinnedがtrueで保存されるべき")
}

// TestNoticeRepository_Create_MissingAssociationID
// 存在しないassociation_idへの挿入は外部キー制約でエラー
func TestNoticeRepository_Create_MissingAssociationID(t *testing.T) {
	assocID := insertTestAssociation(t, "FK確認自治会", "NOT_REPO_FK")
	userID := insertTestUser(t, &assocID, "FKユーザー", "fk@not-repo-fk.test", "pass123", "user")

	repo := notice.NewRepository(testPool)
	now := time.Now()
	id := uuid.New()
	n := &notice.Notice{
		ID:            id,
		AssociationID: uuid.New(), // 存在しない自治会ID
		Title:         "エラーテスト",
		Body:          "エラー本文",
		IsPinned:      false,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.Create(context.Background(), n)
	assert.Error(t, err, "存在しない自治会IDへの挿入は外部キー制約でエラー")
}

// ═══════════════════════════════════════════════════════════
// Repository.List（お知らせ一覧取得）
// ═══════════════════════════════════════════════════════════

// TestNoticeRepository_List_ByAssociationID
// association_idで絞り込める
func TestNoticeRepository_List_ByAssociationID(t *testing.T) {
	assocID := insertTestAssociation(t, "お知らせ一覧自治会", "NOT_REPO_L1")
	userID := insertTestUser(t, &assocID, "一覧ユーザー", "list@not-repo-l1.test", "pass123", "user")

	id1 := insertTestNotice(t, assocID, userID, "お知らせ1", "本文1", false)
	id2 := insertTestNotice(t, assocID, userID, "お知らせ2", "本文2", false)

	repo := notice.NewRepository(testPool)
	items, err := repo.List(context.Background(), assocID, userID)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)
	for _, n := range items {
		ids[n.ID] = true
		assert.Equal(t, assocID, n.AssociationID)
	}
	assert.True(t, ids[id1], "id1が一覧に含まれるべき")
	assert.True(t, ids[id2], "id2が一覧に含まれるべき")
}

// TestNoticeRepository_List_PinnedFirst
// ピン留めが上部に表示される
func TestNoticeRepository_List_PinnedFirst(t *testing.T) {
	assocID := insertTestAssociation(t, "ピン順自治会", "NOT_REPO_PIN_ORD")
	userID := insertTestUser(t, &assocID, "ピン順ユーザー", "pin@not-repo-pin-ord.test", "pass123", "user")

	_ = insertTestNotice(t, assocID, userID, "通常お知らせ", "通常本文", false)
	pinnedID := insertTestNotice(t, assocID, userID, "重要！", "重要本文", true)

	repo := notice.NewRepository(testPool)
	items, err := repo.List(context.Background(), assocID, userID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(items), 2)

	assert.Equal(t, pinnedID, items[0].ID, "ピン留めされたお知らせが先頭に来るべき")
	assert.True(t, items[0].IsPinned)
}

// TestNoticeRepository_List_Empty
// データがない場合は空スライスが返る
func TestNoticeRepository_List_Empty(t *testing.T) {
	assocID := insertTestAssociation(t, "空お知らせ自治会", "NOT_REPO_EMPTY")
	userID := insertTestUser(t, &assocID, "空ユーザー", "empty@not-repo-empty.test", "pass123", "user")

	repo := notice.NewRepository(testPool)
	items, err := repo.List(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

// TestNoticeRepository_List_CrossTenant
// 他自治会のお知らせが混入しない
func TestNoticeRepository_List_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "自治会A一覧", "NOT_REPO_LTA")
	assocB := insertTestAssociation(t, "自治会B一覧", "NOT_REPO_LTB")
	userA := insertTestUser(t, &assocA, "ユーザーA", "a@not-repo-lta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "ユーザーB", "b@not-repo-ltb.test", "pass123", "user")

	_ = insertTestNotice(t, assocA, userA, "自治会Aのお知らせ", "A本文", false)
	_ = insertTestNotice(t, assocB, userB, "自治会Bのお知らせ", "B本文", false)

	repo := notice.NewRepository(testPool)
	itemsA, err := repo.List(context.Background(), assocA, userA)
	require.NoError(t, err)

	for _, n := range itemsA {
		assert.Equal(t, assocA, n.AssociationID,
			"自治会Aの一覧に自治会Bのお知らせが含まれてはいけない")
	}
}

// ═══════════════════════════════════════════════════════════
// Repository.FindByID（お知らせ詳細取得）
// ═══════════════════════════════════════════════════════════

// TestNoticeRepository_FindByID_Success
// IDで取得できる
func TestNoticeRepository_FindByID_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "詳細取得自治会", "NOT_REPO_FID1")
	userID := insertTestUser(t, &assocID, "詳細ユーザー", "find@not-repo-fid1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, userID, "詳細確認", "詳細本文", false)

	repo := notice.NewRepository(testPool)
	got, err := repo.FindByID(context.Background(), noticeID, assocID)

	require.NoError(t, err)
	assert.Equal(t, noticeID, got.ID)
	assert.Equal(t, assocID, got.AssociationID)
	assert.Equal(t, "詳細確認", got.Title)
	assert.Equal(t, "詳細本文", got.Body)
	assert.Nil(t, got.DeletedAt)
}

// TestNoticeRepository_FindByID_NotFound
// 存在しないIDはErrNotFoundが返る
func TestNoticeRepository_FindByID_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "不在詳細自治会", "NOT_REPO_NF")

	repo := notice.NewRepository(testPool)
	_, err := repo.FindByID(context.Background(), uuid.New(), assocID)

	assert.True(t, errors.Is(err, notice.ErrNotFound))
}

// TestNoticeRepository_FindByID_CrossTenant
// 他自治会のお知らせはアクセス不可
func TestNoticeRepository_FindByID_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "自治会A詳細", "NOT_REPO_FTA")
	assocB := insertTestAssociation(t, "自治会B詳細", "NOT_REPO_FTB")
	userA := insertTestUser(t, &assocA, "ユーザーA詳細", "a@not-repo-fta.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)

	repo := notice.NewRepository(testPool)

	// 正しい自治会IDで取得できる
	got, err := repo.FindByID(context.Background(), noticeID, assocA)
	require.NoError(t, err)
	assert.Equal(t, noticeID, got.ID)

	// 別の自治会IDでは取得できない
	_, errB := repo.FindByID(context.Background(), noticeID, assocB)
	assert.True(t, errors.Is(errB, notice.ErrNotFound),
		"他の自治会IDではお知らせを取得できてはいけない")
}

// ═══════════════════════════════════════════════════════════
// Repository.SoftDelete（論理削除）
// ═══════════════════════════════════════════════════════════

// TestNoticeRepository_SoftDelete_SetsDeletedAt
// deleted_atが設定される
func TestNoticeRepository_SoftDelete_SetsDeletedAt(t *testing.T) {
	assocID := insertTestAssociation(t, "削除テスト自治会", "NOT_REPO_DEL1")
	userID := insertTestUser(t, &assocID, "削除ユーザー", "del@not-repo-del1.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, userID, "削除対象", "削除本文", false)

	repo := notice.NewRepository(testPool)
	err := repo.SoftDelete(context.Background(), noticeID, assocID)
	require.NoError(t, err)

	// FindByIDでは取得できない（deleted_at IS NULL 条件）
	_, errFind := repo.FindByID(context.Background(), noticeID, assocID)
	assert.True(t, errors.Is(errFind, notice.ErrNotFound), "論理削除後はFindByIDで取得できない")

	// deleted_atが実際にセットされているかDB直接確認
	var deletedAt *time.Time
	err = testPool.QueryRow(context.Background(),
		`SELECT deleted_at FROM notices WHERE id = $1`, noticeID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "deleted_atがセットされているべき")
}

// TestNoticeRepository_SoftDelete_ExcludedFromList
// 削除済みは一覧に出ない
func TestNoticeRepository_SoftDelete_ExcludedFromList(t *testing.T) {
	assocID := insertTestAssociation(t, "削除一覧自治会", "NOT_REPO_DLIST")
	userID := insertTestUser(t, &assocID, "削除一覧ユーザー", "dellist@not-repo-dlist.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, userID, "削除対象お知らせ", "本文", false)

	repo := notice.NewRepository(testPool)
	require.NoError(t, repo.SoftDelete(context.Background(), noticeID, assocID))

	items, err := repo.List(context.Background(), assocID, userID)
	require.NoError(t, err)
	for _, n := range items {
		assert.NotEqual(t, noticeID, n.ID, "削除済みお知らせは一覧に含まれてはいけない")
	}
}

// TestNoticeRepository_SoftDelete_CrossTenant
// 他自治会のお知らせは削除不可
func TestNoticeRepository_SoftDelete_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "NOT_REPO_DTA")
	assocB := insertTestAssociation(t, "削除テナントB", "NOT_REPO_DTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@not-repo-dta.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)

	repo := notice.NewRepository(testPool)
	// 自治会Bとして削除しようとすると ErrNotFound
	err := repo.SoftDelete(context.Background(), noticeID, assocB)
	assert.True(t, errors.Is(err, notice.ErrNotFound), "他の自治会IDでは削除できない")

	// 自治会Aのお知らせはまだ存在している
	got, err := repo.FindByID(context.Background(), noticeID, assocA)
	require.NoError(t, err)
	assert.Nil(t, got.DeletedAt, "他テナントからの削除操作でお知らせが消えてはいけない")
}

// ═══════════════════════════════════════════════════════════
// Repository.MarkAsRead / Repository.UnreadCount（既読管理）
// ═══════════════════════════════════════════════════════════

// TestNoticeRepository_MarkAsRead_Success
// 既読が正しく保存される
func TestNoticeRepository_MarkAsRead_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "既読自治会", "NOT_REPO_READ1")
	userID := insertTestUser(t, &assocID, "既読ユーザー", "read@not-repo-read1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, userID, "既読対象", "本文", false)

	repo := notice.NewRepository(testPool)

	// 既読前は未読
	countBefore, err := repo.UnreadCount(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countBefore, 1, "既読前は未読件数が1以上")

	// 既読にする
	err = repo.MarkAsRead(context.Background(), noticeID, userID)
	require.NoError(t, err)

	// DBにレコードが作成されているか確認
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notice_reads WHERE notice_id = $1 AND user_id = $2`,
		noticeID, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "notice_readsに1件登録されるべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeRepository_MarkAsRead_Idempotent
// 同じユーザーが2回既読にしても重複しない
func TestNoticeRepository_MarkAsRead_Idempotent(t *testing.T) {
	assocID := insertTestAssociation(t, "重複既読自治会", "NOT_REPO_READ2")
	userID := insertTestUser(t, &assocID, "重複既読ユーザー", "read2@not-repo-read2.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, userID, "重複既読対象", "本文", false)

	repo := notice.NewRepository(testPool)

	// 1回目
	require.NoError(t, repo.MarkAsRead(context.Background(), noticeID, userID))
	// 2回目（ON CONFLICT DO NOTHING なのでエラーにならない）
	require.NoError(t, repo.MarkAsRead(context.Background(), noticeID, userID))

	// DBのレコードは1件だけ
	var count int
	err := testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM notice_reads WHERE notice_id = $1 AND user_id = $2`,
		noticeID, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "2回既読にしてもnotice_readsは1件のまま")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeRepository_UnreadCount_Correct
// 未読件数が正しく取得できる
func TestNoticeRepository_UnreadCount_Correct(t *testing.T) {
	assocID := insertTestAssociation(t, "未読件数自治会", "NOT_REPO_UNRD")
	userID := insertTestUser(t, &assocID, "未読ユーザー", "unrd@not-repo-unrd.test", "pass123", "user")

	n1 := insertTestNotice(t, assocID, userID, "未読1", "本文1", false)
	n2 := insertTestNotice(t, assocID, userID, "未読2", "本文2", false)
	_ = n2

	repo := notice.NewRepository(testPool)

	// 全件未読の場合
	countAll, err := repo.UnreadCount(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countAll, 2, "2件以上未読であるべき")

	// n1を既読にすると未読件数が1減る
	require.NoError(t, repo.MarkAsRead(context.Background(), n1, userID))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, n1, userID)
	})

	countAfter, err := repo.UnreadCount(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.Equal(t, countAll-1, countAfter, "既読にした分だけ未読件数が減るべき")
}
