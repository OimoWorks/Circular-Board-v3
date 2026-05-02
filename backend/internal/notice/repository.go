package notice

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notice not found")

// Repository はお知らせのDB操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create はお知らせを挿入する
func (r *Repository) Create(ctx context.Context, n *Notice) error {
	const q = `
		INSERT INTO notices
			(id, association_id, title, body, is_pinned, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, q,
		n.ID, n.AssociationID, n.Title, n.Body, n.IsPinned,
		n.CreatedBy, n.CreatedAt, n.UpdatedAt,
	)
	return err
}

// FindByID はIDと自治会IDでお知らせを取得する（テナント境界の強制）
func (r *Repository) FindByID(ctx context.Context, id, associationID uuid.UUID) (*Notice, error) {
	const q = `
		SELECT id, association_id, title, body, is_pinned, created_by,
		       created_at, updated_at, deleted_at
		FROM notices
		WHERE id = $1 AND association_id = $2 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, id, associationID))
}

// List は自治会のお知らせ一覧を返す（ピン留め優先・新着順、is_read付き）
func (r *Repository) List(ctx context.Context, associationID, userID uuid.UUID) ([]*Notice, error) {
	const q = `
		SELECT n.id, n.association_id, n.title, n.body, n.is_pinned, n.created_by,
		       n.created_at, n.updated_at, n.deleted_at,
		       (nr.id IS NOT NULL) AS is_read
		FROM notices n
		LEFT JOIN notice_reads nr ON nr.notice_id = n.id AND nr.user_id = $2
		WHERE n.association_id = $1 AND n.deleted_at IS NULL
		ORDER BY n.is_pinned DESC, n.created_at DESC`

	rows, err := r.pool.Query(ctx, q, associationID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Notice
	for rows.Next() {
		n := &Notice{}
		if err := rows.Scan(
			&n.ID, &n.AssociationID, &n.Title, &n.Body, &n.IsPinned, &n.CreatedBy,
			&n.CreatedAt, &n.UpdatedAt, &n.DeletedAt, &n.IsRead,
		); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

// SoftDelete はお知らせを論理削除する（association_idで境界チェック）
func (r *Repository) SoftDelete(ctx context.Context, id, associationID uuid.UUID) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE notices SET deleted_at = $1
		 WHERE id = $2 AND association_id = $3 AND deleted_at IS NULL`,
		now, id, associationID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAsRead はお知らせを既読にする（冪等操作）
func (r *Repository) MarkAsRead(ctx context.Context, noticeID, userID uuid.UUID) error {
	const q = `
		INSERT INTO notice_reads (id, notice_id, user_id, read_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (notice_id, user_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, q, uuid.New(), noticeID, userID, time.Now())
	return err
}

// UnreadCount は未読件数を返す
func (r *Repository) UnreadCount(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM notices n
		LEFT JOIN notice_reads nr ON nr.notice_id = n.id AND nr.user_id = $2
		WHERE n.association_id = $1 AND n.deleted_at IS NULL AND nr.id IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, q, associationID, userID).Scan(&count)
	return count, err
}

// scanOne は単一行スキャン（is_readなし）
func (r *Repository) scanOne(row interface{ Scan(dest ...any) error }) (*Notice, error) {
	n := &Notice{}
	err := row.Scan(
		&n.ID, &n.AssociationID, &n.Title, &n.Body, &n.IsPinned, &n.CreatedBy,
		&n.CreatedAt, &n.UpdatedAt, &n.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return n, nil
}
