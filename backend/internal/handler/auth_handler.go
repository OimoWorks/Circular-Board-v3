package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/config"
	"circular-board/internal/middleware"
	"circular-board/internal/repository"
	"circular-board/internal/service"
)

type AuthHandler struct {
	authSvc   *service.AuthService
	resetRepo *repository.PasswordResetRepository
	userRepo  *repository.UserRepository
	cfg       *config.Config
}

func NewAuthHandler(authSvc *service.AuthService, resetRepo *repository.PasswordResetRepository, userRepo *repository.UserRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authSvc:   authSvc,
		resetRepo: resetRepo,
		userRepo:  userRepo,
		cfg:       cfg,
	}
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

// POST /api/v1/auth/forgot-password
// ブルートフォース対策のため、ユーザーが存在しない場合も成功レスポンスを返す
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が正しくありません")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "メールアドレスは必須です")
		return
	}

	// メール送信は非同期にしてレスポンスを即返す（ユーザー存在確認を隠蔽）
	go func() {
		ctx := context.Background()

		// system_admin 優先でユーザーを検索
		user, err := h.userRepo.FindByEmailForSystemAdmin(ctx, req.Email)
		if errors.Is(err, repository.ErrNotFound) {
			// 通常ユーザーを検索（自治会コードなしで最初の一致を探す）
			user, err = h.userRepo.FindActiveByEmail(ctx, req.Email)
		}
		if err != nil {
			return // ユーザーが存在しない場合は何もしない
		}

		rawToken, err := generateResetToken()
		if err != nil {
			log.Printf("reset token generation error: %v", err)
			return
		}
		tokenHash := hashResetToken(rawToken)
		expiresAt := time.Now().Add(time.Hour)

		if err := h.resetRepo.Create(ctx, user.ID, tokenHash, expiresAt); err != nil {
			log.Printf("reset token save error: %v", err)
			return
		}

		resetURL := fmt.Sprintf("%s/reset-password?token=%s", h.cfg.AppBaseURL, rawToken)
		if err := sendResetEmail(h.cfg, req.Email, user.Name, resetURL); err != nil {
			log.Printf("reset email send error: %v", err)
		}
	}()

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "入力されたメールアドレスにパスワードリセット用のメールを送信しました（登録済みの場合）",
	})
}

// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が正しくありません")
		return
	}
	if req.Token == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "トークンとパスワードは必須です")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "WEAK_PASSWORD", "パスワードは8文字以上で入力してください")
		return
	}

	tokenHash := hashResetToken(req.Token)
	tokenRec, err := h.resetRepo.FindValidByTokenHash(r.Context(), tokenHash)
	if errors.Is(err, repository.ErrTokenExpiredOrUsed) {
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "トークンが無効または期限切れです")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "トークンの検証に失敗しました")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "パスワードの処理に失敗しました")
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), tokenRec.UserID, string(hash)); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "パスワードの更新に失敗しました")
		return
	}

	if err := h.resetRepo.MarkUsed(r.Context(), tokenRec.ID); err != nil {
		log.Printf("mark token used error: %v", err)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "パスワードを更新しました"})
}

// ─── SendGrid メール送信 ──────────────────────────────────────────

func sendResetEmail(cfg *config.Config, toEmail, toName, resetURL string) error {
	if cfg.SendGridAPIKey == "" {
		log.Printf("SENDGRID_API_KEY not set; reset URL: %s", resetURL)
		return nil
	}

	body := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{"to": []map[string]string{{"email": toEmail, "name": toName}}},
		},
		"from":    map[string]string{"email": cfg.SendGridFromEmail, "name": "回覧板システム"},
		"subject": "【回覧板】パスワードリセットのご案内",
		"content": []map[string]string{
			{
				"type":  "text/html",
				"value": buildResetEmailHTML(toName, resetURL),
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal email body: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://api.sendgrid.com/v3/mail/send", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}
	return nil
}

func buildResetEmailHTML(name, resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #2D6A4F;">パスワードリセットのご案内</h2>
  <p>%s 様</p>
  <p>以下のボタンをクリックしてパスワードを再設定してください。</p>
  <p>リンクの有効期限は <strong>1時間</strong> です。</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#2D6A4F;color:#fff;
     border-radius:8px;text-decoration:none;font-weight:bold;">パスワードを再設定する</a>
  <p style="margin-top:24px;font-size:12px;color:#666;">
    ボタンが機能しない場合は以下のURLをブラウザにコピーしてください：<br/>
    <a href="%s">%s</a>
  </p>
  <p style="font-size:12px;color:#666;">
    このメールに心当たりのない場合は無視してください。
  </p>
</body>
</html>`, name, resetURL, resetURL, resetURL)
}

// ─── トークン生成 ─────────────────────────────────────────────────

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashResetToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ─── レスポンス ──────────────────────────────────────────────────

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

