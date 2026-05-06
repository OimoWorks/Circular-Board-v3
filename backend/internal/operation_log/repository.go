package operation_log

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository は操作ログのDB操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Save は操作ログを保存する
func (r *Repository) Save(ctx context.Context, input SaveInput) (*OperationLog, error) {
	log := &OperationLog{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO operation_logs
			(operator_id, operation_type, target_type, target_id, before_value, after_value)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, operator_id, operation_type, target_type, target_id,
		          before_value::text, after_value::text, created_at`,
		input.OperatorID, input.OperationType, input.TargetType,
		input.TargetID, input.BeforeValue, input.AfterValue,
	).Scan(
		&log.ID, &log.OperatorID, &log.OperationType, &log.TargetType,
		&log.TargetID, &log.BeforeValue, &log.AfterValue, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save operation log: %w", err)
	}
	return log, nil
}

// List は操作ログを新しい順に最大50件取得する
func (r *Repository) List(ctx context.Context) ([]*OperationLog, error) {
	return r.query(ctx, `
		SELECT id, operator_id, operation_type, target_type, target_id,
		       before_value::text, after_value::text, created_at
		FROM operation_logs
		ORDER BY created_at DESC
		LIMIT 50`)
}

// ListByOperationType は操作種別で絞り込む
func (r *Repository) ListByOperationType(ctx context.Context, operationType string) ([]*OperationLog, error) {
	return r.query(ctx, `
		SELECT id, operator_id, operation_type, target_type, target_id,
		       before_value::text, after_value::text, created_at
		FROM operation_logs
		WHERE operation_type = $1
		ORDER BY created_at DESC
		LIMIT 50`,
		operationType,
	)
}

// ListByOperatorID は操作者IDで絞り込む
func (r *Repository) ListByOperatorID(ctx context.Context, operatorID uuid.UUID) ([]*OperationLog, error) {
	return r.query(ctx, `
		SELECT id, operator_id, operation_type, target_type, target_id,
		       before_value::text, after_value::text, created_at
		FROM operation_logs
		WHERE operator_id = $1
		ORDER BY created_at DESC
		LIMIT 50`,
		operatorID,
	)
}

func (r *Repository) query(ctx context.Context, sql string, args ...interface{}) ([]*OperationLog, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*OperationLog
	for rows.Next() {
		ol := &OperationLog{}
		if err := rows.Scan(
			&ol.ID, &ol.OperatorID, &ol.OperationType, &ol.TargetType,
			&ol.TargetID, &ol.BeforeValue, &ol.AfterValue, &ol.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, ol)
	}
	if logs == nil {
		logs = []*OperationLog{}
	}
	return logs, rows.Err()
}
