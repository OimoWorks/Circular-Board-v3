package survey_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/survey"
)

// ═══════════════════════════════════════════════════════════
// Repository.CreateImage（画像レコード作成）
// ═══════════════════════════════════════════════════════════

// TestImageRepository_CreateImage_Success
// 正しいデータで画像レコードを作成できる
func TestImageRepository_CreateImage_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "画像作成自治会", "IMG_REPO_C1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@img-repo-c1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "テストアンケート", time.Now().Add(24*time.Hour))

	repo := survey.NewRepository(testPool)
	imgID := uuid.New()
	now := time.Now()
	img := &survey.SurveyImage{
		ID:            imgID,
		SurveyID:      surveyID,
		AssociationID: assocID,
		Filename:      "test.png",
		StoragePath:   filepath.Join(testUploadDir, "test.png"),
		SortOrder:     1,
		CreatedAt:     now,
	}

	err := repo.CreateImage(context.Background(), img)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE id = $1`, imgID)
	})

	got, err := repo.FindImage(context.Background(), imgID)
	require.NoError(t, err)
	assert.Equal(t, imgID, got.ID)
	assert.Equal(t, surveyID, got.SurveyID)
	assert.Equal(t, assocID, got.AssociationID)
	assert.Equal(t, "test.png", got.Filename)
	assert.Equal(t, 1, got.SortOrder)
}

// ═══════════════════════════════════════════════════════════
// Repository.ListImages（画像一覧取得）
// ═══════════════════════════════════════════════════════════

// TestImageRepository_ListImages_SortOrder
// sort_order 昇順で返される
func TestImageRepository_ListImages_SortOrder(t *testing.T) {
	assocID := insertTestAssociation(t, "画像一覧自治会", "IMG_REPO_L1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@img-repo-l1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "テストアンケート", time.Now().Add(24*time.Hour))

	insertTestSurveyImage(t, surveyID, assocID, "third.png", "/tmp/third.png", 3)
	insertTestSurveyImage(t, surveyID, assocID, "first.png", "/tmp/first.png", 1)
	insertTestSurveyImage(t, surveyID, assocID, "second.png", "/tmp/second.png", 2)

	repo := survey.NewRepository(testPool)
	images, err := repo.ListImages(context.Background(), surveyID)
	require.NoError(t, err)
	require.Len(t, images, 3)

	assert.Equal(t, "first.png", images[0].Filename)
	assert.Equal(t, "second.png", images[1].Filename)
	assert.Equal(t, "third.png", images[2].Filename)
}

// TestImageRepository_ListImages_Empty
// 画像がない場合は空スライスを返す
func TestImageRepository_ListImages_Empty(t *testing.T) {
	assocID := insertTestAssociation(t, "空画像自治会", "IMG_REPO_L2")
	userID := insertTestUser(t, &assocID, "管理者", "admin@img-repo-l2.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "テストアンケート", time.Now().Add(24*time.Hour))

	repo := survey.NewRepository(testPool)
	images, err := repo.ListImages(context.Background(), surveyID)
	require.NoError(t, err)
	assert.NotNil(t, images)
	assert.Empty(t, images)
}

// TestImageRepository_ListImages_CrossTenant
// 別自治会のアンケート画像は取得されない
func TestImageRepository_ListImages_CrossTenant(t *testing.T) {
	assocID1 := insertTestAssociation(t, "自治会A", "IMG_REPO_L3A")
	assocID2 := insertTestAssociation(t, "自治会B", "IMG_REPO_L3B")
	userID1 := insertTestUser(t, &assocID1, "管理者A", "admin@img-repo-l3a.test", "pass123", "association_admin")
	userID2 := insertTestUser(t, &assocID2, "管理者B", "admin@img-repo-l3b.test", "pass123", "association_admin")
	surveyID1 := insertTestSurvey(t, assocID1, userID1, "自治会Aのアンケート", time.Now().Add(24*time.Hour))
	surveyID2 := insertTestSurvey(t, assocID2, userID2, "自治会Bのアンケート", time.Now().Add(24*time.Hour))

	insertTestSurveyImage(t, surveyID1, assocID1, "a.png", "/tmp/a.png", 1)
	insertTestSurveyImage(t, surveyID2, assocID2, "b.png", "/tmp/b.png", 1)

	repo := survey.NewRepository(testPool)
	images, err := repo.ListImages(context.Background(), surveyID1)
	require.NoError(t, err)
	require.Len(t, images, 1)
	assert.Equal(t, "a.png", images[0].Filename)
}

// ═══════════════════════════════════════════════════════════
// Repository.FindImage（画像取得）
// ═══════════════════════════════════════════════════════════

// TestImageRepository_FindImage_NotFound
// 存在しない画像IDでErrImageNotFoundを返す
func TestImageRepository_FindImage_NotFound(t *testing.T) {
	repo := survey.NewRepository(testPool)
	_, err := repo.FindImage(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, survey.ErrImageNotFound))
}

// ═══════════════════════════════════════════════════════════
// Repository.DeleteImage（画像削除）
// ═══════════════════════════════════════════════════════════

// TestImageRepository_DeleteImage_Success
// 正常に画像レコードを削除できる
func TestImageRepository_DeleteImage_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "画像削除自治会", "IMG_REPO_D1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@img-repo-d1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "テストアンケート", time.Now().Add(24*time.Hour))
	imgID := insertTestSurveyImage(t, surveyID, assocID, "delete.png", "/tmp/delete.png", 0)

	repo := survey.NewRepository(testPool)
	err := repo.DeleteImage(context.Background(), imgID)
	require.NoError(t, err)

	_, err = repo.FindImage(context.Background(), imgID)
	assert.True(t, errors.Is(err, survey.ErrImageNotFound))
}

// TestImageRepository_DeleteImage_NotFound
// 存在しない画像IDでErrImageNotFoundを返す
func TestImageRepository_DeleteImage_NotFound(t *testing.T) {
	repo := survey.NewRepository(testPool)
	err := repo.DeleteImage(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, survey.ErrImageNotFound))
}

// ═══════════════════════════════════════════════════════════
// Repository.FindByID（画像つきアンケート取得）
// ═══════════════════════════════════════════════════════════

// TestSurveyRepository_FindByID_WithImages
// FindByID は画像も一緒に返す
func TestSurveyRepository_FindByID_WithImages(t *testing.T) {
	assocID := insertTestAssociation(t, "画像取得自治会", "IMG_REPO_F1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@img-repo-f1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "テストアンケート", time.Now().Add(24*time.Hour))
	insertTestSurveyImage(t, surveyID, assocID, "img1.png", "/tmp/img1.png", 1)
	insertTestSurveyImage(t, surveyID, assocID, "img2.png", "/tmp/img2.png", 2)

	repo := survey.NewRepository(testPool)
	sv, err := repo.FindByID(context.Background(), surveyID, nil)
	require.NoError(t, err)
	assert.Len(t, sv.Images, 2)
	assert.Equal(t, "img1.png", sv.Images[0].Filename)
	assert.Equal(t, "img2.png", sv.Images[1].Filename)
}
