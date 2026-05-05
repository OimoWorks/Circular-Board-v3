package home_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/domain"
	"circular-board/internal/home"
)

func newTestService() *home.Service {
	return home.NewService(home.NewRepository(testPool))
}

// ─── 基本データ取得 ───────────────────────────────────────────────────────────

func TestGetHomeData_ReturnsAggregatedData(t *testing.T) {
	assoc := insertTestAssociation(t, "集約自治会", "AGG")
	user := insertTestUser(t, &assoc, "集約ユーザー", "agg@example.com", "pass", "user")

	_ = insertTestNotice(t, assoc, user, "最新お知らせ", false)
	_ = insertTestFile(t, assoc, user, 2024, 1, "最新資料.pdf")
	_ = insertTestSurvey(t, assoc, user, "最新アンケート", time.Now().Add(24*time.Hour))

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.NotNil(t, data.Notices)
	assert.NotNil(t, data.Files)
	assert.NotNil(t, data.Surveys)
}

func TestGetHomeData_MaxThreeItemsEach(t *testing.T) {
	assoc := insertTestAssociation(t, "3件制限自治会", "LIM3")
	user := insertTestUser(t, &assoc, "3件ユーザー", "lim3@example.com", "pass", "user")

	for i := 0; i < 5; i++ {
		insertTestNotice(t, assoc, user, "お知らせ", false)
		insertTestFile(t, assoc, user, 2024, i+1, "資料.pdf")
		insertTestSurvey(t, assoc, user, "アンケート", time.Now().Add(time.Duration(i+1)*24*time.Hour))
	}

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(data.Notices), 3, "お知らせは最大3件")
	assert.LessOrEqual(t, len(data.Files), 3, "回覧物は最大3件")
	assert.LessOrEqual(t, len(data.Surveys), 3, "アンケートは最大3件")
}

func TestGetHomeData_EmptyArraysWhenNoData(t *testing.T) {
	assoc := insertTestAssociation(t, "空自治会", "EMPTY")
	user := insertTestUser(t, &assoc, "空ユーザー", "empty@example.com", "pass", "user")

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)

	assert.NotNil(t, data.Notices)
	assert.NotNil(t, data.Files)
	assert.NotNil(t, data.Surveys)
	assert.Empty(t, data.Notices)
	assert.Empty(t, data.Files)
	assert.Empty(t, data.Surveys)
	assert.Equal(t, 0, data.UnreadNoticeCount)
	assert.Equal(t, 0, data.UnansweredSurveyCount)
}

// ─── カウント精度 ─────────────────────────────────────────────────────────────

func TestGetHomeData_UnreadCountAccuracy(t *testing.T) {
	assoc := insertTestAssociation(t, "未読カウント自治会", "UCNT")
	user := insertTestUser(t, &assoc, "未読ユーザー", "ucnt@example.com", "pass", "user")
	n1 := insertTestNotice(t, assoc, user, "未読お知らせ1", false)
	n2 := insertTestNotice(t, assoc, user, "未読お知らせ2", false)
	n3 := insertTestNotice(t, assoc, user, "既読お知らせ", false)
	_ = n1
	_ = n2
	markNoticeRead(t, n3, user)

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)
	assert.Equal(t, 2, data.UnreadNoticeCount)
}

func TestGetHomeData_UnansweredCountAccuracy(t *testing.T) {
	assoc := insertTestAssociation(t, "未回答カウント自治会", "ANCNT")
	user := insertTestUser(t, &assoc, "未回答ユーザー", "ancnt@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "未回答A", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "未回答B", time.Now().Add(48*time.Hour))
	answeredID := insertTestSurvey(t, assoc, user, "回答済み", time.Now().Add(24*time.Hour))
	markSurveyAnswered(t, answeredID, user)
	_ = insertTestSurvey(t, assoc, user, "期限切れ", time.Now().Add(-1*time.Hour))

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)
	assert.Equal(t, 2, data.UnansweredSurveyCount)
}

// ─── ロール別アクセス制御 ─────────────────────────────────────────────────────

func TestGetHomeData_MemberRole(t *testing.T) {
	assoc := insertTestAssociation(t, "メンバー自治会", "MBR")
	user := insertTestUser(t, &assoc, "メンバー", "mbr@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, user, "お知らせ", false)

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user)
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.NotEmpty(t, data.Notices)
}

