package owner

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
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

func (r *PostgresRepository) GetByAuthID(ctx context.Context, authID uint64) (*Detail, error) {
	var owner Detail
	err := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Select(`
			owners.id,
			owners.auth_id,
			owners.first_name,
			owners.last_name,
			owners.phone,
			auths.email,
			files.slug
		`).
		Joins("JOIN auths ON auths.id = owners.auth_id").
		Joins("LEFT JOIN files ON files.id = owners.avatar_id").
		Where("owners.auth_id = ?", authID).
		First(&owner).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return &owner, nil
}

func (r *PostgresRepository) ExistsByAuthID(ctx context.Context, authID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Where("auth_id = ?", authID).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) Create(ctx context.Context, owner *models.Owner) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(owner).Error)
}

func (r *PostgresRepository) CreateFile(ctx context.Context, file *models.File) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(file).Error)
}

func (r *PostgresRepository) Update(ctx context.Context, authID uint64, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Where("auth_id = ?", authID).
		Updates(data)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) UpdateAvatarByAuthID(ctx context.Context, authID uint64, avatarID uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Where("auth_id = ?", authID).
		Update("avatar_id", avatarID)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) MarkAuthProfileCompleted(ctx context.Context, authID uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Auth{}).
		Where("id = ?", authID).
		Update("full_profile", true)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}
