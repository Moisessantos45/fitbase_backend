package user

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"

	"errors"

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

func (r *PostgresRepository) GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var userID uint64
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("id").Where("auth_id = ?", authID).Scan(&userID).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return 0, nil
		}
		return 0, helper.TranslateDBError(err)
	}

	return userID, nil
}

func (r *PostgresRepository) GetByAuthID(ctx context.Context, authID uint64) (*UserDetail, error) {
	var user UserDetail

	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select(`
			users.id,
			users.first_name,
			users.last_name,
			users.phone,
			auths.email,
			files.slug
		`).
		Joins("JOIN auths ON auths.id = users.auth_id").
		Joins("LEFT JOIN files ON files.id = users.avatar_id").
		Where("users.auth_id = ?", authID).
		First(&user).Error

	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetEmailByID(ctx context.Context, id uint64) (string, error) {
	var email string
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("auths.email").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("users.id = ?", id).
		Scan(&email).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return "", nil
		}
		return "", helper.TranslateDBError(err)
	}

	return email, nil
}

func (r *PostgresRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresRepository) CreateFile(ctx context.Context, file *models.File) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(file).Error)
}

func (r *PostgresRepository) Update(ctx context.Context, authID uint64, data map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
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

func (r *PostgresRepository) UpdateAvatarURL(ctx context.Context, authID uint64, avatarID uint64) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("auth_id = ?", authID).
		Update("avatar_id", avatarID).Error)
}

func (r *PostgresRepository) ChangeCompletedProfile(userId uint64, complete bool) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("full_profile", complete).Error
}
