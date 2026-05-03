package association

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9_]{1,50}$`)

var ErrInvalidCode = errors.New("code must be uppercase alphanumeric (A-Z, 0-9, _)")

// Service は自治会管理のビジネスロジック
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List は全自治会を返す
func (s *Service) List(ctx context.Context) ([]*Association, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list associations: %w", err)
	}
	if items == nil {
		items = []*Association{}
	}
	return items, nil
}

// Get は自治会を1件取得する
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Association, error) {
	a, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get association: %w", err)
	}
	return a, nil
}

// Create は自治会を新規作成する
func (s *Service) Create(ctx context.Context, input CreateInput) (*Association, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(strings.ToUpper(input.Code))

	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !codePattern.MatchString(input.Code) {
		return nil, ErrInvalidCode
	}

	now := time.Now()
	a := &Association{
		ID:        uuid.New(),
		Name:      input.Name,
		Code:      input.Code,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, a); errors.Is(err, ErrCodeConflict) {
		return nil, ErrCodeConflict
	} else if err != nil {
		return nil, fmt.Errorf("create association: %w", err)
	}
	return a, nil
}

// Update は自治会情報を更新する
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*Association, error) {
	a, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find association: %w", err)
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(strings.ToUpper(input.Code))

	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !codePattern.MatchString(input.Code) {
		return nil, ErrInvalidCode
	}

	a.Name = input.Name
	a.Code = input.Code
	a.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, a); errors.Is(err, ErrCodeConflict) {
		return nil, ErrCodeConflict
	} else if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("update association: %w", err)
	}
	return a, nil
}

// Activate は自治会を有効化する
func (s *Service) Activate(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.SetActive(ctx, id, true); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("activate association: %w", err)
	}
	return nil
}

// Deactivate は自治会を無効化する（論理削除）
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.SetActive(ctx, id, false); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("deactivate association: %w", err)
	}
	return nil
}
