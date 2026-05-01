package middleware

import (
	"context"
	"net/http"
	"strings"

	"circular-board/internal/domain"
	"circular-board/internal/service"
)

type contextKey string

const claimsKey contextKey = "claims"

type AuthMiddleware struct {
	authSvc *service.AuthService
}

func NewAuthMiddleware(authSvc *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

// Authenticate はBearerトークンを検証し、クレームをコンテキストに設定する
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			respondUnauthorized(w)
			return
		}

		claims, err := m.authSvc.ValidateAccessToken(token)
		if err != nil {
			respondUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole は指定ロール以上のアクセスを要求するミドルウェアを返す
func (m *AuthMiddleware) RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				respondUnauthorized(w)
				return
			}

			if !service.RequireRole(claims, roles...) {
				respondForbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext はコンテキストからJWTクレームを取り出す
func ClaimsFromContext(ctx context.Context) *service.Claims {
	v := ctx.Value(claimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*service.Claims)
	return c
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"認証が必要です"}}`))
}

func respondForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"この操作を行う権限がありません"}}`))
}
