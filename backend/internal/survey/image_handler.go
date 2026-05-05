package survey

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"circular-board/internal/domain"
)

const maxImageBytes = 5 << 20 // 5 MB

var allowedImageMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
}

// UploadImage は POST /api/v1/surveys/:id/images
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, _, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	surveyID, ok := h.parseID(w, r)
	if !ok {
		return
	}

	// アンケートのテナントチェック
	sv, err := h.svc.repo.FindByID(r.Context(), surveyID, nil)
	if errors.Is(err, ErrNotFound) {
		respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "エラーが発生しました")
		return
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || sv.AssociationID != *callerAssocID {
			respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
			return
		}
	}

	// ボディサイズ制限
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+4096)
	if err := r.ParseMultipartForm(maxImageBytes); err != nil {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "画像サイズは5MB以内にしてください")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		respondError(w, http.StatusBadRequest, "MISSING_FILE", "画像ファイルを指定してください（フィールド名: image）")
		return
	}
	defer file.Close()

	// 全バイト読み込み（最大 maxImageBytes+1 で上限チェック）
	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイルの読み込みに失敗しました")
		return
	}
	if int64(len(data)) > maxImageBytes {
		respondError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "画像サイズは5MB以内にしてください")
		return
	}

	contentType := http.DetectContentType(data)
	ext, allowed := allowedImageMIME[contentType]
	if !allowed {
		respondError(w, http.StatusBadRequest, "UNSUPPORTED_TYPE", "jpg・png・gif のみアップロードできます")
		return
	}

	// sort_order（フォームオプション）
	sortOrder := 0
	if v := r.FormValue("sort_order"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			sortOrder = n
		}
	}

	// ディスクに保存
	storedName := uuid.New().String() + ext
	uploadDir := h.cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "/app/uploads"
	}
	dirPath := filepath.Join(uploadDir, "survey_images", sv.AssociationID.String(), surveyID.String())
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "保存先の作成に失敗しました")
		return
	}
	storagePath := filepath.Join(dirPath, storedName)
	if err := os.WriteFile(storagePath, data, 0o644); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイルの保存に失敗しました")
		return
	}

	img := &SurveyImage{
		ID:            uuid.New(),
		SurveyID:      surveyID,
		AssociationID: sv.AssociationID,
		Filename:      header.Filename,
		StoragePath:   storagePath,
		SortOrder:     sortOrder,
		CreatedAt:     time.Now(),
	}
	if err := h.svc.repo.CreateImage(r.Context(), img); err != nil {
		_ = os.Remove(storagePath)
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "画像の保存に失敗しました")
		return
	}
	respondJSON(w, http.StatusCreated, imageResponse(img))
}

// DeleteImage は DELETE /api/v1/surveys/:id/images/:image_id
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, _, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	surveyID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "image_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "image_id の形式が不正です")
		return
	}

	img, err := h.svc.repo.FindImage(r.Context(), imageID)
	if errors.Is(err, ErrImageNotFound) {
		respondError(w, http.StatusNotFound, "IMAGE_NOT_FOUND", "画像が見つかりません")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "エラーが発生しました")
		return
	}
	// survey_id の一致確認
	if img.SurveyID != surveyID {
		respondError(w, http.StatusNotFound, "IMAGE_NOT_FOUND", "画像が見つかりません")
		return
	}
	// テナントチェック
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || img.AssociationID != *callerAssocID {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "この画像を削除する権限がありません")
			return
		}
	}

	if err := h.svc.repo.DeleteImage(r.Context(), imageID); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "画像の削除に失敗しました")
		return
	}
	_ = os.Remove(img.StoragePath)
	respondJSON(w, http.StatusOK, map[string]string{"message": "削除しました"})
}

// GetImage は GET /api/v1/surveys/:id/images/:image_id
func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, _, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	surveyID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "image_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "image_id の形式が不正です")
		return
	}

	img, err := h.svc.repo.FindImage(r.Context(), imageID)
	if errors.Is(err, ErrImageNotFound) {
		respondError(w, http.StatusNotFound, "IMAGE_NOT_FOUND", "画像が見つかりません")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "エラーが発生しました")
		return
	}
	if img.SurveyID != surveyID {
		respondError(w, http.StatusNotFound, "IMAGE_NOT_FOUND", "画像が見つかりません")
		return
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || img.AssociationID != *callerAssocID {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "この画像を取得する権限がありません")
			return
		}
	}

	data, err := os.ReadFile(img.StoragePath)
	if err != nil {
		respondError(w, http.StatusNotFound, "FILE_NOT_FOUND", "ファイルが見つかりません")
		return
	}

	contentType := http.DetectContentType(data)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(data)
}

// imageResponse は画像レコードのレスポンスマップを返す
func imageResponse(img *SurveyImage) map[string]interface{} {
	return map[string]interface{}{
		"id":             img.ID.String(),
		"survey_id":      img.SurveyID.String(),
		"association_id": img.AssociationID.String(),
		"filename":       img.Filename,
		"sort_order":     img.SortOrder,
		"created_at":     img.CreatedAt,
	}
}
