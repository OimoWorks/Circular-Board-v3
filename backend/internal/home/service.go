package home

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"circular-board/internal/domain"
)

// Service はTOP画面用のビジネスロジックを担う
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetHomeData はTOP画面に表示する集約データを返す。
// callerAssocID が nil（system_admin かつ association_id 未指定）のときは
// 空データを返す。
func (s *Service) GetHomeData(
	ctx context.Context,
	callerRole domain.Role,
	callerAssocID *uuid.UUID,
	callerUserID uuid.UUID,
) (*HomeData, error) {
	// association_id が解決できない場合は空データを返す
	if callerAssocID == nil {
		return &HomeData{
			Notices: []HomeNotice{},
			Files:   []HomeFile{},
			Surveys: []HomeSurvey{},
		}, nil
	}

	assocID := *callerAssocID

	notices, err := s.repo.ListRecentNotices(ctx, assocID, callerUserID, defaultLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent notices: %w", err)
	}

	files, err := s.repo.ListRecentFiles(ctx, assocID, defaultLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent files: %w", err)
	}

	surveys, err := s.repo.ListUnansweredSurveys(ctx, assocID, callerUserID, defaultLimit)
	if err != nil {
		return nil, fmt.Errorf("list unanswered surveys: %w", err)
	}

	unreadCount, err := s.repo.CountUnreadNotices(ctx, assocID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("count unread notices: %w", err)
	}

	unansweredCount, err := s.repo.CountUnansweredSurveys(ctx, assocID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("count unanswered surveys: %w", err)
	}

	return &HomeData{
		Notices:               notices,
		Files:                 files,
		Surveys:               surveys,
		UnreadNoticeCount:     unreadCount,
		UnansweredSurveyCount: unansweredCount,
	}, nil
}
