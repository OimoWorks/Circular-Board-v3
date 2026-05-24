package permission

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service は権限管理のビジネスロジックを担う
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetMatrix は権限マトリクス（ロール・機能・権限）を返す。
// assocID が nil の場合はデフォルト設定を返す。
// assocID が指定された場合はその自治会の有効な権限設定（専用設定または
// デフォルト設定へのフォールバック）を返す。
func (s *Service) GetMatrix(ctx context.Context, assocID *uuid.UUID) (*PermissionMatrix, error) {
	roles, err := s.repo.GetRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	features, err := s.repo.GetFeatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("get features: %w", err)
	}
	perms, err := s.repo.GetAllPermissions(ctx, assocID)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}
	if roles == nil {
		roles = []*RoleModel{}
	}
	if features == nil {
		features = []*FeatureModel{}
	}
	if perms == nil {
		perms = []*RolePermission{}
	}
	return &PermissionMatrix{
		Roles:       roles,
		Features:    features,
		Permissions: perms,
	}, nil
}

// GetRoles はロール一覧を返す
func (s *Service) GetRoles(ctx context.Context) ([]*RoleModel, error) {
	roles, err := s.repo.GetRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	if roles == nil {
		return []*RoleModel{}, nil
	}
	return roles, nil
}

// GetFeatures は機能一覧を返す
func (s *Service) GetFeatures(ctx context.Context) ([]*FeatureModel, error) {
	features, err := s.repo.GetFeatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("get features: %w", err)
	}
	if features == nil {
		return []*FeatureModel{}, nil
	}
	return features, nil
}

// UpdatePermissions は権限を一括更新する。
// assocID が nil の場合はデフォルト設定を更新する。
// assocID が指定された場合はその自治会専用の設定を更新する。
func (s *Service) UpdatePermissions(ctx context.Context, operatorID uuid.UUID, assocID *uuid.UUID, inputs []UpdateInput) error {
	if err := s.repo.UpdatePermissions(ctx, operatorID, assocID, inputs); err != nil {
		return fmt.Errorf("update permissions: %w", err)
	}
	return nil
}

// EmergencyAppointment は緊急任命を実行する
func (s *Service) EmergencyAppointment(ctx context.Context, operatorID uuid.UUID, input EmergencyAppointmentInput) error {
	if err := s.repo.EmergencyAppointment(ctx, operatorID, input); err != nil {
		return fmt.Errorf("emergency appointment: %w", err)
	}
	return nil
}

// CheckPermission はロール名・機能名でアクセス可否を返す。
// assocID が nil の場合はデフォルト設定を参照する。
// assocID が指定された場合は自治会専用設定を優先し、なければデフォルト設定を参照する。
func (s *Service) CheckPermission(ctx context.Context, roleName, featureName string, assocID *uuid.UUID) (*PermissionCheck, error) {
	pc, err := s.repo.GetPermissionByRoleAndFeature(ctx, roleName, featureName, assocID)
	if err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}
	return pc, nil
}

// GetOperationLogs は操作ログ一覧を返す
func (s *Service) GetOperationLogs(ctx context.Context) ([]*OperationLog, error) {
	logs, err := s.repo.GetOperationLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get operation logs: %w", err)
	}
	if logs == nil {
		return []*OperationLog{}, nil
	}
	return logs, nil
}
