package account

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"circular-board/internal/domain"
	"circular-board/internal/middleware"
)

// Handler はアカウント管理の HTTP ハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/v1/accounts[?association_id=xxx]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}

	// system_admin は association_id クエリで絞り込み可（省略時は全件）
	var filterAssocID *uuid.UUID
	if callerRole == domain.RoleSystemAdmin {
		if raw := r.URL.Query().Get("association_id"); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_PARAM", "association_id の形式が不正です")
				return
			}
			filterAssocID = &id
		}
		// nil = 全件
	} else {
		filterAssocID = callerAssocID
	}

	accounts, err := h.svc.List(r.Context(), callerRole, filterAssocID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "アカウント一覧の取得に失敗しました")
		return
	}

	items := make([]map[string]interface{}, 0, len(accounts))
	for _, a := range accounts {
		items = append(items, accountResponse(a))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"accounts": items})
}

// POST /api/v1/accounts
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}

	var req struct {
		Name          string  `json:"name"`
		Email         string  `json:"email"`
		Password      string  `json:"password"`
		Role          string  `json:"role"`
		AssociationID *string `json:"association_id"` // system_admin 専用
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	input := CreateInput{
		Name:     strings.TrimSpace(req.Name),
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
		Role:     domain.Role(req.Role),
	}
	if req.AssociationID != nil && *req.AssociationID != "" {
		id, err := uuid.Parse(*req.AssociationID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "association_id の形式が不正です")
			return
		}
		input.AssociationID = &id
	}

	a, err := h.svc.Create(r.Context(), callerRole, callerAssocID, input)
	switch {
	case err == nil:
		respondJSON(w, http.StatusCreated, accountResponse(a))
	case errors.Is(err, ErrEmailConflict):
		respondError(w, http.StatusConflict, "EMAIL_CONFLICT", "このメールアドレスは既に使用されています")
	case errors.Is(err, ErrWeakPassword):
		respondError(w, http.StatusBadRequest, "WEAK_PASSWORD", "パスワードは8文字以上で入力してください")
	case errors.Is(err, ErrInvalidRole):
		respondError(w, http.StatusBadRequest, "INVALID_ROLE", "ロールが不正です")
	case errors.Is(err, ErrForbidden):
		respondError(w, http.StatusForbidden, "FORBIDDEN", "この操作を行う権限がありません")
	default:
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}

// PUT /api/v1/accounts/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	input := UpdateInput{
		Name:     strings.TrimSpace(req.Name),
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
		Role:     domain.Role(req.Role),
	}

	a, err := h.svc.Update(r.Context(), callerRole, callerAssocID, id, input)
	switch {
	case err == nil:
		respondJSON(w, http.StatusOK, accountResponse(a))
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "アカウントが見つかりません")
	case errors.Is(err, ErrEmailConflict):
		respondError(w, http.StatusConflict, "EMAIL_CONFLICT", "このメールアドレスは既に使用されています")
	case errors.Is(err, ErrWeakPassword):
		respondError(w, http.StatusBadRequest, "WEAK_PASSWORD", "パスワードは8文字以上で入力してください")
	case errors.Is(err, ErrInvalidRole):
		respondError(w, http.StatusBadRequest, "INVALID_ROLE", "ロールが不正です")
	default:
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}

// DELETE /api/v1/accounts/:id  → 論理削除（is_active=false）
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCallerFull(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	switch err := h.svc.Deactivate(r.Context(), callerRole, callerAssocID, callerUserID, id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "無効化しました"})
	case errors.Is(err, ErrSelfDeactivation):
		respondError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", "自分自身を削除することはできません")
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "アカウントが見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "削除に失敗しました")
	}
}

// PUT /api/v1/accounts/:id/activate
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	switch err := h.svc.Activate(r.Context(), callerRole, callerAssocID, id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "有効化しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "アカウントが見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "有効化に失敗しました")
	}
}

// PUT /api/v1/accounts/:id/deactivate
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCallerFull(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	switch err := h.svc.Deactivate(r.Context(), callerRole, callerAssocID, callerUserID, id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "無効化しました"})
	case errors.Is(err, ErrSelfDeactivation):
		respondError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", "自分自身を無効化することはできません")
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "アカウントが見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "無効化に失敗しました")
	}
}

// ─── ヘルパー ────────────────────────────────────────────────────

// mustCaller は JWT クレームからロールと自治会 ID を取り出す。
// system_admin は association_id が nil になる。
func (h *Handler) mustCaller(w http.ResponseWriter, r *http.Request) (role domain.Role, assocID *uuid.UUID, ok bool) {
	role, assocID, _, ok = h.mustCallerFull(w, r)
	return
}

// mustCallerFull はロール・自治会ID・ユーザーIDをすべて取り出す。
func (h *Handler) mustCallerFull(w http.ResponseWriter, r *http.Request) (role domain.Role, assocID *uuid.UUID, userID uuid.UUID, ok bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return "", nil, uuid.Nil, false
	}
	uID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "ユーザー ID が不正です")
		return "", nil, uuid.Nil, false
	}
	if claims.AssociationID != "" {
		id, err := uuid.Parse(claims.AssociationID)
		if err != nil {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会 ID が不正です")
			return "", nil, uuid.Nil, false
		}
		assocID = &id
	}
	return domain.Role(claims.Role), assocID, uID, true
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "ID の形式が不正です")
		return uuid.Nil, false
	}
	return id, true
}

// ─── レスポンス ──────────────────────────────────────────────────

func accountResponse(a *Account) map[string]interface{} {
	assocID := ""
	if a.AssociationID != nil {
		assocID = a.AssociationID.String()
	}
	return map[string]interface{}{
		"id":             a.ID.String(),
		"association_id": assocID,
		"name":           a.Name,
		"email":          a.Email,
		"role":           string(a.Role),
		"is_active":      a.IsActive,
		"created_at":     a.CreatedAt,
		"updated_at":     a.UpdatedAt,
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
