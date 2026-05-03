package association

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler は自治会管理の HTTP ハンドラ（system_admin 専用）
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/v1/associations
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "自治会一覧の取得に失敗しました")
		return
	}
	resp := make([]map[string]interface{}, 0, len(items))
	for _, a := range items {
		resp = append(resp, associationResponse(a))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"associations": resp})
}

// POST /api/v1/associations
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "自治会名は必須です")
		return
	}

	a, err := h.svc.Create(r.Context(), CreateInput{Name: req.Name, Code: req.Code})
	switch {
	case err == nil:
		respondJSON(w, http.StatusCreated, associationResponse(a))
	case errors.Is(err, ErrCodeConflict):
		respondError(w, http.StatusConflict, "CODE_CONFLICT", "この自治会コードは既に使用されています")
	case errors.Is(err, ErrInvalidCode):
		respondError(w, http.StatusBadRequest, "INVALID_CODE", "コードは半角英大文字・数字・アンダースコアのみ使用できます")
	default:
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}

// PUT /api/v1/associations/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	a, err := h.svc.Update(r.Context(), id, UpdateInput{Name: req.Name, Code: req.Code})
	switch {
	case err == nil:
		respondJSON(w, http.StatusOK, associationResponse(a))
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ASSOCIATION_NOT_FOUND", "自治会が見つかりません")
	case errors.Is(err, ErrCodeConflict):
		respondError(w, http.StatusConflict, "CODE_CONFLICT", "この自治会コードは既に使用されています")
	case errors.Is(err, ErrInvalidCode):
		respondError(w, http.StatusBadRequest, "INVALID_CODE", "コードは半角英大文字・数字・アンダースコアのみ使用できます")
	default:
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}

// DELETE /api/v1/associations/:id  → 論理削除
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	switch err := h.svc.Deactivate(r.Context(), id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "無効化しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ASSOCIATION_NOT_FOUND", "自治会が見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "削除に失敗しました")
	}
}

// PUT /api/v1/associations/:id/activate
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	switch err := h.svc.Activate(r.Context(), id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "有効化しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ASSOCIATION_NOT_FOUND", "自治会が見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "有効化に失敗しました")
	}
}

// PUT /api/v1/associations/:id/deactivate
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	switch err := h.svc.Deactivate(r.Context(), id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "無効化しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ASSOCIATION_NOT_FOUND", "自治会が見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "無効化に失敗しました")
	}
}

// ─── ヘルパー ────────────────────────────────────────────────────

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "ID の形式が不正です")
		return uuid.Nil, false
	}
	return id, true
}

// ─── レスポンス ──────────────────────────────────────────────────

func associationResponse(a *Association) map[string]interface{} {
	return map[string]interface{}{
		"id":         a.ID.String(),
		"name":       a.Name,
		"code":       a.Code,
		"is_active":  a.IsActive,
		"created_at": a.CreatedAt,
		"updated_at": a.UpdatedAt,
	}
}

// ─── JSON レスポンス共通 ─────────────────────────────────────────

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
