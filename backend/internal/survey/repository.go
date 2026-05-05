package survey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository はアンケートの DB 操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create はアンケートと質問・選択肢をトランザクションで作成する
func (r *Repository) Create(ctx context.Context, s *Survey) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO surveys (id, association_id, title, description, expires_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		s.ID, s.AssociationID, s.Title, s.Description, s.ExpiresAt, s.CreatedBy, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for _, q := range s.Questions {
		_, err = tx.Exec(ctx, `
			INSERT INTO survey_questions (id, survey_id, question_text, question_type, sort_order)
			VALUES ($1, $2, $3, $4, $5)`,
			q.ID, s.ID, q.QuestionText, q.QuestionType, q.SortOrder,
		)
		if err != nil {
			return err
		}
		for _, c := range q.Choices {
			_, err = tx.Exec(ctx, `
				INSERT INTO survey_choices (id, question_id, choice_text, sort_order)
				VALUES ($1, $2, $3, $4)`,
				c.ID, q.ID, c.ChoiceText, c.SortOrder,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// SoftDelete はアンケートを論理削除する
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE surveys SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List はアンケート一覧を返す（論理削除済みを除く）。
// userID が指定されている場合は IsAnswered フラグも設定する。
func (r *Repository) List(ctx context.Context, associationID uuid.UUID, userID *uuid.UUID) ([]*Survey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.association_id, s.title, s.description, s.expires_at,
		       s.created_by, s.created_at, s.updated_at,
		       CASE WHEN ua.user_id IS NOT NULL THEN true ELSE false END AS is_answered
		FROM surveys s
		LEFT JOIN (
			SELECT DISTINCT survey_id, user_id
			FROM survey_answers
			WHERE user_id = $2
		) ua ON ua.survey_id = s.id
		WHERE s.association_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.created_at DESC`,
		associationID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var surveys []*Survey
	for rows.Next() {
		s := &Survey{}
		if err := rows.Scan(
			&s.ID, &s.AssociationID, &s.Title, &s.Description, &s.ExpiresAt,
			&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt, &s.IsAnswered,
		); err != nil {
			return nil, err
		}
		surveys = append(surveys, s)
	}
	if surveys == nil {
		surveys = []*Survey{}
	}
	return surveys, rows.Err()
}

// FindByID はアンケートを質問・選択肢・画像つきで取得する
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Survey, error) {
	var s Survey
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.association_id, s.title, s.description, s.expires_at,
		       s.created_by, s.created_at, s.updated_at,
		       CASE WHEN ua.user_id IS NOT NULL THEN true ELSE false END AS is_answered
		FROM surveys s
		LEFT JOIN (
			SELECT DISTINCT survey_id, user_id
			FROM survey_answers
			WHERE user_id = $2
		) ua ON ua.survey_id = s.id
		WHERE s.id = $1 AND s.deleted_at IS NULL`,
		id, userID,
	).Scan(
		&s.ID, &s.AssociationID, &s.Title, &s.Description, &s.ExpiresAt,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt, &s.IsAnswered,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	questions, err := r.listQuestions(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Questions = questions

	images, err := r.ListImages(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Images = images

	return &s, nil
}

// listQuestions は質問と選択肢を取得する
func (r *Repository) listQuestions(ctx context.Context, surveyID uuid.UUID) ([]Question, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, survey_id, question_text, question_type, sort_order
		FROM survey_questions
		WHERE survey_id = $1
		ORDER BY sort_order ASC`, surveyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.ID, &q.SurveyID, &q.QuestionText, &q.QuestionType, &q.SortOrder); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range questions {
		choices, err := r.listChoices(ctx, questions[i].ID)
		if err != nil {
			return nil, err
		}
		questions[i].Choices = choices
	}
	return questions, nil
}

// listChoices は選択肢を取得する
func (r *Repository) listChoices(ctx context.Context, questionID uuid.UUID) ([]Choice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, question_id, choice_text, sort_order
		FROM survey_choices
		WHERE question_id = $1
		ORDER BY sort_order ASC`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var choices []Choice
	for rows.Next() {
		var c Choice
		if err := rows.Scan(&c.ID, &c.QuestionID, &c.ChoiceText, &c.SortOrder); err != nil {
			return nil, err
		}
		choices = append(choices, c)
	}
	return choices, rows.Err()
}

// SaveAnswers はユーザーの回答を保存する（既存回答を置き換える）
func (r *Repository) SaveAnswers(ctx context.Context, surveyID, userID uuid.UUID, answers []Answer) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`DELETE FROM survey_answers WHERE survey_id = $1 AND user_id = $2`,
		surveyID, userID,
	)
	if err != nil {
		return err
	}

	for _, a := range answers {
		_, err = tx.Exec(ctx, `
			INSERT INTO survey_answers (id, survey_id, user_id, question_id, choice_id, answered_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, a.SurveyID, a.UserID, a.QuestionID, a.ChoiceID, a.AnsweredAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetResults はアンケートの集計結果を取得する
func (r *Repository) GetResults(ctx context.Context, surveyID uuid.UUID) (*SurveyResult, error) {
	var result SurveyResult
	err := r.pool.QueryRow(ctx, `
		SELECT id, title FROM surveys WHERE id = $1 AND deleted_at IS NULL`, surveyID,
	).Scan(&result.SurveyID, &result.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM survey_answers WHERE survey_id = $1`, surveyID,
	).Scan(&result.TotalAnswered); err != nil {
		return nil, err
	}

	qRows, err := r.pool.Query(ctx, `
		SELECT id, question_text, question_type, sort_order
		FROM survey_questions WHERE survey_id = $1 ORDER BY sort_order ASC`, surveyID)
	if err != nil {
		return nil, err
	}
	defer qRows.Close()

	for qRows.Next() {
		var qr QuestionResult
		if err := qRows.Scan(&qr.QuestionID, &qr.QuestionText, &qr.QuestionType, new(int)); err != nil {
			return nil, err
		}

		cRows, err := r.pool.Query(ctx, `
			SELECT sc.id, sc.choice_text,
			       COUNT(sa.id) AS cnt
			FROM survey_choices sc
			LEFT JOIN survey_answers sa ON sa.choice_id = sc.id AND sa.survey_id = $2
			WHERE sc.question_id = $1
			GROUP BY sc.id, sc.choice_text, sc.sort_order
			ORDER BY sc.sort_order ASC`,
			qr.QuestionID, surveyID,
		)
		if err != nil {
			return nil, err
		}
		for cRows.Next() {
			var cr ChoiceResult
			if err := cRows.Scan(&cr.ChoiceID, &cr.ChoiceText, &cr.Count); err != nil {
				cRows.Close()
				return nil, err
			}
			qr.TotalAnswers += cr.Count
			qr.Choices = append(qr.Choices, cr)
		}
		cRows.Close()

		for i := range qr.Choices {
			if qr.TotalAnswers > 0 {
				qr.Choices[i].Percentage = float64(qr.Choices[i].Count) / float64(qr.TotalAnswers) * 100
			}
		}
		result.Questions = append(result.Questions, qr)
	}

	return &result, qRows.Err()
}

// CountUnanswered はユーザーの未回答アンケート件数を返す
func (r *Repository) CountUnanswered(ctx context.Context, associationID, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM surveys s
		WHERE s.association_id = $1
		  AND s.deleted_at IS NULL
		  AND s.expires_at > NOW()
		  AND NOT EXISTS (
		      SELECT 1 FROM survey_answers sa
		      WHERE sa.survey_id = s.id AND sa.user_id = $2
		  )`,
		associationID, userID,
	).Scan(&count)
	return count, err
}

// ValidateChoices はアンケートの質問と選択肢が正しいか検証する
func (r *Repository) ValidateChoices(ctx context.Context, surveyID uuid.UUID, answers []QuestionAnswerInput) error {
	for _, a := range answers {
		var questionType string
		if err := r.pool.QueryRow(ctx,
			`SELECT question_type FROM survey_questions WHERE id = $1 AND survey_id = $2`,
			a.QuestionID, surveyID,
		).Scan(&questionType); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("invalid question_id")
			}
			return err
		}
		if questionType == "single" && len(a.ChoiceIDs) != 1 {
			return errors.New("single choice question requires exactly one choice")
		}
		if len(a.ChoiceIDs) == 0 {
			return errors.New("at least one choice is required")
		}

		for _, cID := range a.ChoiceIDs {
			var exists bool
			if err := r.pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM survey_choices WHERE id = $1 AND question_id = $2)`,
				cID, a.QuestionID,
			).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return errors.New("invalid choice_id")
			}
		}
	}
	return nil
}

// CheckSurveyExpiry はアンケートが期限内かチェックする
func (r *Repository) CheckSurveyExpiry(ctx context.Context, id uuid.UUID) (bool, error) {
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT expires_at FROM surveys WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	return expiresAt.After(time.Now()), nil
}

// ─── 画像 CRUD ───────────────────────────────────────────────

// CreateImage はアンケート画像レコードを保存する
func (r *Repository) CreateImage(ctx context.Context, img *SurveyImage) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO survey_images (id, survey_id, association_id, filename, storage_path, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		img.ID, img.SurveyID, img.AssociationID, img.Filename, img.StoragePath, img.SortOrder, img.CreatedAt,
	)
	return err
}

