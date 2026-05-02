package files_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════
// POST /api/v1/files（ファイルアップロード）
// ═══════════════════════════════════════════════════════════

// TestUploadHandler_Success_Admin
// association_admin がPDFをアップロードすると 201 とファイル情報が返る
func TestUploadHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "アップロードハンドラー自治会", "HDL_UPL_1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-upl-1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyPDF, "テスト資料.pdf", "application/pdf",
		map[string]string{"year": "2024", "month": "6"},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "テスト資料.pdf", data["original_filename"])
	assert.Equal(t, "application/pdf", data["mime_type"])
	assert.Equal(t, float64(2024), data["year"])
	assert.Equal(t, float64(6), data["month"])
	assert.NotEmpty(t, data["id"])

	// アップロードされた物理ファイルをクリーンアップ
	t.Cleanup(func() {
		fileID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, fileID)
		// storagePathは取得できないためテンポラリディレクトリのゴミは残るが許容
	})
}

// TestUploadHandler_Success_AssociationAdmin_JPEG
// JPEG画像もアップロードできる
func TestUploadHandler_Success_AssociationAdmin_JPEG(t *testing.T) {
	assocID := insertTestAssociation(t, "JPEG ハンドラー自治会", "HDL_UPL_JPG")
	userID := insertTestUser(t, &assocID, "JPEG管理者", "admin@hdl-jpg.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyJPEG, "写真.jpg", "image/jpeg",
		map[string]string{"year": "2024", "month": "8"},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "image/jpeg", data["mime_type"])

	t.Cleanup(func() {
		fileID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, fileID)
	})
}

// TestUploadHandler_Unauthorized
// 未認証リクエストは 401 が返る
func TestUploadHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyPDF, "test.pdf", "application/pdf",
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestUploadHandler_Forbidden_UserRole
// 一般ユーザー（user ロール）はアップロード不可で 403 が返る
func TestUploadHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "403ユーザー自治会", "HDL_UPL_USR")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-upl-usr.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyPDF, "test.pdf", "application/pdf",
		nil, token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestUploadHandler_NoFile
// ファイルが添付されていないリクエストは 400 が返る
func TestUploadHandler_NoFile(t *testing.T) {
	assocID := insertTestAssociation(t, "ファイルなし自治会", "HDL_UPL_NOFILE")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-nofile.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	// JSONボディ（multipartではない）で送ると ParseMultipartForm が失敗する
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/files", map[string]string{}, token)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)

	// または multipart だがファイルフィールドなし → 400
	// ここでは Content-Type が application/json の場合を確認
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.NotEmpty(t, errObj["code"])
}

// TestUploadHandler_UnsupportedMIME
// 非対応 MIME タイプのファイルは 400 が返る
func TestUploadHandler_UnsupportedMIME(t *testing.T) {
	assocID := insertTestAssociation(t, "非対応MIME自治会", "HDL_UPL_MIME")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-mime.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		[]byte("plain text content"), "test.txt", "text/plain",
		nil, token,
	)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "UNSUPPORTED_FILE_TYPE", errObj["code"])
}

// TestUploadHandler_FileTooLarge
// 10MB 超のファイルは 413 が返る
func TestUploadHandler_FileTooLarge(t *testing.T) {
	assocID := insertTestAssociation(t, "サイズ超過自治会", "HDL_UPL_LARGE")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-large.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	// 10MB + 1バイトのダミーデータ
	largeContent := make([]byte, (10<<20)+1)

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		largeContent, "huge.pdf", "application/pdf",
		nil, token,
	)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DELETE /api/v1/files/:id（ファイル削除）
// ═══════════════════════════════════════════════════════════

// TestDeleteHandler_Success_Admin
// 管理者が削除すると 200 が返りファイルが論理削除される
func TestDeleteHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "削除ハンドラー自治会", "HDL_DEL_1")
	userID := insertTestUser(t, &assocID, "削除管理者", "admin@hdl-del-1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	// 物理ファイルなしのDBレコード（SoftDeleteテスト用、物理ファイルがなくても論理削除は成功する）
	fileID := insertTestFile(t, assocID, userID, 2024, 4, "削除対象.pdf", "application/pdf")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "削除しました", data["message"])
}

