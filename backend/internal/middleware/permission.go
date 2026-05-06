package middleware

import (
	"net/http"

	"circular-board/internal/domain"
	"circular-board/internal/permission"
)

// PermissionMiddleware はDBのrole_permissionsを参照してアクセス制御を行う
type PermissionMiddleware struct {
	permSvc *permission.Service
}

func NewPermissionMiddleware(svc *permission.Service) *PermissionMiddleware {
	return &PermissionMiddleware{permSvc: svc}
}

// RequireFeature は指定機能・操作の権限を DB から確認するミドルウェアを返す。
// system_admin は常に許可（DBに依存しない）。
// action: "view" | "create" | "edit" | "delete"
func (m *PermissionMiddleware) RequireFeature(featureName, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				respondUnauthorized(w)
				return
			}

			// system_admin は常に全権限あり（DBに依存しない）
			if domain.Role(claims.Role) == domain.RoleSystemAdmin {
				next.ServeHTTP(w, r)
				return
			}

			pc, err := m.permSvc.CheckPermission(r.Context(), claims.Role, featureName)
			if err != nil {
				// DBエラー時はアクセス拒否（安全側に倒す）
				respondForbidden(w)
				return
			}

			var allowed bool
			switch action {
			case "view":
				allowed = pc.CanView
			case "create":
				allowed = pc.CanCreate
			case "edit":
				allowed = pc.CanEdit
			case "delete":
				allowed = pc.CanDelete
			}

			if !allowed {
				respondForbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
