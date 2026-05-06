package permission

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"circular-board/internal/domain"
	"circular-board/internal/service"
)

// Handler は権限管理のHTTPハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/v1/permissions
// 権限マトリクス（ロール・機能・権限）一覧取得（system_adminのみ）
func (h *Handler) GetMatrix(w http.ResponseWriter, r *http.Request) {
	matrix, err := h.svc.GetMatrix(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "権限情報の取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, matrix)
}

// PUT /api/v1/permissions
// 権限一括更新（system_adminのみ）
func (h *Handler) UpdatePermissions(w http.ResponseWriter, r *http.Request) {
	claims := service.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return
	}
	operatorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "ユーザーIDが不正です")
		return
	}

	var req struct {
		Permissions []struct {
			RoleID    string `json:"role_id"`
			FeatureID string `json:"feature_id"`
			CanView   bool   `json:"can_view"`
			CanCreate bool   `json:"can_create"`
			CanEdit   bool   `json:"can_edit"`
			CanDelete bool   `json:"can_delete"`
			Scope     string `json:"scope"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	inputs := make([]UpdateInput, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		roleID, err := uuid.Parse(p.RoleID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "role_idの形式が不正です")
			return
		}
		featureID, err := uuid.Parse(p.FeatureID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "feature_idの形式が不正です")
			return
		}
		scope := p.Scope
		if scope != "all" && scope != "own_association" {
			scope = "own_association"
		}
		inputs = append(inputs, UpdateInput{
			RoleID:    roleID,
			FeatureID: featureID,
			CanView:   p.CanView,
			CanCreate: p.CanCreate,
			CanEdit:   p.CanEdit,
			CanDelete: p.CanDelete,
			Scope:     scope,
		})
	}

	if err := h.svc.UpdatePermissions(r.Context(), operatorID, inputs); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "権限の更新に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "権限を更新しました"})
}

// GET /api/v1/permissions/roles
// ロール一覧取得（system_adminのみ）
func (h *Handler) GetRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.GetRoles(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ロール一覧の取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"roles": roles})
}

// GET /api/v1/permissions/features
// 機能一覧取得（system_adminのみ）
func (h *Handler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	features, err := h.svc.GetFeatures(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "機能一覧の取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"features": features})
}

// POST /api/v1/permissions/emergency-appointment
// 緊急任命（system_adminのみ）
func (h *Handler) EmergencyAppointment(w http.ResponseWriter, r *http.Request) {
	claims := service.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return
	}
	operatorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "ユーザーIDが不正です")
		return
	}

	var req struct {
		UserID  string `json:"user_id"`
		NewRole string `json:"new_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PARAM", "user_idの形式が不正です")
		return
	}

	role := domain.Role(req.NewRole)
	if !role.IsValid() {
		respondError(w, http.StatusBadRequest, "INVALID_PARAM", "不正なロールです")
		return
	}

	input := EmergencyAppointmentInput{
		UserID:  userID,
		NewRole: req.NewRole,
	}
	if err := h.svc.EmergencyAppointment(r.Context(), operatorID, input); err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "緊急任命に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "緊急任命を実行しました"})
}

// GET /api/v1/permissions/logs
// 操作ログ一覧取得（system_adminのみ）
func (h *Handler) GetOperationLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.svc.GetOperationLogs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "操作ログの取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"logs": logs})
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
