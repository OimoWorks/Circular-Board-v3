package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"circular-board/internal/domain"
)

var (
	ErrNotFound      = errors.New("account not found")
	ErrEmailConflict = errors.New("email already in use")
)

// Repository は users テーブルの管理操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List は自治会内のアカウント一覧を返す。
// associationID が nil の場合は全自治会を対象にする（system_admin 用）。
func (r *Repository) List(ctx context.Context, associationID *uuid.UUID) ([]*Account, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if associationID != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT u.id, u.association_id, u.name, u.email, u.password_hash, u.role,
			       u.is_active, u.created_at, u.updated_at, a.name AS association_name
			FROM users u
			LEFT JOIN associations a ON u.association_id = a.id
			WHERE u.association_id = $1
			ORDER BY u.name ASC`, associationID)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT u.id, u.association_id, u.name, u.email, u.password_hash, u.role,
			       u.is_active, u.created_at, u.updated_at, a.name AS association_name
			FROM users u
			LEFT JOIN associations a ON u.association_id = a.id
			ORDER BY u.name ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		a, err := r.scanWithAssoc(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// FindByID は ID でアカウントを取得する
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	return r.scanWithAssoc(r.pool.QueryRow(ctx, `
		SELECT u.id, u.association_id, u.name, u.email, u.password_hash, u.role,
		       u.is_active, u.created_at, u.updated_at, a.name AS association_name
		FROM users u
		LEFT JOIN associations a ON u.association_id = a.id
		WHERE u.id = $1`, id))
}

// Create はアカウントを挿入する
func (r *Repository) Create(ctx context.Context, a *Account) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users
			(id, association_id, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.AssociationID, a.Name, a.Email, a.PasswordHash,
		string(a.Role), a.IsActive, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return err
	}
	return nil
}

// Update はアカウントの基本情報を更新する
func (r *Repository) Update(ctx context.Context, a *Account) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET name=$2, email=$3, password_hash=$4, role=$5, updated_at=$6
		WHERE id=$1`,
		a.ID, a.Name, a.Email, a.PasswordHash, string(a.Role), a.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive はアカウントの有効/無効状態を切り替える
func (r *Repository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active=$2, updated_at=$3 WHERE id=$1`,
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

// scanWithAssoc は association_name を含む共通行スキャン処理
func (r *Repository) scanWithAssoc(row interface{ Scan(dest ...any) error }) (*Account, error) {
	var a Account
	var roleStr string
	err := row.Scan(
		&a.ID, &a.AssociationID, &a.Name, &a.Email, &a.PasswordHash,
		&roleStr, &a.IsActive, &a.CreatedAt, &a.UpdatedAt, &a.AssociationName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.Role = domain.Role(roleStr)
	return &a, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
