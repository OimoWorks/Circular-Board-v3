package home_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/home"
)

// ─── ListRecentNotices ───────────────────────────────────────────────────────

func TestListRecentNotices_FiltersByAssociation(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会A", "ASSOCA")
	assoc2 := insertTestAssociation(t, "自治会B", "ASSOCB")
	user1 := insertTestUser(t, &assoc1, "ユーザー1", "u1@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc1, user1, "自治会Aのお知らせ", false)
	_ = insertTestNotice(t, assoc2, user1, "自治会Bのお知らせ", false)

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc1, user1, 10)
	require.NoError(t, err)

	for _, n := range notices {
		assert.NotEqual(t, "自治会Bのお知らせ", n.Title)
	}
	titles := make([]string, 0, len(notices))
	for _, n := range notices {
		titles = append(titles, n.Title)
	}
	assert.Contains(t, titles, "自治会Aのお知らせ")
}

func TestListRecentNotices_LimitThree(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会Limit", "LIMITN")
	user := insertTestUser(t, &assoc, "ユーザーL", "ul@example.com", "pass", "user")
	for i := 0; i < 5; i++ {
		insertTestNotice(t, assoc, user, "お知らせ", false)
	}

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc, user, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(notices), 3)
}

func TestListRecentNotices_IsReadFlag(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会Read", "READN")
	user := insertTestUser(t, &assoc, "ユーザーR", "ur@example.com", "pass", "user")
	readNotice := insertTestNotice(t, assoc, user, "既読お知らせ", false)
	_ = insertTestNotice(t, assoc, user, "未読お知らせ", false)
	markNoticeRead(t, readNotice, user)

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	readCount := 0
	unreadCount := 0
	for _, n := range notices {
		if n.IsRead {
			readCount++
		} else {
			unreadCount++
		}
	}
	assert.Equal(t, 1, readCount, "既読は1件のはず")
	assert.Equal(t, 1, unreadCount, "未読は1件のはず")
}

func TestListRecentNotices_ExcludesDeleted(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会Del", "DELN")
	user := insertTestUser(t, &assoc, "ユーザーD", "ud@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, user, "有効なお知らせ", false)
	_ = insertTestNoticeDeleted(t, assoc, user, "削除済みお知らせ")

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	for _, n := range notices {
		assert.NotEqual(t, "削除済みお知らせ", n.Title)
	}
}

func TestListRecentNotices_PinnedFirst(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会Pin", "PINN")
	user := insertTestUser(t, &assoc, "ユーザーP", "up@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, user, "通常お知らせ", false)
	_ = insertTestNotice(t, assoc, user, "ピン留めお知らせ", true)

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(notices), 2)
	assert.True(t, notices[0].IsPinned, "ピン留めが先頭に来るはず")
}

func TestListRecentNotices_NoCrossTenantLeak(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会X1", "CTX1")
	assoc2 := insertTestAssociation(t, "自治会X2", "CTX2")
	user1 := insertTestUser(t, &assoc1, "ユーザーCT1", "ct1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "ユーザーCT2", "ct2@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc2, user2, "他自治会のお知らせ", false)

	repo := home.NewRepository(testPool)
	notices, err := repo.ListRecentNotices(context.Background(), assoc1, user1, 10)
	require.NoError(t, err)

	for _, n := range notices {
		assert.NotEqual(t, "他自治会のお知らせ", n.Title)
	}
}

// ─── ListRecentFiles ─────────────────────────────────────────────────────────

func TestListRecentFiles_FiltersByAssociation(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会FA1", "FA1")
	assoc2 := insertTestAssociation(t, "自治会FA2", "FA2")
	user := insertTestUser(t, &assoc1, "ユーザーFA", "ufa@example.com", "pass", "user")
	_ = insertTestFile(t, assoc1, user, 2024, 1, "自治会A資料.pdf")
	_ = insertTestFile(t, assoc2, user, 2024, 1, "自治会B資料.pdf")

	repo := home.NewRepository(testPool)
	files, err := repo.ListRecentFiles(context.Background(), assoc1, 10)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "自治会B資料.pdf", f.OriginalFilename)
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.OriginalFilename)
	}
	assert.Contains(t, names, "自治会A資料.pdf")
}

