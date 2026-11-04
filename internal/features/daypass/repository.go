package daypass

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"strings"
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

func (r *PostgresRepository) Create(ctx context.Context, dayPass *models.DayPass) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(dayPass).Error)
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug uuid.UUID) (*models.DayPass, error) {
	var dp models.DayPass
	err := r.db.WithContext(ctx).First(&dp, "slug = ?", slug).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &dp, nil
}

func (r *PostgresRepository) GetTicketBySlug(ctx context.Context, slug uuid.UUID) (*DayPassTicketResponse, error) {
	var ticket DayPassTicketResponse
	err := r.db.WithContext(ctx).
		Table("day_passes").
		Select(`
			day_passes.id,
			day_passes.slug::text AS slug,
			day_passes.gym_id,
			gyms.name AS gym_name,
			day_passes.plan_id,
			plans.name AS plan_name,
			COALESCE(plans.price, 0) AS plan_price,
			CASE WHEN EXISTS (
				SELECT 1 FROM payments p WHERE p.day_pass_id = day_passes.id AND p.payment_status_id = 2
			) THEN true ELSE false END AS is_paid,
			day_passes.full_name,
			day_passes.phone,
			day_passes.email,
			COALESCE(TO_CHAR(day_passes.visit_date, 'YYYY-MM-DD'), '') AS visit_date,
			day_passes.is_used,
			day_passes.created_at
		`).
		Joins("JOIN gyms ON gyms.id = day_passes.gym_id").
		Joins("JOIN plans ON plans.id = day_passes.plan_id").
		Where("day_passes.slug = ?", slug).
		Scan(&ticket).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if ticket.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &ticket, nil
}

func (r *PostgresRepository) MarkAsUsed(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.DayPass{}).
		Where("id = ? AND is_used = ?", id, false).
		Update("is_used", true)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) GetAllByGymID(ctx context.Context, gymID uint64, date time.Time, search string, offset, limit int) ([]DayPassDetailResponse, int64, error) {
	baseQuery := r.db.WithContext(ctx).
		Table("day_passes").
		Where("day_passes.gym_id = ?", gymID)

	if !date.IsZero() {
		baseQuery = baseQuery.Where("day_passes.visit_date = ?", date.Format("2006-01-02"))
	}

	trimmedSearch := strings.TrimSpace(search)
	if trimmedSearch != "" {
		term := "%" + strings.ToLower(trimmedSearch) + "%"
		baseQuery = baseQuery.Where(
			"(LOWER(day_passes.full_name) LIKE ? OR LOWER(day_passes.email) LIKE ? OR day_passes.phone LIKE ? OR day_passes.slug::text LIKE ?)",
			term, term, term, term,
		)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var results []DayPassDetailResponse
	err := dataQuery.
		Select(`
			day_passes.id,
			day_passes.slug::text AS slug,
			day_passes.user_id,
			day_passes.gym_id,
			gyms.name AS gym_name,
			day_passes.plan_id,
			plans.name AS plan_name,
			COALESCE(plans.price, 0) AS plan_price,
			CASE WHEN EXISTS (
				SELECT 1 FROM payments p WHERE p.day_pass_id = day_passes.id AND p.payment_status_id = 2
			) THEN true ELSE false END AS is_paid,
			day_passes.full_name,
			day_passes.phone,
			day_passes.email,
			COALESCE(TO_CHAR(day_passes.visit_date, 'YYYY-MM-DD'), '') AS visit_date,
			day_passes.is_used,
			day_passes.created_at
		`).
		Joins("JOIN gyms ON gyms.id = day_passes.gym_id").
		Joins("JOIN plans ON plans.id = day_passes.plan_id").
		Order("day_passes.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error) {
	var schedules []models.GymSchedule
	err := r.db.WithContext(ctx).
		Where("gym_id = ?", gymID).
		Find(&schedules).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return schedules, nil
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
