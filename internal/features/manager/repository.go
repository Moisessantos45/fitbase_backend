package manager

import (
	"context"
	"errors"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"time"

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

func (r *PostgresRepository) CreateManagerWithAuth(ctx context.Context, authRecord *models.Auth, manager *models.GymManager) error {
	return r.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.db.WithContext(ctx).Create(authRecord).Error; err != nil {
			return helper.TranslateDBError(err)
		}

		manager.AuthID = authRecord.ID

		if err := txRepo.db.WithContext(ctx).Create(manager).Error; err != nil {
			return helper.TranslateDBError(err)
		}

		return nil
	})
}

func (r *PostgresRepository) GetActiveCashRegister(ctx context.Context, gymID, authID uint64) (*models.CashRegister, error) {
	var register models.CashRegister
	err := r.db.WithContext(ctx).
		Where("gym_id = ? AND auth_id = ? AND close_time IS NULL", gymID, authID).
		First(&register).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &register, nil
}

func (r *PostgresRepository) GetActiveCashRegisterByAuthID(ctx context.Context, authID uint64) (*models.CashRegister, error) {
	var register models.CashRegister
	err := r.db.WithContext(ctx).
		Where("auth_id = ? AND close_time IS NULL", authID).
		First(&register).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &register, nil
}

func (r *PostgresRepository) OpenCashRegister(ctx context.Context, register *models.CashRegister) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(register).Error)
}

func (r *PostgresRepository) CloseCashRegister(ctx context.Context, registerID uint64, closeCash float64, closeTime time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&models.CashRegister{}).
		Where("id = ? AND close_time IS NULL", registerID).
		Updates(map[string]any{
			"close_cash": closeCash,
			"close_time": closeTime,
		})

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) GetAllByGymID(ctx context.Context, authID uint64, offset int, limit int, query string) ([]Items, int64, error) {
	var managers []Items
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).
		Table("gym_managers gm").
		Joins("JOIN auths ON auths.id = gm.auth_id").
		Where("gm.created_by_auth_id = ?", authID)

	if query != "" {
		searchTerm := "%" + query + "%"
		baseQuery = baseQuery.Where(
			r.db.Where("gm.first_name ILIKE ?", searchTerm).
				Or("gm.last_name ILIKE ?", searchTerm).
				Or("auths.email ILIKE ?", searchTerm).
				Or("gm.phone ILIKE ?", searchTerm),
		)
	}

	countQuery := baseQuery.Session(&gorm.Session{})

	err := countQuery.Count(&totalCount).Error
	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	dataQuery := baseQuery.Session(&gorm.Session{})
	err = dataQuery.
		Select(`
			gm.id,
			gm.gym_id,
			gyms.name AS gym_name,
			gm.auth_id,
			gm.created_by_auth_id,
			gm.first_name,
			gm.last_name,
			gm.phone,
			auths.email,
			COALESCE(files.slug, '') AS slug,
			gm.created_at
		`).
		Joins("JOIN gyms ON gyms.id = gm.gym_id").
		Joins("LEFT JOIN files ON files.id = gm.avatar_id").
		Order("gm.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&managers).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return managers, totalCount, nil
}

func (r *PostgresRepository) GetDetailByAuthID(ctx context.Context, authID uint64) (*Detail, error) {
	var detail Detail
	err := r.db.WithContext(ctx).
		Table("gym_managers").
		Select(`
			gym_managers.id,
			gym_managers.auth_id,
			gym_managers.first_name,
			gym_managers.last_name,
			gym_managers.phone,
			COALESCE(files.slug, '') AS slug
		`).
		Joins("JOIN auths ON auths.id = gym_managers.auth_id").
		Joins("LEFT JOIN files ON files.id = gym_managers.avatar_id").
		Where("gym_managers.auth_id = ?", authID).
		Scan(&detail).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ErrNotFound
		}
		return nil, helper.TranslateDBError(err)
	}

	return &detail, nil
}

func (r *PostgresRepository) ExistsAuthByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Auth{}).
		Where("email = ?", email).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) ExistsByAuthID(ctx context.Context, authID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.GymManager{}).
		Where("auth_id = ?", authID).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) GetGymIDByOwnerAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var gymID uint64
	err := r.db.WithContext(ctx).
		Table("gyms").
		Select("gyms.id").
		Joins("INNER JOIN owners ON owners.id = gyms.owner_id").
		Where("owners.auth_id = ?", authID).
		Order("gyms.id ASC").
		Limit(1).
		Scan(&gymID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if gymID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return gymID, nil
}

func (r *PostgresRepository) CreateFile(ctx context.Context, file *models.File) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(file).Error)
}

func (r *PostgresRepository) Update(ctx context.Context, authID uint64, data map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.GymManager{}).
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
		Model(&models.GymManager{}).
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