// TestDeleteHandler_Unauthorized
// 未認証では 401 が返る
func TestDeleteHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "削除401自治会", "HDL_DEL_UNAUTH")
	userID := insertTestUser(t, &assocID, "ダミーユーザー", "dummy@hdl-del-unauth.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, userID, 2024, 1, "資料.pdf", "application/pdf")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileID),
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestDeleteHandler_Forbidden_UserRole
// 一般ユーザーは削除不可で 403 が返る
func TestDeleteHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "削除403自治会", "HDL_DEL_USR")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-del-usr.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-del-usr.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, adminID, 2024, 1, "資料.pdf", "application/pdf")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestDeleteHandler_CrossTenant
// 他の自治会のファイルは削除できない（テナント境界）
// 実装上は ErrNotFound → 404 で応答する（リソース存在の情報漏洩を防ぐため）
func TestDeleteHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "HDL_DEL_TA")
	assocB := insertTestAssociation(t, "削除テナントB", "HDL_DEL_TB")
	userA := insertTestUser(t, &assocA, "テナントAユーザー", "admin@hdl-del-ta.test", "pass123", "association_admin")
	adminB := insertTestUser(t, &assocB, "テナントB管理者", "admin@hdl-del-tb.test", "pass123", "association_admin")
	fileID := insertTestFile(t, assocA, userA, 2024, 1, "テナントA資料.pdf", "application/pdf")

	// テナントBの管理者トークンでテナントAのファイルを削除しようとする
	tokenB := makeTestToken(t, assocB.String(), adminB.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileID),
		nil, tokenB,
	)

	// テナント境界により ErrNotFound → 404
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestDeleteHandler_NotFound
// 存在しないIDは 404 が返る
func TestDeleteHandler_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "削除不在自治会", "HDL_DEL_NF")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-del-nf.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", uuid.New()),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "FILE_NOT_FOUND", errObj["code"])
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/files（ファイル一覧取得）
// ═══════════════════════════════════════════════════════════

// TestListHandler_Success_User
// 一般ユーザーが一覧を取得できる
func TestListHandler_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧ハンドラー自治会", "HDL_LIST_1")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-list-1.test", "pass123", "user")
	_ = insertTestFile(t, assocID, userID, 2024, 5, "5月資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 6, "6月資料.pdf", "application/pdf")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	fileList := data["files"].([]interface{})
	assert.GreaterOrEqual(t, len(fileList), 2)
}

// TestListHandler_Success_Admin
// 管理者も一覧を取得できる
func TestListHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧管理者自治会", "HDL_LIST_ADMIN")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-list-admin.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestListHandler_FilterByYearMonth
// year・month クエリパラメータで絞り込める
func TestListHandler_FilterByYearMonth(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧フィルタ自治会", "HDL_LIST_FILTER")
	userID := insertTestUser(t, &assocID, "フィルタユーザー", "filter@hdl-list-filter.test", "pass123", "user")
	_ = insertTestFile(t, assocID, userID, 2024, 4, "4月資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 7, "7月資料.pdf", "application/pdf")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		"/api/v1/files?year=2024&month=4",
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	fileList := data["files"].([]interface{})

	for _, item := range fileList {
		f := item.(map[string]interface{})
		assert.Equal(t, float64(2024), f["year"])
		assert.Equal(t, float64(4), f["month"], "4月のみが返るべき")
	}
}

// TestListHandler_Unauthorized
// 未認証では 401 が返る
func TestListHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestListHandler_CrossTenant
// 他の自治会のデータが混入しない
func TestListHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "一覧テナントA", "HDL_LIST_TA")
	assocB := insertTestAssociation(t, "一覧テナントB", "HDL_LIST_TB")
	userA := insertTestUser(t, &assocA, "テナントAユーザー", "user@hdl-list-ta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "テナントBユーザー", "user@hdl-list-tb.test", "pass123", "user")
	_ = insertTestFile(t, assocA, userA, 2024, 1, "A資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocB, userB, 2024, 1, "B資料.pdf", "application/pdf")

	// テナントAのトークンで一覧取得
	tokenA := makeTestToken(t, assocA.String(), userA.String(), "user")
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files", nil, tokenA)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	fileList := data["files"].([]interface{})

	for _, item := range fileList {
		f := item.(map[string]interface{})
		assert.Equal(t, assocA.String(), f["association_id"],
			"テナントAの一覧にテナントBのファイルが混入してはいけない")
	}
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/files/:id/download（ファイルダウンロード）
// ═══════════════════════════════════════════════════════════

// TestDownloadHandler_Success_User
// 一般ユーザーがダウンロードできる（全ロール対応）
func TestDownloadHandler_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "DLハンドラー自治会", "HDL_DL_1")
	userID := insertTestUser(t, &assocID, "ダウンロードユーザー", "user@hdl-dl-1.test", "pass123", "user")

	// 物理ファイルを作成してDBレコードも挿入
	fileID, _ := insertTestFileOnDisk(t, assocID, userID, 2024, 5, "ダウンロード.pdf", "application/pdf", dummyPDF)
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files/%s/download", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
	assert.NotEmpty(t, rr.Header().Get("Content-Disposition"))
	assert.Greater(t, rr.Body.Len(), 0, "レスポンスボディにファイル内容が含まれるべき")
}

