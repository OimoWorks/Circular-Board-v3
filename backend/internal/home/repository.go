package home

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultLimit = 3

// Repository はTOP画面用のDB集約クエリを担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListRecentNotices は自治会の最新お知らせを limit 件返す。
// ピン留め優先・作成日時降順。is_read はユーザーごとに判定する。
func (r *Repository) ListRecentNotices(ctx context.Context, associationID, userID uuid.UUID, limit int) ([]HomeNotice, error) {
	const q = `
		SELECT n.id, n.title, n.is_pinned, n.created_at,
		       (nr.id IS NOT NULL) AS is_read
		FROM notices n
		LEFT JOIN notice_reads nr ON nr.notice_id = n.id AND nr.user_id = $2
		WHERE n.association_id = $1 AND n.deleted_at IS NULL
		ORDER BY n.is_pinned DESC, n.created_at DESC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, q, associationID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notices := make([]HomeNotice, 0)
	for rows.Next() {
		var n HomeNotice
		if err := rows.Scan(&n.ID, &n.Title, &n.IsPinned, &n.CreatedAt, &n.IsRead); err != nil {
			return nil, err
		}
		notices = append(notices, n)
	}
	return notices, rows.Err()
}

// ListRecentFiles は自治会の最新回覧物を limit 件返す（年月降順・作成日時降順）。
func (r *Repository) ListRecentFiles(ctx context.Context, associationID uuid.UUID, limit int) ([]HomeFile, error) {
	const q = `
		SELECT id, year, month, original_filename, file_size, mime_type, created_at
		FROM files
		WHERE association_id = $1 AND deleted_at IS NULL
		ORDER BY year DESC, month DESC, created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, q, associationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]HomeFile, 0)
	for rows.Next() {
		var f HomeFile
		if err := rows.Scan(
			&f.ID, &f.Year, &f.Month,
			&f.OriginalFilename, &f.FileSize, &f.MimeType, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// ListUnansweredSurveys は自治会の未回答・期限内アンケートを limit 件返す（期限昇順）。
func (r *Repository) ListUnansweredSurveys(ctx context.Context, associationID, userID uuid.UUID, limit int) ([]HomeSurvey, error) {
	const q = `
		SELECT id, title, expires_at
		FROM surveys
		WHERE association_id = $1
		  AND deleted_at IS NULL
		  AND expires_at > NOW()
		  AND NOT EXISTS (
		      SELECT 1 FROM survey_answers
		      WHERE survey_id = surveys.id AND user_id = $2
		  )
		ORDER BY expires_at ASC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, q, associationID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	surveys := make([]HomeSurvey, 0)
	for rows.Next() {
		var s HomeSurvey
		if err := rows.Scan(&s.ID, &s.Title, &s.ExpiresAt); err != nil {
			return nil, err
		}
		surveys = append(surveys, s)
	}
	return surveys, rows.Err()
}

// CountUnreadNotices は自治会の未読お知らせ件数を返す。
func (r *Repository) CountUnreadNotices(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM notices n
		LEFT JOIN notice_reads nr ON nr.notice_id = n.id AND nr.user_id = $2
		WHERE n.association_id = $1 AND n.deleted_at IS NULL AND nr.id IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, q, associationID, userID).Scan(&count)
	return count, err
}

// CountUnansweredSurveys は自治会の未回答・期限内アンケート件数を返す。
func (r *Repository) CountUnansweredSurveys(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM surveys
		WHERE association_id = $1
		  AND deleted_at IS NULL
		  AND expires_at > NOW()
		  AND NOT EXISTS (
		      SELECT 1 FROM survey_answers
		      WHERE survey_id = surveys.id AND user_id = $2
		  )`
	var count int
	err := r.pool.QueryRow(ctx, q, associationID, userID).Scan(&count)
	return count, err
}
