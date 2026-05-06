package survey_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════
// POST /api/v1/surveys/:id/images（画像アップロード）
// ═══════════════════════════════════════════════════════════

// TestUploadImageHandler_Success_Admin
// association_admin が画像をアップロードできる
func TestUploadImageHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "アップロード自治会", "HDL_IMG_U1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-u1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "画像テスト", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartRequest(t, deps.router,
		fmt.Sprintf("/api/v1/surveys/%s/images", surveyID),
		minimalPNGBytes, "test.png", "image/png", token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, surveyID.String(), data["survey_id"])
	assert.Equal(t, assocID.String(), data["association_id"])
	assert.Equal(t, "test.png", data["filename"])

	// ディスク上のファイルが実際に存在するか確認し、後片付け
	imgID, err := uuid.Parse(data["id"].(string))
	require.NoError(t, err)
	t.Cleanup(func() {
		var storagePath string
		_ = testPool.QueryRow(context.Background(),
			`SELECT storage_path FROM survey_images WHERE id = $1`, imgID,
		).Scan(&storagePath)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_images WHERE id = $1`, imgID)
		if storagePath != "" {
			_ = os.Remove(storagePath)
		}
	})
}

// TestUploadImageHandler_Unauthorized
// 認証なしでアクセスすると401を返す
func TestUploadImageHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "未認証アップロード自治会", "HDL_IMG_U2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-u2.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))

	deps := newTestDeps()
	rr := doMultipartRequest(t, deps.router,
		fmt.Sprintf("/api/v1/surveys/%s/images", surveyID),
		minimalPNGBytes, "test.png", "image/png", "",
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestUploadImageHandler_MemberForbidden
// member ロールは403を返す
func TestUploadImageHandler_MemberForbidden(t *testing.T) {
	assocID := insertTestAssociation(t, "メンバー禁止自治会", "HDL_IMG_U3")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-u3.test", "pass123", "association_admin")
	memberID := insertTestUser(t, &assocID, "メンバー", "member@hdl-img-u3.test", "pass123", "member")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), memberID.String(), "member")

	deps := newTestDeps()
	rr := doMultipartRequest(t, deps.router,
		fmt.Sprintf("/api/v1/surveys/%s/images", surveyID),
		minimalPNGBytes, "test.png", "image/png", token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestUploadImageHandler_SurveyNotFound
// 存在しないアンケートIDでは404を返す
func TestUploadImageHandler_SurveyNotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "存在しないアンケート自治会", "HDL_IMG_U4")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-u4.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartRequest(t, deps.router,
		fmt.Sprintf("/api/v1/surveys/%s/images", uuid.New()),
		minimalPNGBytes, "test.png", "image/png", token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestUploadImageHandler_CrossTenant
// 別自治会のアンケートには404を返す
func TestUploadImageHandler_CrossTenant(t *testing.T) {
	assocID1 := insertTestAssociation(t, "テナントA", "HDL_IMG_U5A")
	assocID2 := insertTestAssociation(t, "テナントB", "HDL_IMG_U5B")
	adminID1 := insertTestUser(t, &assocID1, "管理者A", "admin@hdl-img-u5a.test", "pass123", "association_admin")
	adminID2 := insertTestUser(t, &assocID2, "管理者B", "admin@hdl-img-u5b.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID1, adminID1, "テナントAのアンケート", time.Now().Add(24*time.Hour))
	token2 := makeTestToken(t, assocID2.String(), adminID2.String(), "association_admin")

	deps := newTestDeps()
	rr := doMultipartRequest(t, deps.router,
		fmt.Sprintf("/api/v1/surveys/%s/images", surveyID),
		minimalPNGBytes, "test.png", "image/png", token2,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DELETE /api/v1/surveys/:id/images/:image_id（画像削除）
// ═══════════════════════════════════════════════════════════

// TestDeleteImageHandler_Success_Admin
// association_admin が自分の自治会の画像を削除できる
func TestDeleteImageHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "削除ハンドラ自治会", "HDL_IMG_D1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-d1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テストアンケート", time.Now().Add(24*time.Hour))

	imgPath := filepath.Join(testUploadDir, "hdl_del_test.png")
	require.NoError(t, os.WriteFile(imgPath, minimalPNGBytes, 0o644))
	imgID := insertTestSurveyImage(t, surveyID, assocID, "hdl_del_test.png", imgPath, 0)
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestDeleteImageHandler_Unauthorized
// 認証なしでは401を返す
func TestDeleteImageHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "未認証削除自治会", "HDL_IMG_D2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-d2.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))
	imgID := insertTestSurveyImage(t, surveyID, assocID, "x.png", "/tmp/x.png", 0)

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, "",
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestDeleteImageHandler_CrossTenant
// 別自治会の画像は403を返す
func TestDeleteImageHandler_CrossTenant(t *testing.T) {
	assocID1 := insertTestAssociation(t, "テナントA", "HDL_IMG_D3A")
	assocID2 := insertTestAssociation(t, "テナントB", "HDL_IMG_D3B")
	adminID1 := insertTestUser(t, &assocID1, "管理者A", "admin@hdl-img-d3a.test", "pass123", "association_admin")
	adminID2 := insertTestUser(t, &assocID2, "管理者B", "admin@hdl-img-d3b.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID1, adminID1, "テナントAのアンケート", time.Now().Add(24*time.Hour))
	imgID := insertTestSurveyImage(t, surveyID, assocID1, "cross.png", "/tmp/cross.png", 0)
	token2 := makeTestToken(t, assocID2.String(), adminID2.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, token2,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestDeleteImageHandler_ImageNotFound
// 存在しない画像IDでは404を返す
func TestDeleteImageHandler_ImageNotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "存在しない画像自治会", "HDL_IMG_D4")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-d4.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, uuid.New()),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// GET /api/v1/surveys/:id/images/:image_id（画像取得）
// ═══════════════════════════════════════════════════════════

// TestGetImageHandler_Success
// 全ロールで画像を取得できる
func TestGetImageHandler_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "画像取得ハンドラ自治会", "HDL_IMG_G1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-g1.test", "pass123", "association_admin")
	memberID := insertTestUser(t, &assocID, "メンバー", "member@hdl-img-g1.test", "pass123", "member")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))

	imgPath := filepath.Join(testUploadDir, "hdl_get_test.png")
	require.NoError(t, os.WriteFile(imgPath, minimalPNGBytes, 0o644))
	imgID := insertTestSurveyImage(t, surveyID, assocID, "hdl_get_test.png", imgPath, 0)
	t.Cleanup(func() { _ = os.Remove(imgPath) })

	token := makeTestToken(t, assocID.String(), memberID.String(), "member")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "image/")
	assert.Equal(t, minimalPNGBytes, rr.Body.Bytes())
}

// TestGetImageHandler_Unauthorized
// 認証なしでは401を返す
func TestGetImageHandler_Unauthorized(t *testing.T) {
	assocID := insertTestAssociation(t, "未認証画像取得自治会", "HDL_IMG_G2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-g2.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))
	imgID := insertTestSurveyImage(t, surveyID, assocID, "x.png", "/tmp/x.png", 0)

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, "",
	)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestGetImageHandler_CrossTenant
// 別自治会の画像は403を返す
func TestGetImageHandler_CrossTenant(t *testing.T) {
	assocID1 := insertTestAssociation(t, "テナントA", "HDL_IMG_G3A")
	assocID2 := insertTestAssociation(t, "テナントB", "HDL_IMG_G3B")
	adminID1 := insertTestUser(t, &assocID1, "管理者A", "admin@hdl-img-g3a.test", "pass123", "association_admin")
	memberID2 := insertTestUser(t, &assocID2, "メンバーB", "member@hdl-img-g3b.test", "pass123", "member")
	surveyID := insertTestSurvey(t, assocID1, adminID1, "テナントAのアンケート", time.Now().Add(24*time.Hour))
	imgID := insertTestSurveyImage(t, surveyID, assocID1, "cross.png", "/tmp/cross.png", 0)
	token2 := makeTestToken(t, assocID2.String(), memberID2.String(), "member")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, imgID),
		nil, token2,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestGetImageHandler_ImageNotFound
// 存在しない画像IDでは404を返す
func TestGetImageHandler_ImageNotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "存在しない画像取得自治会", "HDL_IMG_G4")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-g4.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "テスト", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID, uuid.New()),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestGetImageHandler_WrongSurveyID
// image_id は正しいが survey_id が違う場合は404を返す
func TestGetImageHandler_WrongSurveyID(t *testing.T) {
	assocID := insertTestAssociation(t, "スワップ自治会", "HDL_IMG_G5")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-img-g5.test", "pass123", "association_admin")
	surveyID1 := insertTestSurvey(t, assocID, adminID, "アンケート1", time.Now().Add(24*time.Hour))
	surveyID2 := insertTestSurvey(t, assocID, adminID, "アンケート2", time.Now().Add(24*time.Hour))

	imgPath := filepath.Join(testUploadDir, "hdl_swap_test.png")
	require.NoError(t, os.WriteFile(imgPath, minimalPNGBytes, 0o644))
	imgID := insertTestSurveyImage(t, surveyID1, assocID, "swap.png", imgPath, 0)
	t.Cleanup(func() { _ = os.Remove(imgPath) })

	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/surveys/%s/images/%s", surveyID2, imgID),
		nil, token,
	)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// POST /api/v1/surveys（アンケート作成ハンドラ）
// ═══════════════════════════════════════════════════════════

// TestCreateSurveyHandler_Success_Admin
// association_admin がアンケートを作成できる
func TestCreateSurveyHandler_Success_Admin(t *testing.T) {
	assocID := insertTestAssociation(t, "作成ハンドラ自治会", "HDL_SV_C1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-c1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/surveys",
		map[string]interface{}{
			"title":      "テストアンケート",
			"description": "説明",
			"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			"questions": []map[string]interface{}{
				{
					"question_text": "質問1",
					"question_type": "single",
					"sort_order":    1,
					"choices": []map[string]interface{}{
						{"choice_text": "選択肢A", "sort_order": 1},
						{"choice_text": "選択肢B", "sort_order": 2},
					},
				},
			},
		},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "テストアンケート", data["title"])
	assert.NotEmpty(t, data["id"])

	t.Cleanup(func() {
		svID, _ := uuid.Parse(data["id"].(string))
		_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, svID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, svID)
	})
}

// TestCreateSurveyHandler_MemberForbidden
// member ロールは403を返す
func TestCreateSurveyHandler_MemberForbidden(t *testing.T) {
	assocID := insertTestAssociation(t, "メンバー作成自治会", "HDL_SV_C2")
	memberID := insertTestUser(t, &assocID, "メンバー", "member@hdl-sv-c2.test", "pass123", "member")
	token := makeTestToken(t, assocID.String(), memberID.String(), "member")

	deps := newTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/surveys",
		map[string]interface{}{
			"title":      "テスト",
			"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			"questions":  []interface{}{},
		},
		token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ═══════════════════════════════════════════════════════════
// DB権限チェック（RequireFeature ミドルウェア）
// ═══════════════════════════════════════════════════════════

// TestSurveyListHandler_PermAware_WithPermission
// DB権限あり（surveys:view=true）なら一覧取得できる（200）
func TestSurveyListHandler_PermAware_WithPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "閲覧権限あり自治会", "HDL_SV_PA_L1")
	userID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-pa-l1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/surveys", nil, token)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestSurveyListHandler_PermAware_NoViewPermission
// DB権限なし（surveys:view=false）なら一覧取得を拒否される（403）
func TestSurveyListHandler_PermAware_NoViewPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "閲覧権限なし自治会", "HDL_SV_PA_L2")
	userID := insertTestUser(t, &assocID, "ユーザー", "user@hdl-sv-pa-l2.test", "pass123", "user_admin")
	token := makeTestToken(t, assocID.String(), userID.String(), "user_admin")

	roleID := getRoleIDByName(t, "user_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, false, false, false, false, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodGet, "/api/v1/surveys", nil, token)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestSurveyCreateHandler_PermAware_WithPermission
// DB権限あり（surveys:create=true）なら作成できる（201）
func TestSurveyCreateHandler_PermAware_WithPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "作成権限あり自治会", "HDL_SV_PA_C1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-pa-c1.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/surveys",
		map[string]interface{}{
			"title":       "権限あり作成",
			"description": "",
			"expires_at":  time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			"questions": []map[string]interface{}{
				{
					"question_text": "質問1",
					"question_type": "single",
					"sort_order":    1,
					"choices": []map[string]interface{}{
						{"choice_text": "A", "sort_order": 1},
						{"choice_text": "B", "sort_order": 2},
					},
				},
			},
		},
		token,
	)

	assert.Equal(t, http.StatusCreated, rr.Code)

	t.Cleanup(func() {
		body := decodeBody(t, rr)
		if data, ok := body["data"].(map[string]interface{}); ok {
			if id, ok := data["id"].(string); ok {
				svID, _ := uuid.Parse(id)
				_, _ = testPool.Exec(context.Background(), `DELETE FROM survey_questions WHERE survey_id = $1`, svID)
				_, _ = testPool.Exec(context.Background(), `DELETE FROM surveys WHERE id = $1`, svID)
			}
		}
	})
}

// TestSurveyCreateHandler_PermAware_NoPermission
// DB権限なし（surveys:create=false）なら拒否される（403）
func TestSurveyCreateHandler_PermAware_NoPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "作成権限なし自治会", "HDL_SV_PA_C2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-pa-c2.test", "pass123", "association_admin")
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, false, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router, http.MethodPost, "/api/v1/surveys",
		map[string]interface{}{
			"title":      "権限なし作成",
			"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			"questions":  []interface{}{},
		},
		token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestSurveyDeleteHandler_PermAware_WithPermission
// DB権限あり（surveys:delete=true）なら削除できる（200）
func TestSurveyDeleteHandler_PermAware_WithPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "削除権限あり自治会", "HDL_SV_PA_D1")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-pa-d1.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "削除対象", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, true, true, true, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/surveys/%s", surveyID),
		nil, token,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestSurveyDeleteHandler_PermAware_NoPermission
// DB権限なし（surveys:delete=false）なら拒否される（403）
func TestSurveyDeleteHandler_PermAware_NoPermission(t *testing.T) {
	assocID := insertTestAssociation(t, "削除権限なし自治会", "HDL_SV_PA_D2")
	adminID := insertTestUser(t, &assocID, "管理者", "admin@hdl-sv-pa-d2.test", "pass123", "association_admin")
	surveyID := insertTestSurvey(t, assocID, adminID, "削除対象", time.Now().Add(24*time.Hour))
	token := makeTestToken(t, assocID.String(), adminID.String(), "association_admin")

	roleID := getRoleIDByName(t, "association_admin")
	featureID := getFeatureIDByName(t, "surveys")
	withPermission(t, roleID, featureID, true, true, true, false, "own_association")

	deps := newPermAwareTestDeps()
	rr := doRequest(t, deps.router,
		http.MethodDelete, fmt.Sprintf("/api/v1/surveys/%s", surveyID),
		nil, token,
	)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