// ListImages はアンケートの画像一覧を sort_order 昇順で返す
func (r *Repository) ListImages(ctx context.Context, surveyID uuid.UUID) ([]SurveyImage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, survey_id, association_id, filename, storage_path, sort_order, created_at
		FROM survey_images
		WHERE survey_id = $1
		ORDER BY sort_order ASC, created_at ASC`, surveyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []SurveyImage
	for rows.Next() {
		var img SurveyImage
		if err := rows.Scan(
			&img.ID, &img.SurveyID, &img.AssociationID,
			&img.Filename, &img.StoragePath, &img.SortOrder, &img.CreatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	if images == nil {
		images = []SurveyImage{}
	}
	return images, rows.Err()
}

// FindImage は画像を ID で取得する
func (r *Repository) FindImage(ctx context.Context, imageID uuid.UUID) (*SurveyImage, error) {
	var img SurveyImage
	err := r.pool.QueryRow(ctx, `
		SELECT id, survey_id, association_id, filename, storage_path, sort_order, created_at
		FROM survey_images WHERE id = $1`, imageID,
	).Scan(
		&img.ID, &img.SurveyID, &img.AssociationID,
		&img.Filename, &img.StoragePath, &img.SortOrder, &img.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrImageNotFound
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// DeleteImage は画像レコードを物理削除する
func (r *Repository) DeleteImage(ctx context.Context, imageID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM survey_images WHERE id = $1`, imageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}
