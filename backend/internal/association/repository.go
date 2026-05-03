package association

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("association not found")
	ErrCodeConflict = errors.New("association code already in use")
)

// Repository は associations テーブルの操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List は全自治会を返す
func (r *Repository) List(ctx context.Context) ([]*Association, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, code, is_active, created_at, updated_at
		FROM associations
		ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Association
	for rows.Next() {
		a, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// FindByID は ID で自治会を取得する
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Association, error) {
	return r.scan(r.pool.QueryRow(ctx, `
		SELECT id, name, code, is_active, created_at, updated_at
		FROM associations WHERE id = $1`, id))
}

// Create は自治会を挿入する
func (r *Repository) Create(ctx context.Context, a *Association) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO associations (id, name, code, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.Name, a.Code, a.IsActive, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrCodeConflict
		}
		return err
	}
	return nil
}

// Update は自治会情報を更新する
func (r *Repository) Update(ctx context.Context, a *Association) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE associations
		SET name=$2, code=$3, updated_at=$4
		WHERE id=$1`,
		a.ID, a.Name, a.Code, a.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrCodeConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive は自治会の有効/無効状態を切り替える
func (r *Repository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE associations SET is_active=$2, updated_at=$3 WHERE id=$1`,
		id, active, time.Now(),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) scan(row interface{ Scan(dest ...any) error }) (*Association, error) {
	var a Association
	err := row.Scan(&a.ID, &a.Name, &a.Code, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
