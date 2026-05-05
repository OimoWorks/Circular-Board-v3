package survey

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("survey not found")
	ErrExpired    = errors.New("survey has expired")
	ErrForbidden  = errors.New("forbidden")
)

// Survey はアンケートのドメインモデル
type Survey struct {
	ID            uuid.UUID
	AssociationID uuid.UUID
	Title         string
	Description   string
	ExpiresAt     time.Time
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time

	Questions  []Question
	IsAnswered bool // 呼び出し元ユーザーが回答済みか
}

// Question はアンケートの質問
type Question struct {
	ID           uuid.UUID
	SurveyID     uuid.UUID
	QuestionText string
	QuestionType string // "single" | "multiple"
	SortOrder    int
	Choices      []Choice
}

// Choice は質問の選択肢
type Choice struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	ChoiceText string
	SortOrder  int
}

// Answer は個別の回答行
type Answer struct {
	ID         uuid.UUID
	SurveyID   uuid.UUID
	UserID     uuid.UUID
	QuestionID uuid.UUID
	ChoiceID   uuid.UUID
	AnsweredAt time.Time
}

// SurveyResult は集計結果
type SurveyResult struct {
	SurveyID      uuid.UUID
	Title         string
	TotalAnswered int // ユニーク回答者数
	Questions     []QuestionResult
}

// QuestionResult は質問ごとの集計
type QuestionResult struct {
	QuestionID   uuid.UUID
	QuestionText string
	QuestionType string
	TotalAnswers int
	Choices      []ChoiceResult
}

// ChoiceResult は選択肢ごとの集計
type ChoiceResult struct {
	ChoiceID   uuid.UUID
	ChoiceText string
	Count      int
	Percentage float64
}

// CreateInput はアンケート作成の入力値
type CreateInput struct {
	Title       string
	Description string
	ExpiresAt   time.Time
	Questions   []QuestionInput
}

// QuestionInput は質問の入力値
type QuestionInput struct {
	QuestionText string
	QuestionType string
	SortOrder    int
	Choices      []ChoiceInput
}

// ChoiceInput は選択肢の入力値
type ChoiceInput struct {
	ChoiceText string
	SortOrder  int
}

// AnswerInput は回答送信の入力値
type AnswerInput struct {
	Answers []QuestionAnswerInput
}

// QuestionAnswerInput は質問に対する回答
type QuestionAnswerInput struct {
	QuestionID uuid.UUID
	ChoiceIDs  []uuid.UUID
}
