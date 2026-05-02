package notice

import (
	"time"

	"github.com/google/uuid"
)

// Notice はお知らせのドメインモデル
type Notice struct {
	ID            uuid.UUID
	AssociationID uuid.UUID
	Title         string
	Body          string
	IsPinned      bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
	IsRead        bool // 一覧取得時にユーザーごとに付与
}

// CreateInput はお知らせ作成時の入力値
type CreateInput struct {
	Title    string
	Body     string
	IsPinned bool
}
