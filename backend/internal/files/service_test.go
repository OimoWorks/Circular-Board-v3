package files_test

// サービス層のテスト。
// ロールチェック（管理者のみアップロード可 / 一般ユーザーは不可）はミドルウェアで
// 強制されるため、ハンドラーテスト（handler_test.go）でカバーしている。
// サービス層ではビジネスルール（MIME・サイズ・テナント境界）のみをテストする。

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/files"
)

// ─── テスト用サービス初期化 ──────────────────────────────────
func newFilesService() *files.Service {
	repo := files.NewRepository(testPool)
	return files.NewService(repo, testConfig())
}

// ═══════════════════════════════════════════════════════════
// Service.Upload（ファイルアップロード）
// ═══════════════════════════════════════════════════════════

// TestService_Upload_PDF_Success
// PDFをアップロードするとDBレコードが作成され物理ファイルが保存される
func TestService_Upload_PDF_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "アップロードPDF自治会", "SVC_UPL_PDF")
	userID := insertTestUser(t, &assocID, "PDFアップローダー", "pdf@svc-upl.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "テスト資料.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            6,
	}

	result, err := svc.Upload(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	t.Cleanup(func() {
		_ = os.Remove(result.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, result.ID)
	})

	assert.Equal(t, assocID, result.AssociationID)
	assert.Equal(t, userID, result.UploadedBy)
	assert.Equal(t, 2024, result.Year)
	assert.Equal(t, 6, result.Month)
	assert.Equal(t, "テスト資料.pdf", result.OriginalFilename)
	assert.Equal(t, "application/pdf", result.MimeType)
	assert.Equal(t, ".pdf", filepath.Ext(result.StoragePath))
	assert.True(t, result.FileSize > 0)

	// 物理ファイルが存在する
	_, statErr := os.Stat(result.StoragePath)
	assert.NoError(t, statErr, "アップロードされたファイルがディスク上に存在するべき")
}

// TestService_Upload_JPEG_Success
// JPEGファイルもアップロードできる
func TestService_Upload_JPEG_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "アップロードJPEG自治会", "SVC_UPL_JPG")
	userID := insertTestUser(t, &assocID, "JPEGアップローダー", "jpg@svc-upl.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyJPEG),
		OriginalFilename: "写真.jpg",
		Size:             int64(len(dummyJPEG)),
		MimeType:         "image/jpeg",
		Year:             2024,
		Month:            8,
	}

	result, err := svc.Upload(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(result.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, result.ID)
	})

	assert.Equal(t, "image/jpeg", result.MimeType)
	assert.Equal(t, ".jpg", filepath.Ext(result.StoragePath))
}

// TestService_Upload_PNG_Success
// PNGファイルもアップロードできる
func TestService_Upload_PNG_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "アップロードPNG自治会", "SVC_UPL_PNG")
	userID := insertTestUser(t, &assocID, "PNGアップローダー", "png@svc-upl.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyPNG),
		OriginalFilename: "画像.png",
		Size:             int64(len(dummyPNG)),
		MimeType:         "image/png",
		Year:             2024,
		Month:            9,
	}

	result, err := svc.Upload(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(result.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, result.ID)
	})

	assert.Equal(t, "image/png", result.MimeType)
	assert.Equal(t, ".png", filepath.Ext(result.StoragePath))
}

// TestService_Upload_StoragePathStructure
// 保存パスが UploadDir/{assocID}/{year}/{month:02d}/{uuid}.ext の構造になっている
func TestService_Upload_StoragePathStructure(t *testing.T) {
	assocID := insertTestAssociation(t, "パス構造テスト自治会", "SVC_PATH_1")
	userID := insertTestUser(t, &assocID, "パスユーザー", "path@svc-path.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "パス確認.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            3,
	}

	result, err := svc.Upload(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(result.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, result.ID)
	})

	cfg := testConfig()
	expectedDir := filepath.Join(cfg.UploadDir, assocID.String(), "2024", "03")
	assert.Equal(t, expectedDir, filepath.Dir(result.StoragePath),
		"保存パスは {UploadDir}/{assocID}/{year}/{month:02d}/ 形式であるべき")
}

