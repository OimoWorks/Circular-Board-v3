package notice

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"circular-board/internal/middleware"
)

// Handler はお知らせのHTTPハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/notices
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	assocID, userID, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	var req struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		IsPinned bool   `json:"is_pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "タイトルは必須です")
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "本文は必須です")
		return
	}

	n, err := h.svc.Create(r.Context(), assocID, userID, CreateInput{
		Title:    req.Title,
		Body:     req.Body,
		IsPinned: req.IsPinned,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "お知らせの作成に失敗しました")
		return
	}
	respondJSON(w, http.StatusCreated, noticeResponse(n))
}

// DELETE /api/v1/notices/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	assocID, _, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	noticeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "IDの形式が不正です")
		return
	}

	switch err := h.svc.Delete(r.Context(), assocID, noticeID); {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "NOTICE_NOT_FOUND", "お知らせが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "お知らせの削除に失敗しました")
	default:
		respondJSON(w, http.StatusOK, map[string]string{"message": "削除しました"})
	}
}

// GET /api/v1/notices
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	assocID, userID, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), assocID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "お知らせ一覧の取得に失敗しました")
		return
	}

	resp := make([]map[string]interface{}, 0, len(items))
	for _, n := range items {
		resp = append(resp, noticeResponse(n))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"notices": resp})
}

// GET /api/v1/notices/unread-count
// NOTE: このルートは /{id} より先に登録すること（chiの静的ルート優先）
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	assocID, userID, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	count, err := h.svc.UnreadCount(r.Context(), assocID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "未読件数の取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"unread_count": count})
}

// GET /api/v1/notices/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	assocID, userID, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	noticeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "IDの形式が不正です")
		return
	}

	n, err := h.svc.Get(r.Context(), assocID, noticeID)
	switch {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "NOTICE_NOT_FOUND", "お知らせが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "お知らせの取得に失敗しました")
	default:
		// 閲覧時に自動で既読にする（エラーは無視）
		_ = h.svc.MarkAsRead(r.Context(), assocID, noticeID, userID)
		respondJSON(w, http.StatusOK, noticeResponse(n))
	}
}

// POST /api/v1/notices/:id/read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	assocID, userID, ok := h.mustIDs(w, r)
	if !ok {
		return
	}

	noticeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "IDの形式が不正です")
		return
	}

	switch err := h.svc.MarkAsRead(r.Context(), assocID, noticeID, userID); {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "NOTICE_NOT_FOUND", "お知らせが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "既読処理に失敗しました")
	default:
		respondJSON(w, http.StatusOK, map[string]string{"message": "既読にしました"})
	}
}

// mustIDs はJWTクレームからassociation_idとuser_idを取得する
func (h *Handler) mustIDs(w http.ResponseWriter, r *http.Request) (assocID, userID uuid.UUID, ok bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AssociationID == "" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会に所属していません")
		return uuid.Nil, uuid.Nil, false
	}
	aID, err := uuid.Parse(claims.AssociationID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会IDが不正です")
		return uuid.Nil, uuid.Nil, false
	}
	uID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "ユーザーIDが不正です")
		return uuid.Nil, uuid.Nil, false
	}
	return aID, uID, true
}

// ─── レスポンス構築 ─────────────────────────────────────────────

func noticeResponse(n *Notice) map[string]interface{} {
	return map[string]interface{}{
		"id":             n.ID.String(),
		"association_id": n.AssociationID.String(),
		"title":          n.Title,
		"body":           n.Body,
		"is_pinned":      n.IsPinned,
		"is_read":        n.IsRead,
		"created_by":     n.CreatedBy.String(),
		"created_at":     n.CreatedAt,
		"updated_at":     n.UpdatedAt,
	}
}

// ─── JSON レスポンス共通（filesパッケージと同形式） ───────────────

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
