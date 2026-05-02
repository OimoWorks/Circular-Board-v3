package files

import (
	"time"

	"github.com/google/uuid"
)

// File はファイル管理テーブルのドメインモデル
type File struct {
	ID               uuid.UUID
	AssociationID    uuid.UUID
	Year             int
	Month            int
	Filename         string // 保存ファイル名（UUID形式）
	OriginalFilename string // アップロード元のファイル名
	StoragePath      string // ディスク上のフルパス
	UploadedBy       uuid.UUID
	FileSize         int64
	MimeType         string
	CreatedAt        time.Time
	DeletedAt        *time.Time
}

// ListFilter はファイル一覧取得のフィルタ条件
type ListFilter struct {
	Year  int // 0 = すべての年
	Month int // 0 = すべての月
}

// AllowedMIMETypes は受け付けるMIMEタイプの一覧
var AllowedMIMETypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// IsAllowedMIME はアップロード可能なMIMEタイプかを判定する
func IsAllowedMIME(mimeType string) bool {
	_, ok := AllowedMIMETypes[mimeType]
	return ok
}

// ExtensionForMIME はMIMEタイプに対応する拡張子を返す
func ExtensionForMIME(mimeType string) string {
	if ext, ok := AllowedMIMETypes[mimeType]; ok {
		return ext
	}
	return ""
}

// AssociationRecord は自治会の簡易情報
type AssociationRecord struct {
	ID   uuid.UUID
	Name string
	Code string
}
