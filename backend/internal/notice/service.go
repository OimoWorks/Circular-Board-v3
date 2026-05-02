package notice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service はお知らせのビジネスロジック
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create はお知らせを作成する
func (s *Service) Create(ctx context.Context, associationID, createdBy uuid.UUID, input CreateInput) (*Notice, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	now := time.Now()
	n := &Notice{
		ID:            uuid.New(),
		AssociationID: associationID,
		Title:         input.Title,
		Body:          input.Body,
		IsPinned:      input.IsPinned,
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("create notice: %w", err)
	}
	return n, nil
}

// Delete はお知らせを論理削除する
func (s *Service) Delete(ctx context.Context, associationID, noticeID uuid.UUID) error {
	err := s.repo.SoftDelete(ctx, noticeID, associationID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete notice: %w", err)
	}
	return nil
}

// List はお知らせ一覧を返す（is_read付き）
func (s *Service) List(ctx context.Context, associationID, userID uuid.UUID) ([]*Notice, error) {
	items, err := s.repo.List(ctx, associationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list notices: %w", err)
	}
	if items == nil {
		items = []*Notice{}
	}
	return items, nil
}

// Get はお知らせ詳細を返す
func (s *Service) Get(ctx context.Context, associationID, noticeID uuid.UUID) (*Notice, error) {
	n, err := s.repo.FindByID(ctx, noticeID, associationID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notice: %w", err)
	}
	return n, nil
}

// MarkAsRead はお知らせを既読にする（テナントチェック込み）
func (s *Service) MarkAsRead(ctx context.Context, associationID, noticeID, userID uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, noticeID, associationID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("find notice: %w", err)
	}
	if err := s.repo.MarkAsRead(ctx, noticeID, userID); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

// UnreadCount は未読件数を返す
func (s *Service) UnreadCount(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, associationID, userID)
}
