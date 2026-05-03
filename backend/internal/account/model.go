package account

import (
	"time"

	"github.com/google/uuid"

	"circular-board/internal/domain"
)

// Account はアカウント管理のドメインモデル（users テーブルに対応）
type Account struct {
	ID            uuid.UUID
	AssociationID *uuid.UUID
	Name          string
	Email         string
	PasswordHash  string
	Role          domain.Role
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateInput はアカウント作成時の入力値
type CreateInput struct {
	Name          string
	Email         string
	Password      string
	Role          domain.Role
	AssociationID *uuid.UUID // system_admin が他自治会向けに指定する場合
}

// UpdateInput はアカウント更新時の入力値
type UpdateInput struct {
	Name     string
	Email    string
	Password string // 空文字 = 変更なし
	Role     domain.Role
}
