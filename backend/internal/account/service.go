package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/domain"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRole    = errors.New("invalid role")
	ErrWeakPassword   = errors.New("password too short")
)

// Service はアカウント管理のビジネスロジック
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List は自治会内のアカウント一覧を返す。
// system_admin は associationID=nil で全件取得可能。
func (s *Service) List(ctx context.Context, callerRole domain.Role, associationID *uuid.UUID) ([]*Account, error) {
	if callerRole != domain.RoleSystemAdmin && associationID == nil {
		return nil, ErrForbidden
	}
	accounts, err := s.repo.List(ctx, associationID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	if accounts == nil {
		accounts = []*Account{}
	}
	return accounts, nil
}

// Get はアカウントを1件取得する（テナントチェック付き）。
func (s *Service) Get(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID) (*Account, error) {
	a, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || a.AssociationID == nil || *callerAssocID != *a.AssociationID {
			return nil, ErrNotFound // テナント外には 404 を返す
		}
	}
	return a, nil
}

// Create はアカウントを新規作成する。
func (s *Service) Create(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, input CreateInput) (*Account, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)

	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if len(input.Password) < 8 {
		return nil, ErrWeakPassword
	}
	if !input.Role.IsValid() {
		return nil, ErrInvalidRole
	}

	// system_admin 以外は自身の自治会にしか作成できない
	assocID := input.AssociationID
	if callerRole != domain.RoleSystemAdmin {
		assocID = callerAssocID
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	a := &Account{
		ID:            uuid.New(),
		AssociationID: assocID,
		Name:          input.Name,
		Email:         input.Email,
		PasswordHash:  string(hash),
		Role:          input.Role,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, a); errors.Is(err, ErrEmailConflict) {
		return nil, ErrEmailConflict
	} else if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return a, nil
}

// Update はアカウント情報を更新する。
func (s *Service) Update(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID, input UpdateInput) (*Account, error) {
	a, err := s.Get(ctx, callerRole, callerAssocID, id)
	if err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if !input.Role.IsValid() {
		return nil, ErrInvalidRole
	}

	a.Name = input.Name
	a.Email = input.Email
	a.Role = input.Role
	a.UpdatedAt = time.Now()

	// パスワードは入力がある場合のみ変更
	if input.Password != "" {
		if len(input.Password) < 8 {
			return nil, ErrWeakPassword
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		a.PasswordHash = string(hash)
	}

	if err := s.repo.Update(ctx, a); errors.Is(err, ErrEmailConflict) {
		return nil, ErrEmailConflict
	} else if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	return a, nil
}

// Deactivate はアカウントを無効化する（論理削除）。
func (s *Service) Deactivate(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID) error {
	if _, err := s.Get(ctx, callerRole, callerAssocID, id); err != nil {
		return err
	}
	return s.setActive(ctx, id, false)
}

// Activate はアカウントを有効化する。
func (s *Service) Activate(ctx context.Context, callerRole domain.Role, callerAssocID *uuid.UUID, id uuid.UUID) error {
	// 無効アカウントも操作できるよう直接 FindByID を呼ぶ
	a, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}
	if callerRole != domain.RoleSystemAdmin {
		if callerAssocID == nil || a.AssociationID == nil || *callerAssocID != *a.AssociationID {
			return ErrNotFound
		}
	}
	return s.setActive(ctx, id, true)
}

func (s *Service) setActive(ctx context.Context, id uuid.UUID, active bool) error {
	if err := s.repo.SetActive(ctx, id, active); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	return nil
}