// TestService_Upload_YearMonthDefault
// Year/Month が 0 のとき現在の年月が使われる
func TestService_Upload_YearMonthDefault(t *testing.T) {
	assocID := insertTestAssociation(t, "デフォルト年月自治会", "SVC_UPL_DEF")
	userID := insertTestUser(t, &assocID, "デフォルトユーザー", "def@svc-def.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "デフォルト.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             0,
		Month:            0,
	}

	result, err := svc.Upload(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(result.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, result.ID)
	})

	assert.NotZero(t, result.Year, "年が自動設定されているべき")
	assert.NotZero(t, result.Month, "月が自動設定されているべき")
}

// TestService_Upload_FileTooLarge
// 10MB 超のファイルは ErrFileTooLarge が返る
func TestService_Upload_FileTooLarge(t *testing.T) {
	assocID := insertTestAssociation(t, "サイズ超過自治会", "SVC_UPL_LARGE")
	userID := insertTestUser(t, &assocID, "大ファイルユーザー", "large@svc-large.test", "pass123", "association_admin")

	cfg := testConfig()
	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "huge.pdf",
		Size:             cfg.MaxUploadBytes + 1, // 10MB + 1バイト
		MimeType:         "application/pdf",
	}

	_, err := svc.Upload(context.Background(), assocID, userID, input)
	assert.True(t, errors.Is(err, files.ErrFileTooLarge))
}

// TestService_Upload_UnsupportedMIME
// 非対応 MIME タイプは ErrUnsupportedMIME が返る
func TestService_Upload_UnsupportedMIME(t *testing.T) {
	assocID := insertTestAssociation(t, "非対応MIME自治会", "SVC_UPL_MIME")
	userID := insertTestUser(t, &assocID, "MIMEユーザー", "mime@svc-mime.test", "pass123", "association_admin")

	svc := newFilesService()
	input := files.UploadInput{
		Reader:           bytes.NewReader([]byte("dummy text content")),
		OriginalFilename: "test.txt",
		Size:             18,
		MimeType:         "text/plain",
	}

	_, err := svc.Upload(context.Background(), assocID, userID, input)
	assert.True(t, errors.Is(err, files.ErrUnsupportedMIME))
}

// ═══════════════════════════════════════════════════════════
// Service.Delete（ファイル削除）
// ═══════════════════════════════════════════════════════════

// TestService_Delete_Success
// 管理者が削除するとDBが論理削除されディスク上の物理ファイルも消える
func TestService_Delete_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "削除成功自治会", "SVC_DEL_1")
	userID := insertTestUser(t, &assocID, "削除ユーザー", "del@svc-del-1.test", "pass123", "association_admin")

	svc := newFilesService()
	uploaded, err := svc.Upload(context.Background(), assocID, userID, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "削除対象.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            9,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, uploaded.ID)
	})

	// 削除前はディスク上に存在する
	_, err = os.Stat(uploaded.StoragePath)
	require.NoError(t, err)

	// 削除実行
	require.NoError(t, svc.Delete(context.Background(), assocID, uploaded.ID))

	// 物理ファイルが消えている
	_, statErr := os.Stat(uploaded.StoragePath)
	assert.True(t, os.IsNotExist(statErr), "削除後はディスク上のファイルが消えているべき")

	// GetFile でも取得できない
	_, errGet := svc.GetFile(context.Background(), assocID, uploaded.ID)
	assert.True(t, errors.Is(errGet, files.ErrNotFound))
}

// TestService_Delete_NotFound
// 存在しないファイルの削除は ErrNotFound
func TestService_Delete_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "削除不在自治会", "SVC_DEL_NF")

	svc := newFilesService()
	err := svc.Delete(context.Background(), assocID, uuid.New())
	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// TestService_Delete_CrossTenant
// 他の自治会のファイルは削除できない
func TestService_Delete_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "削除テナントA", "SVC_DEL_TA")
	assocB := insertTestAssociation(t, "削除テナントB", "SVC_DEL_TB")
	userA := insertTestUser(t, &assocA, "テナントAユーザー", "del@svc-del-ta.test", "pass123", "association_admin")

	svc := newFilesService()
	uploaded, err := svc.Upload(context.Background(), assocA, userA, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "テナントA資料.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            10,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(uploaded.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, uploaded.ID)
	})

	// テナントBで削除しようとすると ErrNotFound（テナント境界の強制）
	err = svc.Delete(context.Background(), assocB, uploaded.ID)
	assert.True(t, errors.Is(err, files.ErrNotFound),
		"他の自治会IDではファイルを削除できない")

	// テナントAのファイルはまだ存在している
	got, err := svc.GetFile(context.Background(), assocA, uploaded.ID)
	require.NoError(t, err)
	assert.Equal(t, uploaded.ID, got.ID)
}

