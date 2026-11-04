package subscription

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) WithTransaction(fn func(repo Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) Create(ctx context.Context, sub *models.Subscription) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(sub).Error)
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).First(&sub, "slug = ?", slug).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &sub, nil
}

func (r *PostgresRepository) GetTicketBySlug(ctx context.Context, slug uuid.UUID) (*SubscriptionTicketResponse, error) {
	var ticket SubscriptionTicketResponse
	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Select(`
			subscriptions.id,
			subscriptions.slug::text AS slug,
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			plans.id AS plan_id,
			plans.name AS plan_name,
			plans.price AS plan_price,
			TRIM(CONCAT(users.first_name, ' ', users.last_name)) AS full_name,
			auths.email AS email,
			status_subscriptions.id AS status_id,
			status_subscriptions.name AS status_name,
			COALESCE(TO_CHAR(subscriptions.start_date, 'YYYY-MM-DD'), '') AS start_date,
			COALESCE(TO_CHAR(subscriptions.end_date, 'YYYY-MM-DD'), '') AS end_date,
			subscriptions.auto_renew,
			subscriptions.created_at
		`).
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Joins("JOIN status_subscriptions ON status_subscriptions.id = subscriptions.status_id").
		Where("subscriptions.slug = ?", slug).
		Scan(&ticket).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if ticket.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &ticket, nil
}

func (r *PostgresRepository) GetTicketByID(ctx context.Context, id uint64) (*SubscriptionTicketResponse, error) {
	var ticket SubscriptionTicketResponse
	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Select(`
			subscriptions.id,
			subscriptions.slug::text AS slug,
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			plans.id AS plan_id,
			plans.name AS plan_name,
			plans.price AS plan_price,
			TRIM(CONCAT(users.first_name, ' ', users.last_name)) AS full_name,
			auths.email AS email,
			status_subscriptions.id AS status_id,
			status_subscriptions.name AS status_name,
			COALESCE(TO_CHAR(subscriptions.start_date, 'YYYY-MM-DD'), '') AS start_date,
			COALESCE(TO_CHAR(subscriptions.end_date, 'YYYY-MM-DD'), '') AS end_date,
			subscriptions.auto_renew,
			subscriptions.created_at
		`).
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Joins("JOIN status_subscriptions ON status_subscriptions.id = subscriptions.status_id").
		Where("subscriptions.id = ?", id).
		Scan(&ticket).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if ticket.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &ticket, nil
}

func (r *PostgresRepository) GetActiveByUserID(ctx context.Context, userID, gymID uint64) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Where("subscriptions.user_id = ? AND plans.gym_id = ? AND subscriptions.status_id IN (?, ?) AND subscriptions.end_date >= CURRENT_DATE",
			userID, gymID, models.StatusSubscriptionActiveID, models.StatusSubscriptionInactiveID).
		Order("subscriptions.end_date DESC").
		First(&sub).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &sub, nil
}

func (r *PostgresRepository) GetAllByGymID(ctx context.Context, gymID uint64, statusID uint64, offset, limit int) ([]SubscriptionDetailResponse, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("subscriptions").
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Where("plans.gym_id = ?", gymID)

	if statusID > 0 {
		baseQuery = baseQuery.Where("subscriptions.status_id = ?", statusID)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []SubscriptionDetailResponse
	err := dataQuery.
		Select(`
			subscriptions.id,
			subscriptions.slug::text AS slug,
			subscriptions.user_id,
			TRIM(CONCAT(users.first_name, ' ', users.last_name)) AS user_full_name,
			auths.email AS user_email,
			users.phone AS user_phone,
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			plans.id AS plan_id,
			plans.name AS plan_name,
			plans.price AS plan_price,
			status_subscriptions.id AS status_id,
			status_subscriptions.name AS status_name,
			COALESCE(TO_CHAR(subscriptions.start_date, 'YYYY-MM-DD'), '') AS start_date,
			COALESCE(TO_CHAR(subscriptions.end_date, 'YYYY-MM-DD'), '') AS end_date,
			payment_methods.id AS payment_method_id,
			payment_methods.name AS payment_method_name,
			subscriptions.auto_renew,
			subscriptions.created_at
		`).
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Joins("JOIN status_subscriptions ON status_subscriptions.id = subscriptions.status_id").
		Joins("JOIN payment_methods ON payment_methods.id = subscriptions.payment_method_id").
		Order("subscriptions.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]SubscriptionUserSummary, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("subscriptions").
		Where("subscriptions.user_id = ?", userID)

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []SubscriptionUserSummary
	err := dataQuery.
		Select(`
			subscriptions.id,
			subscriptions.slug::text AS slug,
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			plans.id AS plan_id,
			plans.name AS plan_name,
			plans.price AS plan_price,
			status_subscriptions.id AS status_id,
			status_subscriptions.name AS status_name,
			COALESCE(TO_CHAR(subscriptions.start_date, 'YYYY-MM-DD'), '') AS start_date,
			COALESCE(TO_CHAR(subscriptions.end_date, 'YYYY-MM-DD'), '') AS end_date,
			subscriptions.auto_renew,
			subscriptions.created_at
		`).
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("JOIN status_subscriptions ON status_subscriptions.id = subscriptions.status_id").
		Order("subscriptions.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uint64, statusID uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ?", id).
		Update("status_id", statusID)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uint64) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).First(&sub, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &sub, nil
}

func (r *PostgresRepository) Renew(ctx context.Context, id uint64, newStartDate, newEndDate time.Time, paymentMethodID uint64) error {
	updates := map[string]any{
		"start_date": newStartDate,
		"end_date":   newEndDate,
		"status_id":  models.StatusSubscriptionActiveID,
	}
	if paymentMethodID > 0 {
		updates["payment_method_id"] = paymentMethodID
	}

	result := r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) Cancel(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status_id":  models.StatusSubscriptionInactiveID,
			"auto_renew": false,
		})

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, string, string, string, error) {
	var user UserAuthProfile
	err := r.db.WithContext(ctx).
		Table("users").
		Select("users.id, TRIM(CONCAT(users.first_name, ' ', users.last_name)) AS full_name, users.phone, auths.email").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("users.auth_id = ?", authID).
		Scan(&user).Error

	if err != nil {
		return 0, "", "", "", helper.TranslateDBError(err)
	}

	if user.ID == 0 {
		return 0, "", "", "", errorsx.ErrNotFound
	}

	return user.ID, user.FullName, user.Phone, user.Email, nil
}

func (r *PostgresRepository) GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error) {
	var p models.Plan
	err := r.db.WithContext(ctx).First(&p, "id = ?", planID).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error) {
	var ownerID uint64
	err := r.db.WithContext(ctx).
		Model(&models.Gym{}).
		Select("owner_id").
		Where("id = ?", gymID).
		Scan(&ownerID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if ownerID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return ownerID, nil
}

func (r *PostgresRepository) GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var ownerID uint64
	err := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Select("id").
		Where("auth_id = ?", authID).
		Scan(&ownerID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if ownerID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return ownerID, nil
}

func (r *PostgresRepository) GetManagerGymID(ctx context.Context, authID uint64) (uint64, error) {
	var gymID uint64
	err := r.db.WithContext(ctx).
		Model(&models.GymManager{}).
		Select("gym_id").
		Where("auth_id = ?", authID).
		Scan(&gymID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if gymID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return gymID, nil
}
