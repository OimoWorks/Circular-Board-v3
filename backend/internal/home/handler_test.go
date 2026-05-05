package home_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── 認証チェック ─────────────────────────────────────────────────────────────

func TestGetHomeData_Unauthenticated_Returns401(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetHomeData_InvalidToken_Returns401(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", "invalid.token.here")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ─── 正常レスポンス構造 ───────────────────────────────────────────────────────

func TestGetHomeData_Returns200WithExpectedFields(t *testing.T) {
	assoc := insertTestAssociation(t, "フィールド確認自治会", "FLDCHK")
	user := insertTestUser(t, &assoc, "フィールドユーザー", "fld@example.com", "pass", "user")
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data フィールドが存在するはず")

	assert.Contains(t, data, "notices")
	assert.Contains(t, data, "files")
	assert.Contains(t, data, "surveys")
	assert.Contains(t, data, "unread_notice_count")
	assert.Contains(t, data, "unanswered_survey_count")
}

func TestGetHomeData_Handler_EmptyArraysWhenNoData(t *testing.T) {
	assoc := insertTestAssociation(t, "空ハンドラ自治会", "HEMPTY")
	user := insertTestUser(t, &assoc, "空ハンドラユーザー", "hempty@example.com", "pass", "user")
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})

	notices := data["notices"].([]interface{})
	files := data["files"].([]interface{})
	surveys := data["surveys"].([]interface{})
	assert.Empty(t, notices)
	assert.Empty(t, files)
	assert.Empty(t, surveys)
	assert.Equal(t, float64(0), data["unread_notice_count"])
	assert.Equal(t, float64(0), data["unanswered_survey_count"])
}

