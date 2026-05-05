package survey

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"circular-board/internal/domain"
	"circular-board/internal/middleware"
)

// Handler はアンケートの HTTP ハンドラ
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/surveys
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}

	// system_admin は association_id 必須
	assocID, ok := h.resolveAssocID(w, r, callerRole, callerAssocID)
	if !ok {
		return
	}

	var req struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		ExpiresAt   string          `json:"expires_at"`
		Questions   []questionInput `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PARAM", "expires_at の形式は RFC3339 で指定してください")
		return
	}

	input := CreateInput{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		ExpiresAt:   expiresAt,
	}
	for _, qi := range req.Questions {
		q := QuestionInput{
			QuestionText: qi.QuestionText,
			QuestionType: qi.QuestionType,
			SortOrder:    qi.SortOrder,
		}
		for _, ci := range qi.Choices {
			q.Choices = append(q.Choices, ChoiceInput{
				ChoiceText: ci.ChoiceText,
				SortOrder:  ci.SortOrder,
			})
		}
		input.Questions = append(input.Questions, q)
	}

	s, err := h.svc.Create(r.Context(), assocID, callerUserID, input)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, surveyResponse(s))
}

// DELETE /api/v1/surveys/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, _, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	switch err := h.svc.Delete(r.Context(), callerRole, callerAssocID, id); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "削除しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "削除に失敗しました")
	}
}

// GET /api/v1/surveys
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}

	assocID, ok := h.resolveAssocID(w, r, callerRole, callerAssocID)
	if !ok {
		return
	}

	surveys, err := h.svc.List(r.Context(), assocID, callerUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "アンケート一覧の取得に失敗しました")
		return
	}

	items := make([]map[string]interface{}, 0, len(surveys))
	for _, s := range surveys {
		items = append(items, surveyListResponse(s))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"surveys": items})
}

// GET /api/v1/surveys/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	assocID, ok := h.resolveAssocID(w, r, callerRole, callerAssocID)
	if !ok {
		return
	}

	s, err := h.svc.Get(r.Context(), assocID, callerUserID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "アンケートの取得に失敗しました")
	default:
		respondJSON(w, http.StatusOK, surveyResponse(s))
	}
}

// POST /api/v1/surveys/:id/answer
func (h *Handler) Answer(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	assocID, ok := h.resolveAssocID(w, r, callerRole, callerAssocID)
	if !ok {
		return
	}

	var req struct {
		Answers []struct {
			QuestionID string   `json:"question_id"`
			ChoiceIDs  []string `json:"choice_ids"`
		} `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "リクエストの形式が不正です")
		return
	}

	var input AnswerInput
	for _, a := range req.Answers {
		qID, err := uuid.Parse(a.QuestionID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "question_id の形式が不正です")
			return
		}
		qa := QuestionAnswerInput{QuestionID: qID}
		for _, cStr := range a.ChoiceIDs {
			cID, err := uuid.Parse(cStr)
			if err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_PARAM", "choice_id の形式が不正です")
				return
			}
			qa.ChoiceIDs = append(qa.ChoiceIDs, cID)
		}
		input.Answers = append(input.Answers, qa)
	}

	switch err := h.svc.Answer(r.Context(), assocID, callerUserID, id, input); {
	case err == nil:
		respondJSON(w, http.StatusOK, map[string]string{"message": "回答しました"})
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
	case errors.Is(err, ErrExpired):
		respondError(w, http.StatusBadRequest, "SURVEY_EXPIRED", "このアンケートは期限切れです")
	default:
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}

// GET /api/v1/surveys/:id/results
func (h *Handler) Results(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, _, ok := h.mustCaller(w, r)
	if !ok {
		return
	}
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	result, err := h.svc.GetResults(r.Context(), callerRole, callerAssocID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		respondError(w, http.StatusNotFound, "SURVEY_NOT_FOUND", "アンケートが見つかりません")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "集計結果の取得に失敗しました")
	default:
		respondJSON(w, http.StatusOK, surveyResultResponse(result))
	}
}

