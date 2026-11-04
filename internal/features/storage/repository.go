package storage

import (
	"context"
	"fitbase/internal/models"

	"errors"
	"fitbase/internal/shared/helper"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) WithTransaction(fn func(repo *PostgresRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*models.File, error) {
	var file models.File

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &file, nil
}

func (r *PostgresRepository) Create(ctx context.Context, file *models.File) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(file).Error)
}

func (r *PostgresRepository) Delete(ctx context.Context, slug string) error {
	result := r.db.WithContext(ctx).Where("slug = ?", slug).Delete(&models.File{})
	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrFileNotFound
	}
	return nil
}
