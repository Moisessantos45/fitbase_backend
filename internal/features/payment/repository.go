package payment

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

func (r *PostgresRepository) WithTransaction(fn func(repo Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) Create(ctx context.Context, payment *models.Payment) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(payment).Error)
}

func (r *PostgresRepository) GetTotalRevenueByGymID(ctx context.Context, gymID uint64) (float64, int64, error) {
	var res PaymentRevenueSummary
	err := r.db.WithContext(ctx).
		Table("payments").
		Select("COALESCE(SUM(amount), 0) AS total_revenue, COUNT(id) AS total_transactions").
		Where("gym_id = ? AND payment_status_id = ?", gymID, models.PaymentStatusPaidID).
		Scan(&res).Error

	if err != nil {
		return 0, 0, helper.TranslateDBError(err)
	}

	return res.TotalRevenue, res.TotalTransactions, nil
}

func (r *PostgresRepository) GetAllByGymID(ctx context.Context, gymID uint64, offset, limit int) ([]PaymentListItem, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Model(&models.Payment{}).
		Where("gym_id = ?", gymID)

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []PaymentListItem
	err := dataQuery.
		Table("payments").
		Select(`
			payments.id,
			payments.gym_id,
			gyms.name AS gym_name,
			payments.payment_method_id,
			payment_methods.name AS payment_method_name,
			payments.payment_status_id,
			payment_statuses.name AS payment_status_name,
			payments.amount,
			payments.subscription_id,
			payments.day_pass_id,
			payments.cash_register_id,
			payments.payment_date,
			payments.created_at
		`).
		Joins("JOIN gyms ON gyms.id = payments.gym_id").
		Joins("JOIN payment_methods ON payment_methods.id = payments.payment_method_id").
		Joins("JOIN payment_statuses ON payment_statuses.id = payments.payment_status_id").
		Order("payments.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]PaymentListItem, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("payments").
		Joins("LEFT JOIN subscriptions ON subscriptions.id = payments.subscription_id").
		Joins("LEFT JOIN day_passes ON day_passes.id = payments.day_pass_id").
		Where("subscriptions.user_id = ? OR day_passes.user_id = ?", userID, userID)

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []PaymentListItem
	err := dataQuery.
		Select(`
			payments.id,
			payments.gym_id,
			gyms.name AS gym_name,
			payments.payment_method_id,
			payment_methods.name AS payment_method_name,
			payments.payment_status_id,
			payment_statuses.name AS payment_status_name,
			payments.amount,
			payments.subscription_id,
			payments.day_pass_id,
			payments.cash_register_id,
			payments.payment_date,
			payments.created_at
		`).
		Joins("JOIN gyms ON gyms.id = payments.gym_id").
		Joins("JOIN payment_methods ON payment_methods.id = payments.payment_method_id").
		Joins("JOIN payment_statuses ON payment_statuses.id = payments.payment_status_id").
		Order("payments.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetActiveCashRegisterByAuthID(ctx context.Context, authID, gymID uint64) (*models.CashRegister, error) {
	var register models.CashRegister
	err := r.db.WithContext(ctx).
		Where("auth_id = ? AND gym_id = ? AND close_time IS NULL", authID, gymID).
		First(&register).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return &register, nil
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

func (r *PostgresRepository) GetGymName(ctx context.Context, gymID uint64) (string, error) {
	var name string
	err := r.db.WithContext(ctx).
		Model(&models.Gym{}).
		Select("name").
		Where("id = ?", gymID).
		Scan(&name).Error

	if err != nil {
		return "", helper.TranslateDBError(err)
	}

	if name == "" {
		return "", errorsx.ErrNotFound
	}

	return name, nil
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

func (r *PostgresRepository) GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var userID uint64
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("id").
		Where("auth_id = ?", authID).
		Scan(&userID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if userID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return userID, nil
}

func (r *PostgresRepository) GetPaymentMethodByID(ctx context.Context, methodID uint64) (*models.PaymentMethod, error) {
	var method models.PaymentMethod
	err := r.db.WithContext(ctx).First(&method, "id = ?", methodID).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &method, nil
}

func (r *PostgresRepository) GetReceiptByID(ctx context.Context, paymentID uint64) (*PaymentReceiptResponse, error) {
	var receipt PaymentReceiptResponse
	err := r.db.WithContext(ctx).
		Table("payments").
		Select(`
			payments.id,
			payments.gym_id,
			gyms.name AS gym_name,
			payments.payment_method_id,
			payment_methods.name AS payment_method_name,
			payments.payment_status_id,
			payment_statuses.name AS payment_status_name,
			payments.amount,
			payments.subscription_id,
			payments.day_pass_id,
			payments.cash_register_id,
			payments.payment_date,
			payments.created_at
		`).
		Joins("JOIN gyms ON gyms.id = payments.gym_id").
		Joins("JOIN payment_methods ON payment_methods.id = payments.payment_method_id").
		Joins("JOIN payment_statuses ON payment_statuses.id = payments.payment_status_id").
		Where("payments.id = ?", paymentID).
		Scan(&receipt).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if receipt.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &receipt, nil
}

func (r *PostgresRepository) GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error) {
	var plan models.Plan
	err := r.db.WithContext(ctx).First(&plan, "id = ?", planID).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &plan, nil
}

func (r *PostgresRepository) GetPaymentByID(ctx context.Context, paymentID uint64) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).First(&payment, "id = ?", paymentID).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &payment, nil
}

func (r *PostgresRepository) Update(ctx context.Context, paymentID uint64, updates map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Updates(updates)
	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(sub).Error)
}

