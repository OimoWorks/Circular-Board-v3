package files

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("file not found")

// Repository はファイルのDB操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create はファイルレコードを挿入する
func (r *Repository) Create(ctx context.Context, f *File) error {
	const q = `
		INSERT INTO files
			(id, association_id, year, month, filename, original_filename,
			 storage_path, uploaded_by, file_size, mime_type, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err := r.pool.Exec(ctx, q,
		f.ID, f.AssociationID, f.Year, f.Month,
		f.Filename, f.OriginalFilename, f.StoragePath,
		f.UploadedBy, f.FileSize, f.MimeType, f.CreatedAt,
	)
	return err
}

// FindByID はIDと自治会IDでファイルを取得する（テナント境界の強制）
func (r *Repository) FindByID(ctx context.Context, id, associationID uuid.UUID) (*File, error) {
	const q = `
		SELECT id, association_id, year, month, filename, original_filename,
		       storage_path, uploaded_by, file_size, mime_type, created_at, deleted_at
		FROM files
		WHERE id = $1 AND association_id = $2 AND deleted_at IS NULL`

	return r.scan(r.pool.QueryRow(ctx, q, id, associationID))
}

// List は自治会ID＋年月フィルタでファイル一覧を返す
func (r *Repository) List(ctx context.Context, associationID uuid.UUID, filter ListFilter) ([]*File, error) {
	const q = `
		SELECT id, association_id, year, month, filename, original_filename,
		       storage_path, uploaded_by, file_size, mime_type, created_at, deleted_at
		FROM files
		WHERE association_id = $1
		  AND deleted_at IS NULL
		  AND ($2 = 0 OR year  = $2)
		  AND ($3 = 0 OR month = $3)
		ORDER BY year DESC, month DESC, created_at DESC`

	rows, err := r.pool.Query(ctx, q, associationID, filter.Year, filter.Month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		f, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// SoftDelete はファイルを論理削除する（association_idで境界チェック）
func (r *Repository) SoftDelete(ctx context.Context, id, associationID uuid.UUID) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE files SET deleted_at = $1 WHERE id = $2 AND association_id = $3 AND deleted_at IS NULL`,
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

// AvailableYears はその自治会でファイルが存在する年の一覧を返す
func (r *Repository) AvailableYears(ctx context.Context, associationID uuid.UUID) ([]int, error) {
	const q = `
		SELECT DISTINCT year FROM files
		WHERE association_id = $1 AND deleted_at IS NULL
		ORDER BY year DESC`

	rows, err := r.pool.Query(ctx, q, associationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, rows.Err()
}

// scan は行スキャンの共通処理
func (r *Repository) scan(row interface {
	Scan(dest ...any) error
}) (*File, error) {
	var f File
	err := row.Scan(
		&f.ID, &f.AssociationID, &f.Year, &f.Month,
		&f.Filename, &f.OriginalFilename, &f.StoragePath,
		&f.UploadedBy, &f.FileSize, &f.MimeType,
		&f.CreatedAt, &f.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}
