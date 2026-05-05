package home

import (
	"time"

	"github.com/google/uuid"
)

// HomeNotice はTOP画面用のお知らせ要約
type HomeNotice struct {
	ID        uuid.UUID
	Title     string
	IsPinned  bool
	IsRead    bool
	CreatedAt time.Time
}

// HomeFile はTOP画面用の回覧物要約
type HomeFile struct {
	ID               uuid.UUID
	Year             int
	Month            int
	OriginalFilename string
	FileSize         int64
	MimeType         string
	CreatedAt        time.Time
}

// HomeSurvey はTOP画面用のアンケート要約（未回答のみ）
type HomeSurvey struct {
	ID        uuid.UUID
	Title     string
	ExpiresAt time.Time
}

// HomeData はTOP画面に表示する集約データ
type HomeData struct {
	Notices               []HomeNotice
	Files                 []HomeFile
	Surveys               []HomeSurvey
	UnreadNoticeCount     int
	UnansweredSurveyCount int
}