func TestListRecentFiles_LimitThree(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会FLimit", "FLIMIT")
	user := insertTestUser(t, &assoc, "ユーザーFL", "ufl@example.com", "pass", "user")
	for i := 1; i <= 5; i++ {
		insertTestFile(t, assoc, user, 2024, i, "資料.pdf")
	}

	repo := home.NewRepository(testPool)
	files, err := repo.ListRecentFiles(context.Background(), assoc, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(files), 3)
}

func TestListRecentFiles_ExcludesDeleted(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会FDel", "FDEL")
	user := insertTestUser(t, &assoc, "ユーザーFD", "ufd@example.com", "pass", "user")
	_ = insertTestFile(t, assoc, user, 2024, 1, "有効ファイル.pdf")
	_ = insertTestFileDeleted(t, assoc, user, 2024, 2)

	repo := home.NewRepository(testPool)
	files, err := repo.ListRecentFiles(context.Background(), assoc, 10)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "deleted.pdf", f.OriginalFilename)
	}
	assert.Equal(t, 1, len(files))
}

func TestListRecentFiles_SortedByYearMonthDesc(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会FSort", "FSORT")
	user := insertTestUser(t, &assoc, "ユーザーFS", "ufs@example.com", "pass", "user")
	_ = insertTestFile(t, assoc, user, 2022, 6, "古い資料.pdf")
	_ = insertTestFile(t, assoc, user, 2024, 3, "新しい資料.pdf")
	_ = insertTestFile(t, assoc, user, 2023, 12, "中間資料.pdf")

	repo := home.NewRepository(testPool)
	files, err := repo.ListRecentFiles(context.Background(), assoc, 10)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(files), 3)
	assert.Equal(t, 2024, files[0].Year)
	assert.Equal(t, 2023, files[1].Year)
	assert.Equal(t, 2022, files[2].Year)
}

func TestListRecentFiles_NoCrossTenantLeak(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会FCT1", "FCT1")
	assoc2 := insertTestAssociation(t, "自治会FCT2", "FCT2")
	_ = insertTestUser(t, &assoc1, "ユーザーFCT1", "fct1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "ユーザーFCT2", "fct2@example.com", "pass", "user")
	_ = insertTestFile(t, assoc2, user2, 2024, 1, "他自治会ファイル.pdf")

	repo := home.NewRepository(testPool)
	files, err := repo.ListRecentFiles(context.Background(), assoc1, 10)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "他自治会ファイル.pdf", f.OriginalFilename)
	}
}

// ─── ListUnansweredSurveys ───────────────────────────────────────────────────

func TestListUnansweredSurveys_FiltersByAssociation(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会SA1", "SA1")
	assoc2 := insertTestAssociation(t, "自治会SA2", "SA2")
	user1 := insertTestUser(t, &assoc1, "ユーザーSA1", "sa1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "ユーザーSA2", "sa2@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc1, user1, "自治会Aアンケート", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc2, user2, "自治会Bアンケート", time.Now().Add(24*time.Hour))

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc1, user1, 10)
	require.NoError(t, err)

	for _, s := range surveys {
		assert.NotEqual(t, "自治会Bアンケート", s.Title)
	}
	titles := make([]string, 0, len(surveys))
	for _, s := range surveys {
		titles = append(titles, s.Title)
	}
	assert.Contains(t, titles, "自治会Aアンケート")
}

func TestListUnansweredSurveys_LimitThree(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会SLimit", "SLIMIT")
	user := insertTestUser(t, &assoc, "ユーザーSL", "usl@example.com", "pass", "user")
	for i := 0; i < 5; i++ {
		insertTestSurvey(t, assoc, user, "アンケート", time.Now().Add(time.Duration(i+1)*24*time.Hour))
	}

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc, user, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(surveys), 3)
}

func TestListUnansweredSurveys_SortedByExpiresAtAsc(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会SSort", "SSORT")
	user := insertTestUser(t, &assoc, "ユーザーSS", "uss@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "期限遠いアンケート", time.Now().Add(72*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "期限近いアンケート", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "期限中間アンケート", time.Now().Add(48*time.Hour))

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(surveys), 3)
	assert.True(t, surveys[0].ExpiresAt.Before(surveys[1].ExpiresAt), "期限昇順になっているはず")
	assert.True(t, surveys[1].ExpiresAt.Before(surveys[2].ExpiresAt), "期限昇順になっているはず")
}

