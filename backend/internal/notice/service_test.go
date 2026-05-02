package notice_test

// サービス層のテスト。
// ロールチェック（association_adminのみ登録・削除可 / 一般ユーザーは不可）はミドルウェアで
// 強制されるため、ハンドラーテスト（handler_test.go）でカバーしている。
// サービス層ではバリデーション（タイトル・本文必須）とテナント境界をテストする。

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/notice"
)

// ─── テスト用サービス初期化 ──────────────────────────────────
func newNoticeService() *notice.Service {
	return notice.NewService(notice.NewRepository(testPool))
}

// ═══════════════════════════════════════════════════════════
// Service.Create（お知らせ登録）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_Create_ByAdmin_Success
// association_adminがお知らせを登録できる
func TestNoticeService_Create_ByAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者登録自治会", "SVC_NOT_ADM")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-adm.test", "pass123", "association_admin")

	svc := newNoticeService()
	n, err := svc.Create(context.Background(), assocID, adminID, notice.CreateInput{
		Title:    "重要告知",
		Body:     "大切なお知らせです。",
		IsPinned: true,
	})
	require.NoError(t, err)
	require.NotNil(t, n)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notices WHERE id = $1`, n.ID)
	})

	assert.Equal(t, assocID, n.AssociationID)
	assert.Equal(t, adminID, n.CreatedBy)
	assert.Equal(t, "重要告知", n.Title)
	assert.Equal(t, "大切なお知らせです。", n.Body)
	assert.True(t, n.IsPinned)
	assert.NotZero(t, n.ID)
	assert.NotZero(t, n.CreatedAt)
}

// TestNoticeService_Create_EmptyTitle
// タイトルが空はエラー
func TestNoticeService_Create_EmptyTitle(t *testing.T) {
	assocID := insertTestAssociation(t, "空タイトル自治会", "SVC_NOT_ETITLE")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-etitle.test", "pass123", "association_admin")

	svc := newNoticeService()
	_, err := svc.Create(context.Background(), assocID, adminID, notice.CreateInput{
		Title: "",
		Body:  "本文あり",
	})
	assert.Error(t, err, "タイトルが空のときはエラーが返るべき")
}

// TestNoticeService_Create_WhitespaceTitle
// スペースのみのタイトルもエラー（TrimSpace済み）
func TestNoticeService_Create_WhitespaceTitle(t *testing.T) {
	assocID := insertTestAssociation(t, "空白タイトル自治会", "SVC_NOT_SPTITLE")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-sptitle.test", "pass123", "association_admin")

	svc := newNoticeService()
	_, err := svc.Create(context.Background(), assocID, adminID, notice.CreateInput{
		Title: "   ",
		Body:  "本文あり",
	})
	assert.Error(t, err, "スペースのみのタイトルはエラーが返るべき")
}

// TestNoticeService_Create_EmptyBody
// 本文が空はエラー
func TestNoticeService_Create_EmptyBody(t *testing.T) {
	assocID := insertTestAssociation(t, "空本文自治会", "SVC_NOT_EBODY")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-ebody.test", "pass123", "association_admin")

	svc := newNoticeService()
	_, err := svc.Create(context.Background(), assocID, adminID, notice.CreateInput{
		Title: "タイトルあり",
		Body:  "",
	})
	assert.Error(t, err, "本文が空のときはエラーが返るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.Delete（お知らせ削除）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_Delete_ByAdmin_Success
// association_adminが削除できる（サービス層はロールを見ない）
func TestNoticeService_Delete_ByAdmin_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者削除自治会", "SVC_NOT_DEL1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-del1.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocID, adminID, "削除対象", "削除本文", false)

	svc := newNoticeService()
	err := svc.Delete(context.Background(), assocID, noticeID)
	require.NoError(t, err)

	// 削除後はGetで取得できない
	_, errGet := svc.Get(context.Background(), assocID, noticeID)
	assert.True(t, errors.Is(errGet, notice.ErrNotFound), "削除後はGetで取得できない")
}

// TestNoticeService_Delete_NotFound
// 存在しないIDはErrNotFound
func TestNoticeService_Delete_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "削除不在自治会", "SVC_NOT_DELNF")

	svc := newNoticeService()
	err := svc.Delete(context.Background(), assocID, uuid.New())
	assert.True(t, errors.Is(err, notice.ErrNotFound))
}

// TestNoticeService_Delete_CrossTenant
// 他自治会のお知らせは削除不可（テナント境界）
func TestNoticeService_Delete_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "SVC_NOT_DTA")
	assocB := insertTestAssociation(t, "削除テナントB", "SVC_NOT_DTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@svc-not-dta.test", "pass123", "association_admin")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)

	svc := newNoticeService()
	// 自治会Bとして削除しようとする
	err := svc.Delete(context.Background(), assocB, noticeID)
	assert.True(t, errors.Is(err, notice.ErrNotFound), "他の自治会IDでは削除できない")

	// 自治会Aのお知らせは残っている
	got, err := svc.Get(context.Background(), assocA, noticeID)
	require.NoError(t, err)
	assert.Equal(t, noticeID, got.ID)
}

// ═══════════════════════════════════════════════════════════
// Service.Get（お知らせ閲覧）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_Get_AnyRole_Success
// サービス層はロールを見ないため全ロールが閲覧できる
func TestNoticeService_Get_AnyRole_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "全ロール閲覧自治会", "SVC_NOT_GET1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-not-get1.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@svc-not-get1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, adminID, "閲覧テスト", "本文", false)

	svc := newNoticeService()

	// 管理者が取得
	gotByAdmin, err := svc.Get(context.Background(), assocID, noticeID)
	require.NoError(t, err)
	assert.Equal(t, noticeID, gotByAdmin.ID)

	// 一般ユーザーが取得（サービス層はロールを見ない）
	gotByUser, err := svc.Get(context.Background(), assocID, noticeID)
	require.NoError(t, err)
	assert.Equal(t, noticeID, gotByUser.ID)
	_ = userID
}

// TestNoticeService_Get_NotFound
// 存在しないIDはErrNotFound
func TestNoticeService_Get_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "取得不在自治会", "SVC_NOT_GETNF")

	svc := newNoticeService()
	_, err := svc.Get(context.Background(), assocID, uuid.New())
	assert.True(t, errors.Is(err, notice.ErrNotFound))
}

// TestNoticeService_Get_CrossTenant
// 他自治会のお知らせは閲覧不可
func TestNoticeService_Get_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "閲覧テナントA", "SVC_NOT_GTA")
	assocB := insertTestAssociation(t, "閲覧テナントB", "SVC_NOT_GTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@svc-not-gta.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)

	svc := newNoticeService()
	_, err := svc.Get(context.Background(), assocB, noticeID)
	assert.True(t, errors.Is(err, notice.ErrNotFound),
		"他の自治会IDではお知らせを閲覧できてはいけない")
}

// ═══════════════════════════════════════════════════════════
// Service.MarkAsRead（既読）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_MarkAsRead_Success
// 全ロールが既読にできる（サービス層はロールを見ない）
func TestNoticeService_MarkAsRead_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "既読サービス自治会", "SVC_NOT_MAR1")
	userID := insertTestUser(t, &assocID, "既読ユーザー", "read@svc-not-mar1.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocID, userID, "既読テスト", "本文", false)

	svc := newNoticeService()
	err := svc.MarkAsRead(context.Background(), assocID, noticeID, userID)
	require.NoError(t, err)

	// 一覧のis_readがtrueになっているか確認
	items, err := svc.List(context.Background(), assocID, userID)
	require.NoError(t, err)
	for _, n := range items {
		if n.ID == noticeID {
			assert.True(t, n.IsRead, "既読にしたお知らせのis_readはtrueであるべき")
		}
	}

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, noticeID, userID)
	})
}

// TestNoticeService_MarkAsRead_CrossTenant
// 他自治会のお知らせは既読にできない（テナントチェック込み）
func TestNoticeService_MarkAsRead_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "既読テナントA", "SVC_NOT_MTA")
	assocB := insertTestAssociation(t, "既読テナントB", "SVC_NOT_MTB")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@svc-not-mta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "Bユーザー", "b@svc-not-mtb.test", "pass123", "user")
	noticeID := insertTestNotice(t, assocA, userA, "Aのお知らせ", "A本文", false)

	svc := newNoticeService()
	// 自治会Bのユーザーが自治会Aのお知らせを既読にしようとする
	err := svc.MarkAsRead(context.Background(), assocB, noticeID, userB)
	assert.True(t, errors.Is(err, notice.ErrNotFound),
		"他の自治会のお知らせは既読にできない")
}

// ═══════════════════════════════════════════════════════════
// Service.UnreadCount（未読件数）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_UnreadCount_Correct
// 未読件数が正しく返る
func TestNoticeService_UnreadCount_Correct(t *testing.T) {
	assocID := insertTestAssociation(t, "未読件数サービス自治会", "SVC_NOT_UNRD1")
	userID := insertTestUser(t, &assocID, "未読ユーザー", "unrd@svc-not-unrd1.test", "pass123", "user")

	n1 := insertTestNotice(t, assocID, userID, "未読1", "本文1", false)
	_ = insertTestNotice(t, assocID, userID, "未読2", "本文2", false)

	svc := newNoticeService()

	countBefore, err := svc.UnreadCount(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countBefore, 2, "2件以上未読であるべき")

	// n1を既読にする
	require.NoError(t, svc.MarkAsRead(context.Background(), assocID, n1, userID))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, n1, userID)
	})

	countAfter, err := svc.UnreadCount(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.Equal(t, countBefore-1, countAfter, "既読後は未読件数が1減るべき")
}

// ═══════════════════════════════════════════════════════════
// Service.List（お知らせ一覧）
// ═══════════════════════════════════════════════════════════

// TestNoticeService_List_WithIsReadFlag
// is_readフラグが正しく返る（既読/未読の区別）
func TestNoticeService_List_WithIsReadFlag(t *testing.T) {
	assocID := insertTestAssociation(t, "is_read一覧自治会", "SVC_NOT_LREAD")
	userID := insertTestUser(t, &assocID, "既読確認ユーザー", "read@svc-not-lread.test", "pass123", "user")

	readID := insertTestNotice(t, assocID, userID, "既読済み", "本文", false)
	unreadID := insertTestNotice(t, assocID, userID, "未読", "本文", false)

	svc := newNoticeService()

	// readIDを既読にする
	require.NoError(t, svc.MarkAsRead(context.Background(), assocID, readID, userID))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM notice_reads WHERE notice_id = $1 AND user_id = $2`, readID, userID)
	})

	items, err := svc.List(context.Background(), assocID, userID)
	require.NoError(t, err)

	readMap := make(map[uuid.UUID]bool)
	for _, n := range items {
		readMap[n.ID] = n.IsRead
	}
	assert.True(t, readMap[readID], "既読にしたお知らせのis_readはtrueであるべき")
	assert.False(t, readMap[unreadID], "未読のお知らせのis_readはfalseであるべき")
}

// TestNoticeService_List_PinnedFirst
// ピン留めが上部に来る
func TestNoticeService_List_PinnedFirst(t *testing.T) {
	assocID := insertTestAssociation(t, "ピン順サービス自治会", "SVC_NOT_LPIN")
	userID := insertTestUser(t, &assocID, "ピン順ユーザー", "pin@svc-not-lpin.test", "pass123", "user")

	_ = insertTestNotice(t, assocID, userID, "通常", "本文", false)
	pinnedID := insertTestNotice(t, assocID, userID, "重要！", "重要本文", true)

	svc := newNoticeService()
	items, err := svc.List(context.Background(), assocID, userID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(items), 2)

	assert.Equal(t, pinnedID, items[0].ID, "ピン留めされたお知らせが先頭に来るべき")
}
