package home

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"circular-board/internal/domain"
	"circular-board/internal/middleware"
)

// Handler はTOP画面データ取得のHTTPハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetHomeData は GET /api/v1/home を処理する。
// system_admin は任意で ?association_id=<uuid> を指定できる（省略時は空データ）。
func (h *Handler) GetHomeData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		return
	}

	callerUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "ユーザー ID が不正です")
		return
	}

	callerRole := domain.Role(claims.Role)

	var callerAssocID *uuid.UUID

	if callerRole == domain.RoleSystemAdmin {
		// system_admin はクエリパラメータで自治会を指定できる
		if raw := r.URL.Query().Get("association_id"); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_PARAM", "association_id の形式が不正です")
				return
			}
			callerAssocID = &id
		}
	} else {
		// 一般ユーザー・association_admin はJWTの自治会IDを使用
		if claims.AssociationID != "" {
			id, err := uuid.Parse(claims.AssociationID)
			if err != nil {
				respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会 ID が不正です")
				return
			}
			callerAssocID = &id
		}
	}

	data, err := h.svc.GetHomeData(r.Context(), callerRole, callerAssocID, callerUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "データの取得に失敗しました")
		return
	}

	respondJSON(w, http.StatusOK, buildResponse(data))
}

// ─── レスポンス組み立て ──────────────────────────────────────────

func buildResponse(d *HomeData) map[string]interface{} {
	notices := make([]map[string]interface{}, 0, len(d.Notices))
	for _, n := range d.Notices {
		notices = append(notices, map[string]interface{}{
			"id":         n.ID.String(),
			"title":      n.Title,
			"is_pinned":  n.IsPinned,
			"is_read":    n.IsRead,
			"created_at": n.CreatedAt,
		})
	}

	files := make([]map[string]interface{}, 0, len(d.Files))
	for _, f := range d.Files {
		files = append(files, map[string]interface{}{
			"id":                f.ID.String(),
			"year":              f.Year,
			"month":             f.Month,
			"original_filename": f.OriginalFilename,
			"file_size":         f.FileSize,
			"mime_type":         f.MimeType,
			"created_at":        f.CreatedAt,
		})
	}

	surveys := make([]map[string]interface{}, 0, len(d.Surveys))
	for _, s := range d.Surveys {
		surveys = append(surveys, map[string]interface{}{
			"id":         s.ID.String(),
			"title":      s.Title,
			"expires_at": s.ExpiresAt,
		})
	}

	return map[string]interface{}{
		"notices":                notices,
		"files":                  files,
		"surveys":                surveys,
		"unread_notice_count":    d.UnreadNoticeCount,
		"unanswered_survey_count": d.UnansweredSurveyCount,
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
