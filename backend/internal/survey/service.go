package survey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"circular-board/internal/domain"
)

// Service はアンケート管理のビジネスロジック
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create はアンケートを作成する（association_admin 以上）
func (s *Service) Create(ctx context.Context, associationID, callerID uuid.UUID, input CreateInput) (*Survey, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(input.Questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}
	if input.ExpiresAt.IsZero() || !input.ExpiresAt.After(time.Now()) {
		return nil, fmt.Errorf("expires_at must be a future datetime")
	}
	for i, q := range input.Questions {
		if strings.TrimSpace(q.QuestionText) == "" {
			return nil, fmt.Errorf("question[%d]: text is required", i)
		}
		if q.QuestionType != "single" && q.QuestionType != "multiple" {
			return nil, fmt.Errorf("question[%d]: type must be 'single' or 'multiple'", i)
		}
		if len(q.Choices) < 2 {
			return nil, fmt.Errorf("question[%d]: at least 2 choices are required", i)
		}
	}

	now := time.Now()
	survey := &Survey{
		ID:            uuid.New(),
		AssociationID: associationID,
		Title:         input.Title,
		Description:   strings.TrimSpace(input.Description),
		ExpiresAt:     input.ExpiresAt,
		CreatedBy:     callerID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	for i, qi := range input.Questions {
		q := Question{
			ID:           uuid.New(),
			SurveyID:     survey.ID,
			QuestionText: strings.TrimSpace(qi.QuestionText),
			QuestionType: qi.QuestionType,
			SortOrder:    qi.SortOrder,
		}
		if q.SortOrder == 0 {
			q.SortOrder = i + 1
		}
		for j, ci := range qi.Choices {
			c := Choice{
				ID:         uuid.New(),
				QuestionID: q.ID,
				ChoiceText: strings.TrimSpace(ci.ChoiceText),
				SortOrder:  ci.SortOrder,
			}
			if c.SortOrder == 0 {
				c.SortOrder = j + 1
			}
			q.Choices = append(q.Choices, c)
		}
		survey.Questions = append(survey.Questions, q)
	}

	if err := s.repo.Create(ctx, survey); err != nil {
		return nil, fmt.Errorf("create survey: %w", err)
	}
	return survey, nil
}

// Delete はアンケートを論理削除する
func (s *Service) Delete(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID) error {
	survey, err := s.repo.FindByID(ctx, id, nil)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find survey: %w", err)
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || survey.AssociationID != *callerAssocID {
			return ErrNotFound
		}
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete survey: %w", err)
	}
	return nil
}

// List はアンケート一覧を返す
func (s *Service) List(ctx context.Context, associationID, userID uuid.UUID) ([]*Survey, error) {
	surveys, err := s.repo.List(ctx, associationID, &userID)
	if err != nil {
		return nil, fmt.Errorf("list surveys: %w", err)
	}
	if surveys == nil {
		surveys = []*Survey{}
	}
	return surveys, nil
}

// Get はアンケート詳細を取得する
func (s *Service) Get(ctx context.Context, associationID, userID, id uuid.UUID) (*Survey, error) {
	survey, err := s.repo.FindByID(ctx, id, &userID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get survey: %w", err)
	}
	if survey.AssociationID != associationID {
		return nil, ErrNotFound
	}
	return survey, nil
}

// Answer はアンケートに回答する
func (s *Service) Answer(ctx context.Context, associationID, userID, surveyID uuid.UUID, input AnswerInput) error {
	active, err := s.repo.CheckSurveyExpiry(ctx, surveyID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check expiry: %w", err)
	}
	if !active {
		return ErrExpired
	}

	// テナントチェック
	survey, err := s.repo.FindByID(ctx, surveyID, nil)
	if err != nil {
		return fmt.Errorf("find survey: %w", err)
	}
	if survey.AssociationID != associationID {
		return ErrNotFound
	}

	if err := s.repo.ValidateChoices(ctx, surveyID, input.Answers); err != nil {
		return fmt.Errorf("validate choices: %w", err)
	}

	now := time.Now()
	var answers []Answer
	for _, qa := range input.Answers {
		for _, cID := range qa.ChoiceIDs {
			answers = append(answers, Answer{
				ID:         uuid.New(),
				SurveyID:   surveyID,
				UserID:     userID,
				QuestionID: qa.QuestionID,
				ChoiceID:   cID,
				AnsweredAt: now,
			})
		}
	}

	if err := s.repo.SaveAnswers(ctx, surveyID, userID, answers); err != nil {
		return fmt.Errorf("save answers: %w", err)
	}
	return nil
}

// GetResults は集計結果を取得する（管理者用）
func (s *Service) GetResults(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID) (*SurveyResult, error) {
	survey, err := s.repo.FindByID(ctx, id, nil)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find survey: %w", err)
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || survey.AssociationID != *callerAssocID {
			return nil, ErrNotFound
		}
	}

	result, err := s.repo.GetResults(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get results: %w", err)
	}
	return result, nil
}

// CountUnanswered は未回答アンケート件数を返す
func (s *Service) CountUnanswered(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	return s.repo.CountUnanswered(ctx, associationID, userID)
}