func TestGetHomeData_AssociationAdminRole(t *testing.T) {
	assoc := insertTestAssociation(t, "管理者自治会", "ADM")
	admin := insertTestUser(t, &assoc, "管理者", "adm@example.com", "pass", "association_admin")
	_ = insertTestNotice(t, assoc, admin, "管理者お知らせ", false)

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleAssociationAdmin, &assoc, admin)
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.NotEmpty(t, data.Notices)
}

func TestGetHomeData_SystemAdminWithAssocID(t *testing.T) {
	assoc := insertTestAssociation(t, "システム管理自治会", "SYS")
	member := insertTestUser(t, &assoc, "メンバーSYS", "sys@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, member, "システム管理確認", false)

	sysAdmin := insertTestUser(t, nil, "システム管理者", "sysadmin@example.com", "pass", "system_admin")

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleSystemAdmin, &assoc, sysAdmin)
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.NotEmpty(t, data.Notices)
}

func TestGetHomeData_SystemAdminWithoutAssocID_ReturnsEmpty(t *testing.T) {
	sysAdmin := insertTestUser(t, nil, "SysAdminNoAssoc", "sysnoassoc@example.com", "pass", "system_admin")

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleSystemAdmin, nil, sysAdmin)
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Empty(t, data.Notices)
	assert.Empty(t, data.Files)
	assert.Empty(t, data.Surveys)
	assert.Equal(t, 0, data.UnreadNoticeCount)
	assert.Equal(t, 0, data.UnansweredSurveyCount)
}

// ─── クロステナント分離 ───────────────────────────────────────────────────────

func TestGetHomeData_CrossTenantIsolation(t *testing.T) {
	assoc1 := insertTestAssociation(t, "テナント分離1", "TEN1")
	assoc2 := insertTestAssociation(t, "テナント分離2", "TEN2")
	user1 := insertTestUser(t, &assoc1, "テナント1ユーザー", "ten1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "テナント2ユーザー", "ten2@example.com", "pass", "user")

	_ = insertTestNotice(t, assoc2, user2, "他自治会お知らせ", false)
	_ = insertTestFile(t, assoc2, user2, 2024, 1, "他自治会ファイル.pdf")
	_ = insertTestSurvey(t, assoc2, user2, "他自治会アンケート", time.Now().Add(24*time.Hour))

	svc := newTestService()
	data, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc1, user1)
	require.NoError(t, err)

	for _, n := range data.Notices {
		assert.NotEqual(t, "他自治会お知らせ", n.Title)
	}
	for _, f := range data.Files {
		assert.NotEqual(t, "他自治会ファイル.pdf", f.OriginalFilename)
	}
	for _, s := range data.Surveys {
		assert.NotEqual(t, "他自治会アンケート", s.Title)
	}
	assert.Equal(t, 0, data.UnreadNoticeCount)
	assert.Equal(t, 0, data.UnansweredSurveyCount)
}

// ─── ユーザーごとの既読・回答状態分離 ─────────────────────────────────────────

func TestGetHomeData_ReadStatusPerUser(t *testing.T) {
	assoc := insertTestAssociation(t, "既読分離自治会", "RDSEP")
	user1 := insertTestUser(t, &assoc, "ユーザーRD1", "rd1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc, "ユーザーRD2", "rd2@example.com", "pass", "user")
	noticeID := insertTestNotice(t, assoc, user1, "共有お知らせ", false)
	markNoticeRead(t, noticeID, user1)

	svc := newTestService()

	// user1 には既読
	data1, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user1)
	require.NoError(t, err)
	assert.Equal(t, 0, data1.UnreadNoticeCount)

	// user2 には未読
	data2, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user2)
	require.NoError(t, err)
	assert.Equal(t, 1, data2.UnreadNoticeCount)
}

func TestGetHomeData_AnswerStatusPerUser(t *testing.T) {
	assoc := insertTestAssociation(t, "回答分離自治会", "ANSEP")
	user1 := insertTestUser(t, &assoc, "ユーザーAN1", "an1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc, "ユーザーAN2", "an2@example.com", "pass", "user")
	surveyID := insertTestSurvey(t, assoc, user1, "共有アンケート", time.Now().Add(24*time.Hour))
	markSurveyAnswered(t, surveyID, user1)

	svc := newTestService()

	// user1 は回答済みなので未回答0
	data1, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user1)
	require.NoError(t, err)
	assert.Equal(t, 0, data1.UnansweredSurveyCount)

	// user2 は未回答なので未回答1
	data2, err := svc.GetHomeData(context.Background(), domain.RoleUser, &assoc, user2)
	require.NoError(t, err)
	assert.Equal(t, 1, data2.UnansweredSurveyCount)
}