func TestGetHomeData_NoticesContainExpectedFields(t *testing.T) {
	assoc := insertTestAssociation(t, "お知らせフィールド自治会", "NFLD")
	user := insertTestUser(t, &assoc, "お知らせユーザー", "nfld@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, user, "フィールド確認お知らせ", true)
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	require.NotEmpty(t, notices)

	n := notices[0].(map[string]interface{})
	assert.Contains(t, n, "id")
	assert.Contains(t, n, "title")
	assert.Contains(t, n, "is_pinned")
	assert.Contains(t, n, "is_read")
	assert.Contains(t, n, "created_at")
	assert.Equal(t, "フィールド確認お知らせ", n["title"])
	assert.Equal(t, true, n["is_pinned"])
}

func TestGetHomeData_FilesContainExpectedFields(t *testing.T) {
	assoc := insertTestAssociation(t, "ファイルフィールド自治会", "FFLD")
	user := insertTestUser(t, &assoc, "ファイルユーザー", "ffld@example.com", "pass", "user")
	_ = insertTestFile(t, assoc, user, 2024, 3, "フィールド確認.pdf")
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	files := data["files"].([]interface{})
	require.NotEmpty(t, files)

	f := files[0].(map[string]interface{})
	assert.Contains(t, f, "id")
	assert.Contains(t, f, "year")
	assert.Contains(t, f, "month")
	assert.Contains(t, f, "original_filename")
	assert.Contains(t, f, "file_size")
	assert.Contains(t, f, "mime_type")
	assert.Contains(t, f, "created_at")
	assert.Equal(t, "フィールド確認.pdf", f["original_filename"])
	assert.Equal(t, float64(2024), f["year"])
}

func TestGetHomeData_SurveysContainExpectedFields(t *testing.T) {
	assoc := insertTestAssociation(t, "アンケートフィールド自治会", "SFLD")
	user := insertTestUser(t, &assoc, "アンケートユーザー", "sfld@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "フィールド確認アンケート", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	surveys := data["surveys"].([]interface{})
	require.NotEmpty(t, surveys)

	s := surveys[0].(map[string]interface{})
	assert.Contains(t, s, "id")
	assert.Contains(t, s, "title")
	assert.Contains(t, s, "expires_at")
	assert.Equal(t, "フィールド確認アンケート", s["title"])
}

// ─── 件数上限 ─────────────────────────────────────────────────────────────────

func TestGetHomeData_MaxThreeItemsInResponse(t *testing.T) {
	assoc := insertTestAssociation(t, "ハンドラ3件自治会", "H3LIM")
	user := insertTestUser(t, &assoc, "ハンドラ3件ユーザー", "h3lim@example.com", "pass", "user")
	for i := 0; i < 5; i++ {
		insertTestNotice(t, assoc, user, "お知らせ", false)
		insertTestFile(t, assoc, user, 2024, i+1, "資料.pdf")
		insertTestSurvey(t, assoc, user, "アンケート", time.Now().Add(time.Duration(i+1)*24*time.Hour))
	}
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	files := data["files"].([]interface{})
	surveys := data["surveys"].([]interface{})
	assert.LessOrEqual(t, len(notices), 3)
	assert.LessOrEqual(t, len(files), 3)
	assert.LessOrEqual(t, len(surveys), 3)
}

// ─── クロステナント分離 ───────────────────────────────────────────────────────

func TestGetHomeData_Handler_CrossTenantIsolation(t *testing.T) {
	assoc1 := insertTestAssociation(t, "ハンドラCT1", "HCT1")
	assoc2 := insertTestAssociation(t, "ハンドラCT2", "HCT2")
	user1 := insertTestUser(t, &assoc1, "ハンドラCTユーザー1", "hct1@example.com", "pass", "user")
	user2 := insertTestUser(t, &assoc2, "ハンドラCTユーザー2", "hct2@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc2, user2, "他自治会お知らせ", false)
	token1 := makeTestToken(t, assoc1.String(), user1.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token1)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	for _, item := range notices {
		n := item.(map[string]interface{})
		assert.NotEqual(t, "他自治会お知らせ", n["title"])
	}
}

// ─── ロール別レスポンス ───────────────────────────────────────────────────────

func TestGetHomeData_AssociationAdmin_Returns200(t *testing.T) {
	assoc := insertTestAssociation(t, "管理者ハンドラ自治会", "HADM")
	admin := insertTestUser(t, &assoc, "管理者ハンドラ", "hadm@example.com", "pass", "association_admin")
	_ = insertTestNotice(t, assoc, admin, "管理者ハンドラお知らせ", false)
	token := makeTestToken(t, assoc.String(), admin.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "notices")
}

func TestGetHomeData_SystemAdmin_NoAssocParam_ReturnsEmpty(t *testing.T) {
	sysAdmin := insertTestUser(t, nil, "SysAdminHandler", "syshandler@example.com", "pass", "system_admin")
	token := makeTestToken(t, "", sysAdmin.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	files := data["files"].([]interface{})
	surveys := data["surveys"].([]interface{})
	assert.Empty(t, notices)
	assert.Empty(t, files)
	assert.Empty(t, surveys)
}

func TestGetHomeData_SystemAdmin_WithAssocParam_Returns200(t *testing.T) {
	assoc := insertTestAssociation(t, "システム管理ハンドラ自治会", "HSYS")
	member := insertTestUser(t, &assoc, "メンバーHSYS", "hsys@example.com", "pass", "user")
	_ = insertTestNotice(t, assoc, member, "システム管理ハンドラお知らせ", false)
	sysAdmin := insertTestUser(t, nil, "SysAdminWithParam", "sysparam@example.com", "pass", "system_admin")
	token := makeTestToken(t, "", sysAdmin.String(), "system_admin")

	deps := newTestDeps()
	path := "/api/v1/home?association_id=" + assoc.String()
	rr := doRequest(t, deps.router, http.MethodGet, path, token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	notices := data["notices"].([]interface{})
	require.NotEmpty(t, notices)
	n := notices[0].(map[string]interface{})
	assert.Equal(t, "システム管理ハンドラお知らせ", n["title"])
}

func TestGetHomeData_SystemAdmin_InvalidAssocParam_Returns400(t *testing.T) {
	sysAdmin := insertTestUser(t, nil, "SysAdminBadParam", "sysbad@example.com", "pass", "system_admin")
	token := makeTestToken(t, "", sysAdmin.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home?association_id=not-a-uuid", token)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─── カウントの精度 ───────────────────────────────────────────────────────────

func TestGetHomeData_UnreadCountInResponse(t *testing.T) {
	assoc := insertTestAssociation(t, "未読カウントハンドラ", "HUCNT")
	user := insertTestUser(t, &assoc, "未読ハンドラユーザー", "hucnt@example.com", "pass", "user")
	n1 := insertTestNotice(t, assoc, user, "未読1H", false)
	_ = insertTestNotice(t, assoc, user, "未読2H", false)
	_ = insertTestNotice(t, assoc, user, "未読3H", false)
	markNoticeRead(t, n1, user)
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["unread_notice_count"])
}

func TestGetHomeData_UnansweredSurveyCountInResponse(t *testing.T) {
	assoc := insertTestAssociation(t, "未回答カウントハンドラ", "HANCNT")
	user := insertTestUser(t, &assoc, "未回答ハンドラユーザー", "hancnt@example.com", "pass", "user")
	_ = insertTestSurvey(t, assoc, user, "未回答H1", time.Now().Add(24*time.Hour))
	_ = insertTestSurvey(t, assoc, user, "未回答H2", time.Now().Add(48*time.Hour))
	answeredID := insertTestSurvey(t, assoc, user, "回答済みH", time.Now().Add(24*time.Hour))
	markSurveyAnswered(t, answeredID, user)
	token := makeTestToken(t, assoc.String(), user.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/home", token)
	require.Equal(t, http.StatusOK, rr.Code)

	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["unanswered_survey_count"])
}