// ═══════════════════════════════════════════════════════════
// Service.GetFile（ファイルダウンロード用取得）
// ═══════════════════════════════════════════════════════════

// TestService_GetFile_Success_AnyRole
// ロールを問わず GetFile でファイル情報を取得できる（ロール制限はハンドラー層）
func TestService_GetFile_Success_AnyRole(t *testing.T) {
	assocID := insertTestAssociation(t, "ダウンロード全ロール自治会", "SVC_DL_ROLE")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-dl.test", "pass123", "association_admin")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "user@svc-dl.test", "pass123", "user")
	fileID := insertTestFile(t, assocID, adminID, 2024, 5, "ダウンロード資料.pdf", "application/pdf")

	svc := newFilesService()

	// 管理者が取得
	gotByAdmin, err := svc.GetFile(context.Background(), assocID, fileID)
	require.NoError(t, err)
	assert.Equal(t, fileID, gotByAdmin.ID)

	// 一般ユーザーが取得（サービス層はロールを見ない）
	gotByUser, err := svc.GetFile(context.Background(), assocID, fileID)
	require.NoError(t, err)
	assert.Equal(t, fileID, gotByUser.ID)
	_ = userID
}

// TestService_GetFile_CrossTenant
// 他の自治会のファイルはダウンロードできない
func TestService_GetFile_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "ダウンロードテナントA", "SVC_DL_TA")
	assocB := insertTestAssociation(t, "ダウンロードテナントB", "SVC_DL_TB")
	userA := insertTestUser(t, &assocA, "ダウンロードTAユーザー", "dl@svc-dl-ta.test", "pass123", "user")
	fileID := insertTestFile(t, assocA, userA, 2024, 6, "テナントA秘密資料.pdf", "application/pdf")

	svc := newFilesService()
	// テナントBのassocIDで取得しようとすると ErrNotFound
	_, err := svc.GetFile(context.Background(), assocB, fileID)
	assert.True(t, errors.Is(err, files.ErrNotFound),
		"他の自治会のファイルはダウンロードできない")
}

// TestService_GetFile_DeletedFile
// 論理削除済みファイルはダウンロードできない
func TestService_GetFile_DeletedFile(t *testing.T) {
	assocID := insertTestAssociation(t, "削除済みDL自治会", "SVC_DL_DEL")
	userID := insertTestUser(t, &assocID, "削除済みDLユーザー", "del@svc-dl-del.test", "pass123", "association_admin")

	svc := newFilesService()
	uploaded, err := svc.Upload(context.Background(), assocID, userID, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "削除済み資料.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            11,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, uploaded.ID)
	})

	// 削除してから取得を試みる
	require.NoError(t, svc.Delete(context.Background(), assocID, uploaded.ID))

	_, errGet := svc.GetFile(context.Background(), assocID, uploaded.ID)
	assert.True(t, errors.Is(errGet, files.ErrNotFound),
		"論理削除済みファイルは GetFile で取得できない")
}

// TestService_GetFile_NotFound
// 存在しないIDは ErrNotFound
func TestService_GetFile_NotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "GetFile不在自治会", "SVC_GET_NF")

	svc := newFilesService()
	_, err := svc.GetFile(context.Background(), assocID, uuid.New())
	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// ═══════════════════════════════════════════════════════════
// Service.List（ファイル一覧取得）
// ═══════════════════════════════════════════════════════════

// TestService_List_YearMonthHierarchy
// 年月フィルタで絞り込みができ、最新年月が先頭に来る
func TestService_List_YearMonthHierarchy(t *testing.T) {
	assocID := insertTestAssociation(t, "一覧階層自治会", "SVC_LIST_HIER")
	userID := insertTestUser(t, &assocID, "階層ユーザー", "hier@svc-list.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2023, 1, "2023-01.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 3, "2024-03.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 6, "2024-06.pdf", "application/pdf")

	svc := newFilesService()

	// 年フィルタ
	list2024, err := svc.List(context.Background(), assocID, 2024, 0)
	require.NoError(t, err)
	for _, f := range list2024 {
		assert.Equal(t, 2024, f.Year, "2024年のみが含まれるべき")
	}

	// 年月フィルタ
	listJune, err := svc.List(context.Background(), assocID, 2024, 6)
	require.NoError(t, err)
	for _, f := range listJune {
		assert.Equal(t, 2024, f.Year)
		assert.Equal(t, 6, f.Month, "6月のみが含まれるべき")
	}
}

// TestService_List_LatestFirst
// 最新年月のファイルが先頭に来る（降順ソート）
func TestService_List_LatestFirst(t *testing.T) {
	assocID := insertTestAssociation(t, "ソート最新自治会", "SVC_LIST_SORT")
	userID := insertTestUser(t, &assocID, "ソートユーザー", "sort@svc-list.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2023, 1, "古い資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2024, 12, "新しい資料.pdf", "application/pdf")

	svc := newFilesService()
	list, err := svc.List(context.Background(), assocID, 0, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list), 2)

	first := list[0]
	last := list[len(list)-1]
	assert.GreaterOrEqual(t,
		first.Year*100+first.Month,
		last.Year*100+last.Month,
		"最新年月が先頭に来るべき")
}

