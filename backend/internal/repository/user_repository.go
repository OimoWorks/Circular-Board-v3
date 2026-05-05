package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"circular-board/internal/domain"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type UserWithAssociation struct {
	domain.User
	AssociationName *string
	AssociationCode *string
}

func (r *UserRepository) FindByEmailAndAssociationCode(ctx context.Context, email, associationCode string) (*UserWithAssociation, error) {
	query := `
		SELECT
			u.id, u.association_id, u.name, u.email, u.password_hash, u.role, u.is_active,
			u.created_at, u.updated_at,
			a.name AS association_name, a.code AS association_code
		FROM users u
		LEFT JOIN associations a ON u.association_id = a.id
		WHERE u.email = $1
		  AND (a.code = $2 OR (u.role = 'system_admin' AND $2 = ''))
		  AND u.is_active = true
		  AND (u.role = 'system_admin' OR a.is_active = true)`

	row := r.pool.QueryRow(ctx, query, email, associationCode)
	return scanUserWithAssociation(row)
}

func (r *UserRepository) FindByEmailForSystemAdmin(ctx context.Context, email string) (*UserWithAssociation, error) {
	query := `
		SELECT
			u.id, u.association_id, u.name, u.email, u.password_hash, u.role, u.is_active,
			u.created_at, u.updated_at,
			NULL AS association_name, NULL AS association_code
		FROM users u
		WHERE u.email = $1
		  AND u.role = 'system_admin'
		  AND u.is_active = true`

	row := r.pool.QueryRow(ctx, query, email)
	return scanUserWithAssociation(row)
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*UserWithAssociation, error) {
	query := `
		SELECT
			u.id, u.association_id, u.name, u.email, u.password_hash, u.role, u.is_active,
			u.created_at, u.updated_at,
			a.name AS association_name, a.code AS association_code
		FROM users u
		LEFT JOIN associations a ON u.association_id = a.id
		WHERE u.id = $1
		  AND u.is_active = true`

	row := r.pool.QueryRow(ctx, query, id)
	result, err := scanUserWithAssociation(row)
	if err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return result, nil
}

// FindActiveByEmail はメールアドレスでアクティブなユーザーを検索する（パスワードリセット用）
func (r *UserRepository) FindActiveByEmail(ctx context.Context, email string) (*UserWithAssociation, error) {
	query := `
		SELECT
			u.id, u.association_id, u.name, u.email, u.password_hash, u.role, u.is_active,
			u.created_at, u.updated_at,
			a.name AS association_name, a.code AS association_code
		FROM users u
		LEFT JOIN associations a ON u.association_id = a.id
		WHERE u.email = $1
		  AND u.is_active = true
		LIMIT 1`

	row := r.pool.QueryRow(ctx, query, email)
	return scanUserWithAssociation(row)
}

// UpdatePassword はパスワードハッシュを更新する（パスワードリセット用）
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUserWithAssociation(row pgx.Row) (*UserWithAssociation, error) {
	var u UserWithAssociation
	var roleStr string
	err := row.Scan(
		&u.ID, &u.AssociationID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt,
		&u.AssociationName, &u.AssociationCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.Role = domain.Role(roleStr)
	return &u, nil
}