func TestListUnansweredSurveys_ExcludesExpired(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会SExp", "SEXP")
	user := insertTestUser(t, &assoc, "ユーザーSE", "use@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "有効アンケート", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "期限切れアンケート", time.Now().Add(-1*time.Hour))

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	for _, s := range surveys {
		assert.NotEqual(t, "期限切れアンケート", s.Title)
	}
	assert.Equal(t, 1, len(surveys))
}

func TestListUnansweredSurveys_ExcludesAnswered(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会SAns", "SANS")
	user := insertTestUser(t, &assoc, "ユーザーSAns", "uans@example.com", "pass", "user")
	unansweredID := insertTestSurvey(t, assoc, user, "未回答アンケート", time.Now().Add(24*time.Hour))
	answeredID := insertTestSurvey(t, assoc, user, "回答済みアンケート", time.Now().Add(24*time.Hour))
	markSurveyAnswered(t, answeredID, user)

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	ids := make([]uuid.UUID, 0, len(surveys))
	for _, s := range surveys {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, unansweredID)
	assert.NotContains(t, ids, answeredID)
}

func TestListUnansweredSurveys_ExcludesDeleted(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会SDel", "SDEL")
	user := insertTestUser(t, &assoc, "ユーザーSD", "usd@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "有効アンケート", time.Now().Add(24*time.Hour))
	_ = insertTestSurveyDeleted(t, assoc, user, "削除済みアンケート")

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc, user, 10)
	require.NoError(t, err)

	for _, s := range surveys {
		assert.NotEqual(t, "削除済みアンケート", s.Title)
	}
}

func TestListUnansweredSurveys_NoCrossTenantLeak(t *testing.T) {
	assoc1 := insertTestAssociation(t, "自治会SCT1", "SCT1")
	assoc2 := insertTestAssociation(t, "自治会SCT2", "SCT2")
	user1 := insertTestUser(t, &assoc1, "ユーザーSCT1", "sct1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "ユーザーSCT2", "sct2@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc2, user2, "他自治会アンケート", time.Now().Add(24*time.Hour))

	repo := home.NewRepository(testPool)
	surveys, err := repo.ListUnansweredSurveys(context.Background(), assoc1, user1, 10)
	require.NoError(t, err)

	for _, s := range surveys {
		assert.NotEqual(t, "他自治会アンケート", s.Title)
	}
}

// ─── CountUnreadNotices ──────────────────────────────────────────────────────

func TestCountUnreadNotices_Accuracy(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会CNR", "CNR")
	user := insertTestUser(t, &assoc, "ユーザーCNR", "cnr@example.com", "pass", "user")
	n1 := insertTestNotice(t, assoc, user, "未読1", false)
	n2 := insertTestNotice(t, assoc, user, "未読2", false)
	n3 := insertTestNotice(t, assoc, user, "既読1", false)
	_ = n1
	_ = n2
	markNoticeRead(t, n3, user)

	repo := home.NewRepository(testPool)
	count, err := repo.CountUnreadNotices(context.Background(), assoc, user)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCountUnreadNotices_ExcludesDeleted(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会CNRD", "CNRD")
	user := insertTestUser(t, &assoc, "ユーザーCNRD", "cnrd@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, user, "未読お知らせ", false)
	_ = insertTestNoticeDeleted(t, assoc, user, "削除済み未読")

	repo := home.NewRepository(testPool)
	count, err := repo.CountUnreadNotices(context.Background(), assoc, user)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ─── CountUnansweredSurveys ──────────────────────────────────────────────────

func TestCountUnansweredSurveys_Accuracy(t *testing.T) {
	assoc := insertTestAssociation(t, "自治会CUS", "CUS")
	user := insertTestUser(t, &assoc, "ユーザーCUS", "cus@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "未回答A", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "未回答B", time.Now().Add(48*time.Hour))
	answeredID := insertTestSurvey(t, assoc, user, "回答済み", time.Now().Add(24*time.Hour))
	markSurveyAnswered(t, answeredID, user)
	_ = insertTestSurvey(t, assoc, user, "期限切れ", time.Now().Add(-1*time.Hour))

	repo := home.NewRepository(testPool)
	count, err := repo.CountUnansweredSurveys(context.Background(), assoc, user)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