// TestService_List_CrossTenant
// 他の自治会のデータが混入しない
func TestService_List_CrossTenant(t *testing.T) {
	assocA := insertTestAssociation(t, "一覧テナントA", "SVC_LIST_TA")
	assocB := insertTestAssociation(t, "一覧テナントB", "SVC_LIST_TB")
	userA := insertTestUser(t, &assocA, "一覧TAユーザー", "list@svc-list-ta.test", "pass123", "user")
	userB := insertTestUser(t, &assocB, "一覧TBユーザー", "list@svc-list-tb.test", "pass123", "user")

	_ = insertTestFile(t, assocA, userA, 2024, 1, "A資料.pdf", "application/pdf")
	_ = insertTestFile(t, assocB, userB, 2024, 1, "B資料.pdf", "application/pdf")

	svc := newFilesService()
	listA, err := svc.List(context.Background(), assocA, 0, 0)
	require.NoError(t, err)

	for _, f := range listA {
		assert.Equal(t, assocA, f.AssociationID,
			"自治会Aの一覧に自治会Bのファイルが混入してはいけない")
	}
}

// ═══════════════════════════════════════════════════════════
// Service.AvailableYears
// ═══════════════════════════════════════════════════════════

// TestService_AvailableYears
// ファイルが存在する年の一覧が返る
func TestService_AvailableYears(t *testing.T) {
	assocID := insertTestAssociation(t, "年一覧サービス自治会", "SVC_YEARS_1")
	userID := insertTestUser(t, &assocID, "年一覧SVCユーザー", "yrs@svc-years.test", "pass123", "user")

	_ = insertTestFile(t, assocID, userID, 2021, 1, "2021.pdf", "application/pdf")
	_ = insertTestFile(t, assocID, userID, 2022, 6, "2022.pdf", "application/pdf")

	svc := newFilesService()
	years, err := svc.AvailableYears(context.Background(), assocID)
	require.NoError(t, err)
	assert.Contains(t, years, 2021)
	assert.Contains(t, years, 2022)
}

// ═══════════════════════════════════════════════════════════
// system_admin 対応：Service.AdminGetFile
// ═══════════════════════════════════════════════════════════

// TestService_AdminGetFile_CrossTenant_Success
// system_adminは他自治会のファイルをIDのみで取得できる
func TestService_AdminGetFile_CrossTenant_Success(t *testing.T) {
	assocA := insertTestAssociation(t, "SA取得自治会A", "SVC_SA_GET_A")
	assocB := insertTestAssociation(t, "SA取得自治会B", "SVC_SA_GET_B")
	userA := insertTestUser(t, &assocA, "ユーザーA", "a@svc-sa-get.test", "pass123", "user")
	fileID := insertTestFile(t, assocA, userA, 2024, 6, "SA取得テスト.pdf", "application/pdf")

	svc := newFilesService()

	// 通常のGetFileはテナント境界があるためassocBでは取得できない
	_, errCross := svc.GetFile(context.Background(), assocB, fileID)
	assert.True(t, errors.Is(errCross, files.ErrFileTooLarge) || errors.Is(errCross, files.ErrNotFound),
		"通常GetFileは他テナントのファイルを取得できない")

	// AdminGetFileはテナント境界なしで取得できる
	got, err := svc.AdminGetFile(context.Background(), fileID)
	require.NoError(t, err)
	assert.Equal(t, fileID, got.ID)
	assert.Equal(t, assocA, got.AssociationID)
	_ = assocB
}

