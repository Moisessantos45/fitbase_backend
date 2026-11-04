package storage

import (
	"context"
	"errors"
	"fitbase/internal/models"
)

var (
	ErrFileNotFound     = errors.New("file not found")
	ErrInvalidFile      = errors.New("invalid file")
	ErrFileIsPrivate    = errors.New("file is private")
	ErrUnauthorizedFile = errors.New("unauthorized to access this file")
)

type FileInfoBasic struct {
	FileID       string `json:"file_id" gorm:"column:file_id"`
	PermissionID uint64 `json:"permission_id" gorm:"column:permission_id"`
	Url          string `json:"url"`
}

type PaginatedFiles struct {
	Data     []models.File     `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetBySlug(ctx context.Context, slug string) (*models.File, error)
	Delete(ctx context.Context, slug string) error
}

type Service interface {
	GetBySlug(ctx context.Context, slug string) (*models.File, error)
	GetBySlugPublic(ctx context.Context, slug string) (*models.File, error)
	GetBySlugPrivate(ctx context.Context, authID uint64, slug string) (*models.File, error)
	Delete(ctx context.Context, authID uint64, slug string) error
}
