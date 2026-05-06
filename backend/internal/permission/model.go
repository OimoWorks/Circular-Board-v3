package permission

import (
	"time"

	"github.com/google/uuid"
)

// RoleModel はロール定義
type RoleModel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FeatureModel は機能定義
type FeatureModel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	SortOrder   int       `json:"sort_order"`
}

// RolePermission はロールと機能の権限マトリクス
type RolePermission struct {
	ID        uuid.UUID `json:"id"`
	RoleID    uuid.UUID `json:"role_id"`
	FeatureID uuid.UUID `json:"feature_id"`
	CanView   bool      `json:"can_view"`
	CanCreate bool      `json:"can_create"`
	CanEdit   bool      `json:"can_edit"`
	CanDelete bool      `json:"can_delete"`
	Scope     string    `json:"scope"` // "all" | "own_association"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PermissionMatrix は権限一覧取得のレスポンス形式
type PermissionMatrix struct {
	Roles    []*RoleModel      `json:"roles"`
	Features []*FeatureModel   `json:"features"`
	Permissions []*RolePermission `json:"permissions"`
}

// UpdateInput は権限一括更新の入力
type UpdateInput struct {
	RoleID    uuid.UUID `json:"role_id"`
	FeatureID uuid.UUID `json:"feature_id"`
	CanView   bool      `json:"can_view"`
	CanCreate bool      `json:"can_create"`
	CanEdit   bool      `json:"can_edit"`
	CanDelete bool      `json:"can_delete"`
	Scope     string    `json:"scope"`
}

// OperationLog は操作ログ
type OperationLog struct {
	ID            uuid.UUID  `json:"id"`
	OperatorID    uuid.UUID  `json:"operator_id"`
	OperationType string     `json:"operation_type"`
	TargetType    string     `json:"target_type"`
	TargetID      *string    `json:"target_id"`
	BeforeValue   *string    `json:"before_value"`
	AfterValue    *string    `json:"after_value"`
	CreatedAt     time.Time  `json:"created_at"`
}

// EmergencyAppointmentInput は緊急任命の入力
type EmergencyAppointmentInput struct {
	UserID  uuid.UUID `json:"user_id"`
	NewRole string    `json:"new_role"`
}

// PermissionCheck はミドルウェア用の権限チェック結果
type PermissionCheck struct {
	CanView   bool
	CanCreate bool
	CanEdit   bool
	CanDelete bool
	Scope     string
}
