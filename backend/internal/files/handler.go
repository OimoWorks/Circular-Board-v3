package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"circular-board/internal/middleware"
)

const maxUploadBodyBytes = 11 << 20 // 11MB（10MB本体＋フォームフィールド分）

// Handler はファイル管理のHTTPハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/files
// アップロード（association_admin以上）
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	assocID, ok := h.mustAssociationID(w, r)
	if !ok {
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID)

	// リクエストボディサイズを制限
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "ファイルサイズが上限(10MB)を超えています")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ファイルが指定されていません")
		return
	}
	defer file.Close()

	// MIME タイプを取得（ブラウザが設定する値＋魔法バイトで検出）
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		mimeType = http.DetectContentType(buf[:n])
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイル処理に失敗しました")
			return
		}
	}

	year := parseIntForm(r, "year", time.Now().Year())
	month := parseIntForm(r, "month", int(time.Now().Month()))

	input := UploadInput{
		Reader:           file,
		OriginalFilename: filepath.Base(header.Filename),
		Size:             header.Size,
		MimeType:         mimeType,
		Year:             year,
		Month:            month,
	}

	result, err := h.svc.Upload(r.Context(), assocID, userID, input)
	switch {
	case errors.Is(err, ErrFileTooLarge):
		respondError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "ファイルサイズが上限(10MB)を超えています")
	case errors.Is(err, ErrUnsupportedMIME):
		respondError(w, http.StatusBadRequest, "UNSUPPORTED_FILE_TYPE", "対応ファイル形式はPDF・JPG・PNGです")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイルのアップロードに失敗しました")
	default:
		respondJSON(w, http.StatusCreated, fileResponse(result))
	}
}

// DELETE /api/v1/files/:id
// 削除（association_admin以上）
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	assocID, ok := h.mustAssociationID(w, r)
	if !ok {
		return
	}

	fileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "IDの形式が不正です")
		return
	}

	switch err := h.svc.Delete(r.Context(), assocID, fileID); {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "FILE_NOT_FOUND", "ファイルが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイルの削除に失敗しました")
	default:
		respondJSON(w, http.StatusOK, map[string]string{"message": "削除しました"})
	}
}

// GET /api/v1/files?year=2024&month=1
// 一覧取得（全ロール）
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	assocID, ok := h.mustAssociationID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	year := parseIntQuery(q, "year", 0)
	month := parseIntQuery(q, "month", 0)

	list, err := h.svc.List(r.Context(), assocID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイル一覧の取得に失敗しました")
		return
	}

	items := make([]map[string]interface{}, 0, len(list))
	for _, f := range list {
		items = append(items, fileResponse(f))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"files": items,
		"year":  year,
		"month": month,
	})
}

// GET /api/v1/files/:id/download
// ダウンロード（全ロール）
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	assocID, ok := h.mustAssociationID(w, r)
	if !ok {
		return
	}

	fileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "IDの形式が不正です")
		return
	}

	f, err := h.svc.GetFile(r.Context(), assocID, fileID)
	switch {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "FILE_NOT_FOUND", "ファイルが見つかりません")
		return
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ファイルの取得に失敗しました")
		return
	}

	w.Header().Set("Content-Type", f.MimeType)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(f.OriginalFilename)))
	w.Header().Set("Content-Length", strconv.FormatInt(f.FileSize, 10))
	http.ServeFile(w, r, f.StoragePath)
}

// GET /api/v1/files/years
// 利用可能な年一覧（全ロール）
func (h *Handler) AvailableYears(w http.ResponseWriter, r *http.Request) {
	assocID, ok := h.mustAssociationID(w, r)
	if !ok {
		return
	}

	years, err := h.svc.AvailableYears(r.Context(), assocID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "年一覧の取得に失敗しました")
		return
	}
	if years == nil {
		years = []int{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"years": years})
}

// mustAssociationID はコンテキストからassociation_idを取得し、なければエラーを返す
// system_admin（association_idなし）は現時点ではファイルAPIを直接利用不可
func (h *Handler) mustAssociationID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AssociationID == "" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会に所属していません")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.AssociationID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会IDが不正です")
		return uuid.Nil, false
	}
	return id, true
}

// ─── レスポンス構築 ─────────────────────────────────────────────

func fileResponse(f *File) map[string]interface{} {
	return map[string]interface{}{
		"id":                f.ID.String(),
		"association_id":    f.AssociationID.String(),
		"year":              f.Year,
		"month":             f.Month,
		"original_filename": f.OriginalFilename,
		"file_size":         f.FileSize,
		"mime_type":         f.MimeType,
		"uploaded_by":       f.UploadedBy.String(),
		"created_at":        f.CreatedAt,
	}
}

// ─── レスポンス共通処理（handler パッケージと同形式） ────────────

type successResp struct {
	Data interface{} `json:"data"`
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errResp struct {
	Error errDetail `json:"error"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successResp{Data: data})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errResp{
		Error: errDetail{Code: code, Message: message},
	})
}

// ─── フォーム・クエリ値パーサー ──────────────────────────────────

func parseIntForm(r *http.Request, key string, defaultVal int) int {
	if v := r.FormValue(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func parseIntQuery(q url.Values, key string, defaultVal int) int {
	if v := q.Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
