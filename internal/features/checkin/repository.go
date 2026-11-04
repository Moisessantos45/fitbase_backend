package checkin

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

func (r *PostgresRepository) Register(ctx context.Context, checkIn *models.CheckIn) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(checkIn).Error)
}

func (r *PostgresRepository) GetAllByGymID(ctx context.Context, gymID uint64, from, to time.Time, offset, limit int) ([]CheckInSummaryItem, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("check_ins").
		Where("check_ins.gym_id = ?", gymID)

	if !from.IsZero() {
		baseQuery = baseQuery.Where("check_ins.checked_in_at >= ?", from)
	}
	if !to.IsZero() {
		baseQuery = baseQuery.Where("check_ins.checked_in_at <= ?", to)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []CheckInSummaryItem
	err := dataQuery.
		Select(`
			check_ins.id,
			check_ins.gym_id,
			gyms.name AS gym_name,
			check_ins.access_type_id,
			access_types.name AS access_type_name,
			check_ins.user_id,
			COALESCE(NULLIF(TRIM(CONCAT(users.first_name, ' ', users.last_name)), ''), day_passes.full_name, 'Invitado') AS user_full_name,
			check_ins.day_pass_id,
			check_ins.subscription_id,
			check_ins.registered_by_id,
			COALESCE(auths.email, '') AS registered_by_name,
			check_ins.checked_in_at
		`).
		Joins("JOIN gyms ON gyms.id = check_ins.gym_id").
		Joins("JOIN access_types ON access_types.id = check_ins.access_type_id").
		Joins("LEFT JOIN users ON users.id = check_ins.user_id").
		Joins("LEFT JOIN day_passes ON day_passes.id = check_ins.day_pass_id").
		Joins("LEFT JOIN auths ON auths.id = check_ins.registered_by_id").
		Order("check_ins.checked_in_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]CheckInSummaryItem, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("check_ins").
		Where("check_ins.user_id = ?", userID)

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []CheckInSummaryItem
	err := dataQuery.
		Select(`
			check_ins.id,
			check_ins.gym_id,
			gyms.name AS gym_name,
			check_ins.access_type_id,
			access_types.name AS access_type_name,
			check_ins.user_id,
			COALESCE(NULLIF(TRIM(CONCAT(users.first_name, ' ', users.last_name)), ''), day_passes.full_name, 'Invitado') AS user_full_name,
			check_ins.day_pass_id,
			check_ins.subscription_id,
			check_ins.registered_by_id,
			COALESCE(auths.email, '') AS registered_by_name,
			check_ins.checked_in_at
		`).
		Joins("JOIN gyms ON gyms.id = check_ins.gym_id").
		Joins("JOIN access_types ON access_types.id = check_ins.access_type_id").
		Joins("LEFT JOIN users ON users.id = check_ins.user_id").
		Joins("LEFT JOIN day_passes ON day_passes.id = check_ins.day_pass_id").
		Joins("LEFT JOIN auths ON auths.id = check_ins.registered_by_id").
		Order("check_ins.checked_in_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetDayPassBySlug(ctx context.Context, slug uuid.UUID) (*models.DayPass, error) {
	var dp models.DayPass
	err := r.db.WithContext(ctx).First(&dp, "slug = ?", slug).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &dp, nil
}

func (r *PostgresRepository) GetSubscriptionBySlug(ctx context.Context, slug uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).First(&sub, "slug = ?", slug).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &sub, nil
}

func (r *PostgresRepository) GetSubscriptionByEmail(ctx context.Context, email string, gymID uint64) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Where("LOWER(auths.email) = LOWER(?) AND plans.gym_id = ?", email, gymID).
		Order("subscriptions.end_date DESC").
		First(&sub).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &sub, nil
}

func (r *PostgresRepository) GetDayPassByEmail(ctx context.Context, email string, gymID uint64) (*models.DayPass, error) {
	var dp models.DayPass
	err := r.db.WithContext(ctx).
		Table("day_passes").
		Where("LOWER(email) = LOWER(?) AND gym_id = ? AND visit_date = CURRENT_DATE AND is_used = false", email, gymID).
		Order("created_at DESC").
		First(&dp).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &dp, nil
}

func (r *PostgresRepository) IsDayPassPaid(ctx context.Context, dayPassID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("payments").
		Where("day_pass_id = ? AND payment_status_id = ?", dayPassID, models.PaymentStatusPaidID).
		Count(&count).Error
	if err != nil {
		return false, helper.TranslateDBError(err)
	}
	return count > 0, nil
}

func (r *PostgresRepository) MarkDayPassAsUsed(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.DayPass{}).
		Where("id = ? AND is_used = ?", id, false).
		Update("is_used", true)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrAlreadyExists
	}

	return nil
}

func (r *PostgresRepository) GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error) {
	var schedules []models.GymSchedule
	err := r.db.WithContext(ctx).Where("gym_id = ?", gymID).Find(&schedules).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return schedules, nil
}

func (r *PostgresRepository) GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error) {
	var p models.Plan
	err := r.db.WithContext(ctx).First(&p, "id = ?", planID).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetUserDetailsByUserID(ctx context.Context, userID uint64) (string, error) {
	var fullName string
	err := r.db.WithContext(ctx).
		Table("users").
		Select("TRIM(CONCAT(first_name, ' ', last_name))").
		Where("id = ?", userID).
		Scan(&fullName).Error

	if err != nil {
		return "", helper.TranslateDBError(err)
	}

	return fullName, nil
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

func (r *PostgresRepository) GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, string, error) {
	var row UserAuthSummary
	err := r.db.WithContext(ctx).
		Table("users").
		Select("id, TRIM(CONCAT(first_name, ' ', last_name)) AS full_name").
		Where("auth_id = ?", authID).
		Scan(&row).Error

	if err != nil {
		return 0, "", helper.TranslateDBError(err)
	}

	if row.ID == 0 {
		return 0, "", errorsx.ErrNotFound
	}

	return row.ID, row.FullName, nil
}