// TestService_AdminGetFile_NotFound
// 存在しないIDはErrNotFound
func TestService_AdminGetFile_NotFound(t *testing.T) {
	svc := newFilesService()
	_, err := svc.AdminGetFile(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// ═══════════════════════════════════════════════════════════
// system_admin 対応：Service.AdminDelete
// ═══════════════════════════════════════════════════════════

// TestService_AdminDelete_CrossTenant_Success
// system_adminは他自治会のファイルを削除できる
func TestService_AdminDelete_CrossTenant_Success(t *testing.T) {
	assocA := insertTestAssociation(t, "SA削除自治会A", "SVC_SA_DEL_A")
	assocB := insertTestAssociation(t, "SA削除自治会B", "SVC_SA_DEL_B")
	userA := insertTestUser(t, &assocA, "ユーザーA", "a@svc-sa-del.test", "pass123", "association_admin")

	svc := newFilesService()
	uploaded, err := svc.Upload(context.Background(), assocA, userA, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "SA削除テスト.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            7,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, uploaded.ID)
	})

	// assocBの管理者として通常Deleteしようとしても失敗する
	errCross := svc.Delete(context.Background(), assocB, uploaded.ID)
	assert.True(t, errors.Is(errCross, files.ErrNotFound),
		"通常Deleteは他テナントのファイルを削除できない")

	// AdminDeleteはテナント境界なしで削除できる
	err = svc.AdminDelete(context.Background(), uploaded.ID)
	require.NoError(t, err, "AdminDeleteは他テナントのファイルも削除できる")

	// GetFileでも取得できなくなっている
	_, errGet := svc.AdminGetFile(context.Background(), uploaded.ID)
	assert.True(t, errors.Is(errGet, files.ErrNotFound), "AdminDelete後はファイルが取得できない")
}

// TestService_AdminDelete_NotFound
// 存在しないIDはErrNotFound
func TestService_AdminDelete_NotFound(t *testing.T) {
	svc := newFilesService()
	err := svc.AdminDelete(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, files.ErrNotFound))
}

// ═══════════════════════════════════════════════════════════
// system_admin 対応：Service.ListAssociations
// ═══════════════════════════════════════════════════════════

// TestService_ListAssociations_Success
// system_adminは全自治会の一覧を取得できる
func TestService_ListAssociations_Success(t *testing.T) {
	assocA := insertTestAssociation(t, "SA一覧自治会A", "SVC_SA_ASSOC_A")
	assocB := insertTestAssociation(t, "SA一覧自治会B", "SVC_SA_ASSOC_B")

	svc := newFilesService()
	associations, err := svc.ListAssociations(context.Background())
	require.NoError(t, err)
	require.NotNil(t, associations)

	ids := make(map[uuid.UUID]bool)
	for _, a := range associations {
		ids[a.ID] = true
	}
	assert.True(t, ids[assocA], "自治会Aが一覧に含まれるべき")
	assert.True(t, ids[assocB], "自治会Bが一覧に含まれるべき")
}

// TestService_Upload_SystemAdmin_ToAnyAssociation
// system_adminは自治会を選択してアップロードできる（サービス層はロールを見ない）
func TestService_Upload_SystemAdmin_ToAnyAssociation(t *testing.T) {
	assocA := insertTestAssociation(t, "SAアップロード自治会A", "SVC_SA_UPL_A")
	assocB := insertTestAssociation(t, "SAアップロード自治会B", "SVC_SA_UPL_B")
	// system_admin は associations に属さないため association_id なしで作成
	saID := insertTestUser(t, nil, "system_admin", "sa@svc-sa-upl.test", "pass123", "system_admin")

	svc := newFilesService()

	// assocAに対してアップロード（system_adminのuserIDを使用）
	resultA, err := svc.Upload(context.Background(), assocA, saID, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "SA自治会A向け.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            8,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(resultA.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, resultA.ID)
	})
	assert.Equal(t, assocA, resultA.AssociationID, "assocAにアップロードされるべき")

	// assocBに対してもアップロードできる
	resultB, err := svc.Upload(context.Background(), assocB, saID, files.UploadInput{
		Reader:           bytes.NewReader(dummyPDF),
		OriginalFilename: "SA自治会B向け.pdf",
		Size:             int64(len(dummyPDF)),
		MimeType:         "application/pdf",
		Year:             2024,
		Month:            9,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(resultB.StoragePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM files WHERE id = $1`, resultB.ID)
	})
	assert.Equal(t, assocB, resultB.AssociationID, "assocBにアップロードされるべき")
}
