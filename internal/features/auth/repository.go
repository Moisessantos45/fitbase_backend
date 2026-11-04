package auth

import (
	"context"
	"errors"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fmt"
	"log"

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

func (r *PostgresRepository) Register(user *models.Auth) error {
	return helper.TranslateDBError(r.db.Create(user).Error)
}

func (r *PostgresRepository) GetByEmail(email string) (*models.Auth, error) {
	var user models.Auth
	err := r.db.Where("email = ? AND email_confirmed = ? AND active = ?", email, true, true).First(&user).Error

	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetByEmailAny(email string) (*models.Auth, error) {
	var user models.Auth
	err := r.db.Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetBasicByID(id uint64) (*UserBasic, error) {
	var user UserBasic
	err := r.db.Model(&models.Auth{}).Where("id = ? AND email_confirmed = ? AND active = ?", id, true, true).Take(&user).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, err
}

func (r *PostgresRepository) GetBasicByEmail(email string) (*UserBasic, error) {
	var user UserBasic
	err := r.db.Model(&models.Auth{}).Where("email = ? AND email_confirmed = ? AND active = ?", email, true, true).Take(&user).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, err
}

func (r *PostgresRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Auth{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) GetActiveUserByEmail(context context.Context, email string) (*models.Auth, error) {
	var user models.Auth
	err := r.db.
		WithContext(context).
		// Preload("roles", func(db *gorm.DB) *gorm.DB {
		// 	return db.Select("id", "name")
		// }).
		Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	if user.Active == false {
		log.Printf("Intento de inicio de sesión con cuenta inactiva: %s", email)
		return nil, errorsx.ErrCodeAccountInactive
	}

	if user.EmailConfirmed == false {
		log.Printf("Intento de inicio de sesión con email no verificado: %s", email)
		return nil, errorsx.ErrCodeEmailNotVerified
	}

	return &user, nil
}

func (r *PostgresRepository) GetByID(userId uint64) (*models.Auth, error) {
	var user models.Auth
	err := r.db.Where("id = ? AND active = ? AND email_confirmed = ?", userId, true, true).First(&user).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateEmailConfirm(userId uint64) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("email_confirmed", true).Error
}

func (r *PostgresRepository) UpdatePassword(userId uint64, newPassword string) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("password_hash", newPassword).Error
}

func (r *PostgresRepository) SetTwoFactor(userId uint64, enabled bool, secret string) error {
	updates := map[string]any{
		"two_factor_enabled": enabled,
		"two_factor_secret":  "",
	}

	if enabled {
		updates["two_factor_secret"] = secret
	}

	return r.db.Model(&models.Auth{}).
		Where("id = ?", userId).
		Updates(updates).Error
}

func (r *PostgresRepository) ChangeStatus(id uint64, active bool) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", id).Update("active", active).Error
}

func (r *PostgresRepository) DeleteByID(id uint64) error {
	return r.db.Where("id = ?", id).Delete(&models.Auth{}).Error
}

func (r *PostgresRepository) UpdateEmail(id uint64, email string) error {
	result := r.db.Model(&models.Auth{}).Where("id = ?", id).Updates(map[string]any{
		"email":           email,
		"email_confirmed": false,
	})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no auth found with id %d", id)
	}

	return nil
}
