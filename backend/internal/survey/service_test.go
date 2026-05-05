package survey_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/survey"
)

// ═══════════════════════════════════════════════════════════
// Service.Create（アンケート作成）
// ═══════════════════════════════════════════════════════════

// TestSurveyService_Create_Success
// 正常にアンケートを作成できる
func TestSurveyService_Create_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "サービス作成自治会", "SVC_C1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@svc-c1.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	input := survey.CreateInput{
		Title:       "テストアンケート",
		Description: "説明文",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Questions: []survey.QuestionInput{
			{
				QuestionText: "質問1",
				QuestionType: "single",
				SortOrder:    1,
				Choices: []survey.ChoiceInput{
					{ChoiceText: "選択肢A", SortOrder: 1},
					{ChoiceText: "選択肢B", SortOrder: 2},
				},
			},
		},
	}

	sv, err := svc.Create(context.Background(), assocID, userID, input)
	require.NoError(t, err)
	require.NotNil(t, sv)
	assert.Equal(t, "テストアンケート", sv.Title)
	assert.Equal(t, assocID, sv.AssociationID)
	assert.Len(t, sv.Questions, 1)
	assert.Len(t, sv.Questions[0].Choices, 2)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, sv.ID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, sv.ID)
	})
}

// TestSurveyService_Create_EmptyTitle
// タイトルが空の場合はエラーを返す
func TestSurveyService_Create_EmptyTitle(t *testing.T) {
	assocID := insertTestAssociation(t, "タイトル空自治会", "SVC_C2")
	userID := insertTestUser(t, &assocID, "管理者", "admin@svc-c2.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	input := survey.CreateInput{
		Title:     "",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Questions: []survey.QuestionInput{
			{
				QuestionText: "質問1",
				QuestionType: "single",
				Choices: []survey.ChoiceInput{
					{ChoiceText: "A"}, {ChoiceText: "B"},
				},
			},
		},
	}
	_, err := svc.Create(context.Background(), assocID, userID, input)
	assert.Error(t, err)
}

// TestSurveyService_Create_PastExpiry
// 過去の期限はエラーを返す
func TestSurveyService_Create_PastExpiry(t *testing.T) {
	assocID := insertTestAssociation(t, "過去期限自治会", "SVC_C3")
	userID := insertTestUser(t, &assocID, "管理者", "admin@svc-c3.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	input := survey.CreateInput{
		Title:     "テスト",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Questions: []survey.QuestionInput{
			{
				QuestionText: "質問1",
				QuestionType: "single",
				Choices:      []survey.ChoiceInput{{ChoiceText: "A"}, {ChoiceText: "B"}},
			},
		},
	}
	_, err := svc.Create(context.Background(), assocID, userID, input)
	assert.Error(t, err)
}

// ═══════════════════════════════════════════════════════════
// Service.CountUnanswered（未回答件数）
// ═══════════════════════════════════════════════════════════

// TestSurveyService_CountUnanswered_ReturnsCount
// 未回答件数が正しく返される
func TestSurveyService_CountUnanswered_ReturnsCount(t *testing.T) {
	assocID := insertTestAssociation(t, "未回答件数自治会", "SVC_UA1")
	userID := insertTestUser(t, &assocID, "一般ユーザー", "member@svc-ua1.test", "pass123", "member")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-ua1.test", "pass123", "association_admin")

	insertTestSurvey(t, assocID, adminID, "アンケート1", time.Now().Add(24*time.Hour))
	insertTestSurvey(t, assocID, adminID, "アンケート2", time.Now().Add(48*time.Hour))

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	count, err := svc.CountUnanswered(context.Background(), assocID, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2)
}

// ═══════════════════════════════════════════════════════════
// Service 画像関連（Upload Dir 連動テスト）
// ═══════════════════════════════════════════════════════════

// TestSurveyService_GetResults_Success
// 集計結果が取得できる
func TestSurveyService_GetResults_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "集計自治会", "SVC_RES1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-res1.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	input := survey.CreateInput{
		Title:     "集計テスト",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Questions: []survey.QuestionInput{
			{
				QuestionText: "好きな色は？",
				QuestionType: "single",
				Choices: []survey.ChoiceInput{
					{ChoiceText: "赤", SortOrder: 1},
					{ChoiceText: "青", SortOrder: 2},
				},
			},
		},
	}
	sv, err := svc.Create(context.Background(), assocID, adminID, input)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, sv.ID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, sv.ID)
	})

	assocIDCopy := assocID
	result, err := svc.GetResults(context.Background(), "association_admin", &assocIDCopy, sv.ID)
	require.NoError(t, err)
	assert.Equal(t, sv.ID, result.SurveyID)
	assert.Equal(t, "集計テスト", result.Title)
	assert.Len(t, result.Questions, 1)
}

// TestSurveyService_Delete_Success
// 管理者が自治会のアンケートを削除できる
func TestSurveyService_Delete_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "削除テスト自治会", "SVC_DEL1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@svc-del1.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	surveyID := insertTestSurvey(t, assocID, adminID, "削除対象", time.Now().Add(24*time.Hour))

	assocIDCopy := assocID
	err := svc.Delete(context.Background(), "association_admin", &assocIDCopy, surveyID)
	require.NoError(t, err)
}

// TestSurveyService_Delete_CrossTenant
// 別自治会のアンケートはErrNotFoundを返す
func TestSurveyService_Delete_CrossTenant(t *testing.T) {
	assocID1 := insertTestAssociation(t, "削除自治会A", "SVC_DEL2A")
	assocID2 := insertTestAssociation(t, "削除自治会B", "SVC_DEL2B")
	adminID1 := insertTestUser(t, &assocID1, "管理者A", "admin@svc-del2a.test", "pass123", "association_admin")
	insertTestUser(t, &assocID2, "管理者B", "admin@svc-del2b.test", "pass123", "association_admin")

	repo := survey.NewRepository(testPool)
	svc := survey.NewService(repo)

	surveyID := insertTestSurvey(t, assocID1, adminID1, "自治会Aのアンケート", time.Now().Add(24*time.Hour))

	err := svc.Delete(context.Background(), "association_admin", &assocID2, surveyID)
	assert.ErrorIs(t, err, survey.ErrNotFound)
}

// TestImageHandler_UploadImage_FileOnDisk
// 画像ファイルがディスクに保存される（統合テスト的確認）
func TestImageHandler_UploadImage_FileOnDisk(t *testing.T) {
	assocID := insertTestAssociation(t, "ディスク保存自治会", "SVC_IMG1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@svc-img1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, userID, "画像テスト", time.Now().Add(24*time.Hour))

	imgID := uuid.New()
	imgPath := filepath.Join(testUploadDir, "survey_test_file.png")
	require.NoError(t, os.WriteFile(imgPath, minimalPNGBytes, 0o644))

	repo := survey.NewRepository(testPool)
	img := &survey.SurveyImage{
		ID:            imgID,
		SurveyID:      surveyID,
		AssociationID: assocID,
		Filename:      "survey_test_file.png",
		StoragePath:   imgPath,
		SortOrder:     0,
		CreatedAt:     time.Now(),
	}
	require.NoError(t, repo.CreateImage(context.Background(), img))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE id = $1`, imgID)
		_ = os.Remove(imgPath)
	})

	got, err := repo.FindImage(context.Background(), imgID)
	require.NoError(t, err)
	assert.Equal(t, imgPath, got.StoragePath)

	data, err := os.ReadFile(got.StoragePath)
	require.NoError(t, err)
	assert.Equal(t, minimalPNGBytes, data)
}
