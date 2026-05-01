package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"circular-board/internal/config"
)

var (
	ErrFileTooLarge       = errors.New("file size exceeds limit")
	ErrUnsupportedMIME    = errors.New("unsupported file type")
	ErrForbidden          = errors.New("access forbidden")
	ErrNoAssociation      = errors.New("user has no association")
)

// UploadInput はアップロード時の入力値
type UploadInput struct {
	Reader           io.Reader
	OriginalFilename string
	Size             int64
	MimeType         string
	Year             int // 0 = 現在の年
	Month            int // 0 = 現在の月
}

// Service はファイル管理のビジネスロジック
type Service struct {
	repo *Repository
	cfg  *config.Config
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// Upload はファイルを受け取りディスクに保存してDBに登録する
func (s *Service) Upload(ctx context.Context, associationID, uploadedBy uuid.UUID, input UploadInput) (*File, error) {
	if !IsAllowedMIME(input.MimeType) {
		return nil, ErrUnsupportedMIME
	}
	if input.Size > s.cfg.MaxUploadBytes {
		return nil, ErrFileTooLarge
	}

	now := time.Now()
	year := input.Year
	month := input.Month
	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}

	ext := ExtensionForMIME(input.MimeType)
	storedName := uuid.New().String() + ext

	dirPath := filepath.Join(
		s.cfg.UploadDir,
		associationID.String(),
		fmt.Sprintf("%d", year),
		fmt.Sprintf("%02d", month),
	)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	storagePath := filepath.Join(dirPath, storedName)
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, input.Reader)
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("write file: %w", err)
	}

	file := &File{
		ID:               uuid.New(),
		AssociationID:    associationID,
		Year:             year,
		Month:            month,
		Filename:         storedName,
		OriginalFilename: input.OriginalFilename,
		StoragePath:      storagePath,
		UploadedBy:       uploadedBy,
		FileSize:         written,
		MimeType:         input.MimeType,
		CreatedAt:        now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("save file record: %w", err)
	}

	return file, nil
}

// Delete はファイルを論理削除しディスクからも削除する
func (s *Service) Delete(ctx context.Context, associationID uuid.UUID, fileID uuid.UUID) error {
	// ファイル情報を取得して物理パスを得る
	file, err := s.repo.FindByID(ctx, fileID, associationID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find file: %w", err)
	}

	if err := s.repo.SoftDelete(ctx, fileID, associationID); err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}

	// 物理ファイルを削除（論理削除後に実施）
	_ = os.Remove(file.StoragePath)

	return nil
}

// List は自治会のファイル一覧を返す
func (s *Service) List(ctx context.Context, associationID uuid.UUID, year, month int) ([]*File, error) {
	return s.repo.List(ctx, associationID, ListFilter{Year: year, Month: month})
}

// GetFile はダウンロード用にファイルレコードを返す（テナントチェック込み）
func (s *Service) GetFile(ctx context.Context, associationID uuid.UUID, fileID uuid.UUID) (*File, error) {
	file, err := s.repo.FindByID(ctx, fileID, associationID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	return file, err
}

// AvailableYears はその自治会でファイルが存在する年の一覧を返す
func (s *Service) AvailableYears(ctx context.Context, associationID uuid.UUID) ([]int, error) {
	return s.repo.AvailableYears(ctx, associationID)
}