func (r *PostgresRepository) UpdateSubscription(ctx context.Context, subID uint64, updates map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ?", subID).
		Updates(updates)
	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateDayPass(ctx context.Context, dp *models.DayPass) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(dp).Error)
}

func (r *PostgresRepository) GetDayPassDetailsByID(ctx context.Context, dayPassID uint64) (*DayPassPaymentDetails, error) {
	var details DayPassPaymentDetails
	err := r.db.WithContext(ctx).
		Table("day_passes").
		Select(`
			day_passes.id,
			day_passes.slug::text AS slug,
			day_passes.gym_id,
			gyms.name AS gym_name,
			day_passes.plan_id,
			plans.name AS plan_name,
			day_passes.full_name,
			day_passes.email,
			COALESCE(TO_CHAR(day_passes.visit_date, 'YYYY-MM-DD'), '') AS visit_date,
			day_passes.is_used
		`).
		Joins("JOIN gyms ON gyms.id = day_passes.gym_id").
		Joins("JOIN plans ON plans.id = day_passes.plan_id").
		Where("day_passes.id = ?", dayPassID).
		Scan(&details).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if details.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &details, nil
}

func (r *PostgresRepository) GetSubscriptionDetailsByID(ctx context.Context, subID uint64) (*SubscriptionPaymentDetails, error) {
	var details SubscriptionPaymentDetails
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
			COALESCE(TO_CHAR(subscriptions.start_date, 'YYYY-MM-DD'), '') AS start_date,
			COALESCE(TO_CHAR(subscriptions.end_date, 'YYYY-MM-DD'), '') AS end_date,
			subscriptions.auto_renew
		`).
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("subscriptions.id = ?", subID).
		Scan(&details).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if details.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &details, nil
}

func (r *PostgresRepository) GetUserProfileByAuthID(ctx context.Context, authID uint64) (*UserAuthProfile, error) {
	var profile UserAuthProfile
	err := r.db.WithContext(ctx).
		Table("users").
		Select(`
			users.id,
			users.auth_id,
			users.first_name,
			users.last_name,
			TRIM(CONCAT(users.first_name, ' ', users.last_name)) AS full_name,
			users.phone,
			auths.email
		`).
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("users.auth_id = ?", authID).
		Scan(&profile).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if profile.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &profile, nil
}

func (r *PostgresRepository) GetGymByID(ctx context.Context, gymID uint64) (*models.Gym, error) {
	var gym models.Gym
	if err := r.db.WithContext(ctx).
		First(&gym, "id = ?", gymID).
		Error; err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return &gym, nil
}

func (r *PostgresRepository) GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error) {
	var schedules []models.GymSchedule
	err := r.db.WithContext(ctx).Where("gym_id = ?", gymID).Find(&schedules).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return schedules, nil
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
