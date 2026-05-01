package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"circular-board/internal/middleware"
	"circular-board/internal/repository"
	"circular-board/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssociationCode string `json:"association_code"`
		Email           string `json:"email"`
		Password        string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が正しくありません")
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "メールアドレスとパスワードは必須です")
		return
	}

	result, err := h.authSvc.Login(r.Context(), req.AssociationCode, req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "自治会コード・メールアドレス・パスワードをご確認ください")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ログインに失敗しました")
		return
	}

	respondJSON(w, http.StatusOK, buildTokenResponse(result))
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "トークンが不正です")
		return
	}

	userWithAssoc, err := h.authSvc.GetMe(r.Context(), userID)
	if errors.Is(err, service.ErrUserNotFound) {
		respondError(w, http.StatusNotFound, "USER_NOT_FOUND", "ユーザーが見つかりません")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ユーザー情報の取得に失敗しました")
		return
	}

	respondJSON(w, http.StatusOK, userResponse(userWithAssoc))
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "トークンが不正です")
		return
	}

	if err := h.authSvc.Logout(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ログアウトに失敗しました")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "ログアウトしました"})
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リフレッシュトークンが必要です")
		return
	}

	result, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if errors.Is(err, service.ErrInvalidToken) {
		respondError(w, http.StatusUnauthorized, "INVALID_TOKEN", "トークンが無効または期限切れです")
		return
	}
	if errors.Is(err, service.ErrUserNotFound) {
		respondError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "ユーザーが見つかりません")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "トークンの更新に失敗しました")
		return
	}

	respondJSON(w, http.StatusOK, buildTokenResponse(result))
}

func buildTokenResponse(result *service.LoginResult) map[string]interface{} {
	return map[string]interface{}{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"token_type":    "Bearer",
		"user":          userResponse(result.User),
	}
}

func userResponse(u *repository.UserWithAssociation) map[string]interface{} {
	assocID := interface{}(nil)
	if u.AssociationID != nil {
		assocID = u.AssociationID.String()
	}
	return map[string]interface{}{
		"id":               u.ID.String(),
		"name":             u.Name,
		"email":            u.Email,
		"role":             string(u.Role),
		"association_id":   assocID,
		"association_name": u.AssociationName,
		"is_active":        u.IsActive,
		"created_at":       u.CreatedAt,
	}
}
