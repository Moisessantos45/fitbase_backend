package storage

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/utils"
	"fmt"
	"strings"
)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) Service {
	return &UseCase{repo: repo}
}

func (s *UseCase) GetBySlug(ctx context.Context, slug string) (*models.File, error) {
	file, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, ErrFileNotFound
	}

	file.Url = utils.BuildEvidenceURL(file.Slug)

	return file, nil
}

func (s *UseCase) GetBySlugPublic(ctx context.Context, slug string) (*models.File, error) {
	file, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, ErrFileNotFound
	}

	file.Url = utils.BuildEvidenceURL(file.Slug)

	return file, nil
}

func (s *UseCase) GetBySlugPrivate(ctx context.Context, authID uint64, slug string) (*models.File, error) {
	file, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, ErrFileNotFound
	}

	file.Url = utils.BuildEvidenceURL(file.Slug)

	return file, nil
}

func (s *UseCase) Delete(ctx context.Context, authID uint64, slug string) error {
	if authID == 0 {
		return fmt.Errorf("authID is required")
	}

	if slug == "" || strings.TrimSpace(slug) == "" {
		return fmt.Errorf("slug is required")
	}

	file, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	if file == nil {
		return ErrFileNotFound
	}

	if file.AuthorID != authID {
		return ErrUnauthorizedFile
	}

	return s.repo.WithTransaction(func(repo *PostgresRepository) error {

		if err := repo.Delete(ctx, slug); err != nil {
			return fmt.Errorf("failed to delete file record: %w", err)
		}

		if err := utils.DeleteFile(file.Path); err != nil {
			return fmt.Errorf("failed to delete physical file: %w", err)
		}

		return nil
	})
}