// GET /api/v1/surveys/unanswered-count
func (h *Handler) UnansweredCount(w http.ResponseWriter, r *http.Request) {
	callerRole, callerAssocID, callerUserID, ok := h.mustCaller(w, r)
	if !ok {
		return
	}

	assocID, ok := h.resolveAssocID(w, r, callerRole, callerAssocID)
	if !ok {
		return
	}

	count, err := h.svc.CountUnanswered(r.Context(), assocID, callerUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "未回答件数の取得に失敗しました")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"unanswered_count": count})
}

// ─── ヘルパー ────────────────────────────────────────────────────

type questionInput struct {
	QuestionText string        `json:"question_text"`
	QuestionType string        `json:"question_type"`
	SortOrder    int           `json:"sort_order"`
	Choices      []choiceInput `json:"choices"`
}

type choiceInput struct {
	ChoiceText string `json:"choice_text"`
	SortOrder  int    `json:"sort_order"`
}

func (h *Handler) mustCaller(w http.ResponseWriter, r *http.Request) (role domain.Role, assocID *uuid.UUID, userID uuid.UUID, ok bool) {
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

// resolveAssocID は呼び出し元のロールに応じて自治会 ID を解決する
func (h *Handler) resolveAssocID(w http.ResponseWriter, r *http.Request, role domain.Role, callerAssocID *uuid.UUID) (uuid.UUID, bool) {
	if role == domain.RoleSystemAdmin {
		raw := r.URL.Query().Get("association_id")
		if raw == "" {
			// system_admin でも POST body に association_id を含む場合があるが、
			// ここでは一覧・詳細用にクエリパラメータのみを見る
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "association_id を指定してください")
			return uuid.Nil, false
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PARAM", "association_id の形式が不正です")
			return uuid.Nil, false
		}
		return id, true
	}
	if callerAssocID == nil {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "自治会に所属していません")
		return uuid.Nil, false
	}
	return *callerAssocID, true
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

func surveyListResponse(s *Survey) map[string]interface{} {
	return map[string]interface{}{
		"id":             s.ID.String(),
		"association_id": s.AssociationID.String(),
		"title":          s.Title,
		"description":    s.Description,
		"expires_at":     s.ExpiresAt,
		"is_answered":    s.IsAnswered,
		"is_expired":     s.ExpiresAt.Before(time.Now()),
		"created_by":     s.CreatedBy.String(),
		"created_at":     s.CreatedAt,
	}
}

func surveyResponse(s *Survey) map[string]interface{} {
	resp := surveyListResponse(s)
	questions := make([]map[string]interface{}, 0, len(s.Questions))
	for _, q := range s.Questions {
		choices := make([]map[string]interface{}, 0, len(q.Choices))
		for _, c := range q.Choices {
			choices = append(choices, map[string]interface{}{
				"id":          c.ID.String(),
				"choice_text": c.ChoiceText,
				"sort_order":  c.SortOrder,
			})
		}
		questions = append(questions, map[string]interface{}{
			"id":            q.ID.String(),
			"question_text": q.QuestionText,
			"question_type": q.QuestionType,
			"sort_order":    q.SortOrder,
			"choices":       choices,
		})
	}
	resp["questions"] = questions
	return resp
}

func surveyResultResponse(r *SurveyResult) map[string]interface{} {
	questions := make([]map[string]interface{}, 0, len(r.Questions))
	for _, q := range r.Questions {
		choices := make([]map[string]interface{}, 0, len(q.Choices))
		for _, c := range q.Choices {
			choices = append(choices, map[string]interface{}{
				"choice_id":   c.ChoiceID.String(),
				"choice_text": c.ChoiceText,
				"count":       c.Count,
				"percentage":  c.Percentage,
			})
		}
		questions = append(questions, map[string]interface{}{
			"question_id":   q.QuestionID.String(),
			"question_text": q.QuestionText,
			"question_type": q.QuestionType,
			"total_answers": q.TotalAnswers,
			"choices":       choices,
		})
	}
	return map[string]interface{}{
		"survey_id":      r.SurveyID.String(),
		"title":          r.Title,
		"total_answered": r.TotalAnswered,
		"questions":      questions,
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
