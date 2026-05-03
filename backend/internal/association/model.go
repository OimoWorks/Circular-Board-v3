package association

import (
	"time"

	"github.com/google/uuid"
)

// Association は自治会管理のドメインモデル
type Association struct {
	ID        uuid.UUID
	Name      string
	Code      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateInput は自治会作成時の入力値
type CreateInput struct {
	Name string
	Code string
}

// UpdateInput は自治会更新時の入力値
type UpdateInput struct {
	Name string
	Code string
}
