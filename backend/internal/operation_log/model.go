package operation_log

import (
	"time"

	"github.com/google/uuid"
)

// OperationLog は操作ログの1件を表す
type OperationLog struct {
	ID            uuid.UUID
	OperatorID    uuid.UUID
	OperationType string
	TargetType    string
	TargetID      *string
	BeforeValue   *string // JSON文字列
	AfterValue    *string // JSON文字列
	CreatedAt     time.Time
}

// SaveInput は操作ログ保存の入力
type SaveInput struct {
	OperatorID    uuid.UUID
	OperationType string
	TargetType    string
	TargetID      *string
	BeforeValue   *string
	AfterValue    *string
}