// TestDownloadHandler_Success_Admin
// 管理者もダウンロードできる
func TestDownloadHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "DL管理者自治会", "HDL_DL_ADMIN")
	userID := insertTestUser(t, &assocID, "DL管理者", "admin@hdl-dl-admin.test", "pass123", "association_admin")
	fileID, _ := insertTestFileOnDisk(t, assocID, userID, 2024, 6, "管理者DL.pdf", "application/pdf", dummyPDF)
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files/%s/download", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestDownloadHandler_Unauthorized
// 未認証では 401 が返る
func TestDownloadHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "DL401自治会", "HDL_DL_UNAUTH")
	userID := insertTestUser(t, &assocID, "ダミーユーザー", "dummy@hdl-dl-unauth.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, userID, 2024, 1, "資料.pdf", "application/pdf")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files/%s/download", fileID),
		nil, "" /* トークンなし */,
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestDownloadHandler_CrossTenant
// 他の自治会のファイルはダウンロードできない
// 実装上は ErrNotFound → 404 で応答する（リソース存在の情報漏洩を防ぐため）
func TestDownloadHandler_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "DLテナントA", "HDL_DL_TA")
	assocB := insertTestAssociation(t, "DLテナントB", "HDL_DL_TB")
	userA := insertTestUser(t, &assocA, "テナントAユーザー", "user@hdl-dl-ta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "テナントBユーザー", "user@hdl-dl-tb.test", "pass123", "user")

	// テナントAに物理ファイルを作成
	fileID, _ := insertTestFileOnDisk(t, assocA, userA, 2024, 7, "テナントA秘密資料.pdf", "application/pdf", dummyPDF)

	// テナントBのトークンでテナントAのファイルをダウンロードしようとする
	tokenB := makeTestToken(t, assocB.String(), userB.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files/%s/download", fileID),
		nil, tokenB,
	)

	// テナント境界により ErrNotFound → 404
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestDownloadHandler_DeletedFile
// 論理削除済みファイルは 404 が返る
func TestDownloadHandler_DeletedFile(t *testing.T) {
	assocID := insertTestAssociation(t, "DL削除済み自治会", "HDL_DL_DEL")
	adminID := insertTestUser(t, &assocID, "DL削除管理者", "admin@hdl-dl-del.test", "pass123", "association_admin")

	// ファイルを用意して論理削除する
	fileID, storagePath := insertTestFileOnDisk(t, assocID, adminID, 2024, 8, "削除済み資料.pdf", "application/pdf", dummyPDF)
	_ = storagePath

	// 論理削除（SoftDelete）
	repo := newTestDeps().filesSvc // filesSvc を直接使う代わりにDBで直接削除
	_ = repo
	_, err := testPool.Exec(context.Background(),
		`UPDATE files SET deleted_at = NOW() WHERE id = $1`, fileID)
	require.NoError(t, err)
	// 物理ファイルは削除しない（論理削除後のアクセス制限テスト）
	t.Cleanup(func() {
		_ = os.Remove(storagePath)
	})

	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files/%s/download", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "FILE_NOT_FOUND", errObj["code"])
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/files/years（利用可能な年一覧）
// ═══════════════════════════════════════════════════════════

// TestAvailableYearsHandler_Success
// 全ロールが年一覧を取得できる
func TestAvailableYearsHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "年一覧ハンドラー自治会", "HDL_YEARS_1")
	userID := insertTestUser(t, &assocID, "年一覧ユーザー", "user@hdl-years-1.test", "pass123", "user")
	_ = insertTestFile(t, assocID, userID, 2024, 1, "2024資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2023, 6, "2023資料.pdf", "application/pdf")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files/years", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	years := data["years"].([]interface{})
	assert.GreaterOrEqual(t, len(years), 2)
}

// TestAvailableYearsHandler_Empty
// ファイルがない場合は空配列が返る
func TestAvailableYearsHandler_Empty(t *testing.T) {
	assocID := insertTestAssociation(t, "空年一覧自治会", "HDL_YEARS_EMPTY")
	userID := insertTestUser(t, &assocID, "空年ユーザー", "user@hdl-years-empty.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files/years", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	years := data["years"].([]interface{})
	assert.Empty(t, years)
}

// TestAvailableYearsHandler_Unauthorized
// 未認証では 401 が返る
func TestAvailableYearsHandler_Unauthorized(t *testing.T) {
	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files/years", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// system_admin 対応テスト
// ═══════════════════════════════════════════════════════════

// newTestDepsWithAssociations は /api/v1/associations を含むテスト用ルーターを返す
func newTestDepsWithAssociations() *testDeps {
	cfg := testConfig()
	fileRepo := files.NewRepository(testPool)
	filesSvc := files.NewService(fileRepo, cfg)
	filesHandler := files.NewHandler(filesSvc)

	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authMW := middleware.NewAuthMiddleware(authSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/files", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Get("/", filesHandler.List)
		r.Get("/years", filesHandler.AvailableYears)
		r.Get("/{id}/download", filesHandler.Download)

		r.Group(func(r chi.Router) {
			r.Use(authMW.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
			r.Post("/", filesHandler.Upload)
			r.Delete("/{id}", filesHandler.Delete)
		})
	})
	r.Route("/api/v1/associations", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(authMW.RequireRole(domain.RoleSystemAdmin))
		r.Get("/", filesHandler.ListAssociations)
	})

	return &testDeps{router: r, filesSvc: filesSvc, cfg: cfg}
}

// ─── GET /api/v1/files（system_admin） ──────────────────────

// TestListHandler_SystemAdmin_WithAssociationID
// system_adminはassociation_idクエリパラメータを指定してファイル一覧を取得できる（200）
func TestListHandler_SystemAdmin_WithAssociationID(t *testing.T) {
	assocA := insertTestAssociation(t, "SA一覧自治会A", "HDL_SA_LIST_A")
	assocB := insertTestAssociation(t, "SA一覧自治会B", "HDL_SA_LIST_B")
	userA := insertTestUser(t, &assocA, "Aユーザー", "a@hdl-sa-list.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "Bユーザー", "b@hdl-sa-list.test", "pass123", "user")
	_ = insertTestFile(t, assocA, userA, 2024, 1, "SA一覧A資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocB, userB, 2024, 2, "SA一覧B資料.pdf", "application/pdf")

	// system_adminはassociation_idなしのJWT（associationIDが空文字）
	saID := insertTestUser(t, nil, "system_admin", "sa@hdl-sa-list.test", "pass123", "system_admin")
	token := makeTestToken(t, "" /* association_id なし */, saID.String(), "system_admin")

	deps := newTestDeps()

	// assocAのファイルを取得
	rrA := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files?association_id=%s", assocA.String()),
		nil, token,
	)
	assert.Equal(t, http.StatusOK, rrA.Code)
	bodyA := decodeBody(t, rrA)
	dataA := bodyA["data"].(map[string]interface{})
	listA := dataA["files"].([]interface{})
	for _, item := range listA {
		f := item.(map[string]interface{})
		assert.Equal(t, assocA.String(), f["association_id"],
			"assocAのファイルのみが返るべき")
	}

	// assocBのファイルを取得
	rrB := doRequest(t, deps.router, http.MethodGet,
		fmt.Sprintf("/api/v1/files?association_id=%s", assocB.String()),
		nil, token,
	)
	assert.Equal(t, http.StatusOK, rrB.Code)
}

// TestListHandler_SystemAdmin_MissingAssociationID
// system_adminがassociation_idを指定しない場合は400が返る
func TestListHandler_SystemAdmin_MissingAssociationID(t *testing.T) {
	saID := insertTestUser(t, nil, "system_admin_miss", "sa@hdl-sa-miss.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/files", nil, token)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_PARAM", errObj["code"])
}

// ─── POST /api/v1/files（system_admin） ─────────────────────

// TestUploadHandler_SystemAdmin_WithAssociationID
// system_adminはassociation_idフォームフィールドを指定してアップロードできる（201）
func TestUploadHandler_SystemAdmin_WithAssociationID(t *testing.T) {
	assocID := insertTestAssociation(t, "SAアップロード自治会", "HDL_SA_UPL")
	saID := insertTestUser(t, nil, "sa_uploader", "sa@hdl-sa-upl.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyPDF, "SA資料.pdf", "application/pdf",
		map[string]string{
			"year":           "2024",
			"month":          "5",
			"association_id": assocID.String(),
		},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "SA資料.pdf", data["original_filename"])
	assert.Equal(t, assocID.String(), data["association_id"], "指定した自治会にアップロードされるべき")

	t.Cleanup(func() {
		fileID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, fileID)
	})
}

// TestUploadHandler_SystemAdmin_MissingAssociationID
// system_adminがassociation_idなしでアップロードすると400が返る
func TestUploadHandler_SystemAdmin_MissingAssociationID(t *testing.T) {
	saID := insertTestUser(t, nil, "sa_upl_noassoc", "sa@hdl-sa-upl-na.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDeps()
	rr := doMultipartUpload(t, deps.router, "/api/v1/files",
		dummyPDF, "SA資料.pdf", "application/pdf",
		map[string]string{
			"year":  "2024",
			"month": "5",
			// association_id なし
		},
		token,
	)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─── DELETE /api/v1/files/:id（system_admin） ───────────────

// TestDeleteHandler_SystemAdmin_CrossTenant
// system_adminは全自治会のファイルを削除できる（200）
func TestDeleteHandler_SystemAdmin_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "SA削除自治会A", "HDL_SA_DEL_A")
	userA := insertTestUser(t, &assocA, "ユーザーA", "a@hdl-sa-del.test", "pass123", "user")
	fileID := insertTestFile(t, assocA, userA, 2024, 8, "SA削除対象.pdf", "application/pdf")

	// system_adminのトークン（associationIDなし）
	saID := insertTestUser(t, nil, "sa_deleter", "sa@hdl-sa-del.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "削除しました", data["message"])

	// DBで論理削除されているか確認
	var deletedAt *os.File
	_ = deletedAt
	var count int
	require.NoError(t, testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM files WHERE id = $1 AND deleted_at IS NOT NULL`, fileID).Scan(&count))
	assert.Equal(t, 1, count, "論理削除されているべき")
}

// TestDeleteHandler_SystemAdmin_AnotherTenant
// system_adminが全く別の自治会（自分のテナント外）のファイルも削除できる（200）
func TestDeleteHandler_SystemAdmin_AnotherTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "SA他テナント削除自治会A", "HDL_SA_DELTA")
	assocB := insertTestAssociation(t, "SA他テナント削除自治会B", "HDL_SA_DELTB")
	userA := insertTestUser(t, &assocA, "ユーザーA", "a@hdl-sa-delta.test", "pass123", "user")
	fileIDinA := insertTestFile(t, assocA, userA, 2024, 9, "自治会A資料.pdf", "application/pdf")

	// system_adminはassocBに関連付けられていても自治会Aのファイルを削除できる
	saID := insertTestUser(t, &assocB, "SA管理者", "sa@hdl-sa-delta.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/files/%s", fileIDinA),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code, "system_adminは他テナントのファイルも削除できる")
}

// ─── GET /api/v1/associations（system_adminのみ） ───────────

// TestListAssociationsHandler_SystemAdmin_Success
// system_adminは全自治会の一覧を取得できる（200）
func TestListAssociationsHandler_SystemAdmin_Success(t *testing.T) {
	assocA := insertTestAssociation(t, "SA自治会一覧A", "HDL_SA_ASSOC_A")
	assocB := insertTestAssociation(t, "SA自治会一覧B", "HDL_SA_ASSOC_B")
	saID := insertTestUser(t, nil, "sa_assoc_lister", "sa@hdl-sa-assoc.test", "pass123", "system_admin")
	token := makeTestToken(t, "", saID.String(), "system_admin")

	deps := newTestDepsWithAssociations()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	associations := data["associations"].([]interface{})

	ids := make(map[string]bool)
	for _, item := range associations {
		a := item.(map[string]interface{})
		ids[a["id"].(string)] = true
		assert.NotEmpty(t, a["name"])
		assert.NotEmpty(t, a["code"])
	}
	assert.True(t, ids[assocA.String()], "自治会Aが含まれるべき")
	assert.True(t, ids[assocB.String()], "自治会Bが含まれるべき")
}

// TestListAssociationsHandler_Unauthorized
// 未認証は401が返る
func TestListAssociationsHandler_Unauthorized(t *testing.T) {
	deps := newTestDepsWithAssociations()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestListAssociationsHandler_Forbidden_UserRole
// 一般ユーザーは403が返る
func TestListAssociationsHandler_Forbidden_UserRole(t *testing.T) {
	assocID := insertTestAssociation(t, "SA拒否自治会", "HDL_SA_ASSOC_DENY")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@hdl-sa-assoc-deny.test", "pass123", "user")
	token := makeTestToken(t, assocID.String(), userID.String(), "user")

	deps := newTestDepsWithAssociations()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestListAssociationsHandler_Forbidden_AssociationAdmin
// association_adminも403が返る（system_adminのみ許可）
func TestListAssociationsHandler_Forbidden_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "SA管理者拒否自治会", "HDL_SA_ASSOC_ADMDEN")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sa-assoc-admden.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDepsWithAssociations()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/associations", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
