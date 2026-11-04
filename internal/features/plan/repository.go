package plan

import (
	"context"
	"encoding/json"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fmt"
	"sort"

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

func (r *PostgresRepository) CreateWithItems(ctx context.Context, plan *models.Plan, items []models.PlanItem) error {
	return r.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.db.WithContext(ctx).Create(plan).Error; err != nil {
			return helper.TranslateDBError(err)
		}

		if len(items) > 0 {
			for i := range items {
				items[i].PlanID = plan.ID
			}
			if err := txRepo.db.WithContext(ctx).Create(&items).Error; err != nil {
				return helper.TranslateDBError(err)
			}
		}

		return nil
	})
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uint64) (*models.Plan, error) {
	var plan models.Plan
	err := r.db.WithContext(ctx).First(&plan, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &plan, nil
}

func (r *PostgresRepository) GetDetailsByID(ctx context.Context, id uint64) (*PlanDetailResponse, error) {
	var plan PlanDetailResponse
	err := r.db.WithContext(ctx).
		Table("plans").
		Select(`
			plans.id,
			plans.gym_id,
			gyms.name AS gym_name,
			plans.name,
			plans.description,
			plans.price,
			plans.discount,
			(plans.price * (1 - (plans.discount / 100))) AS final_price,
			plans.duration,
			plans.plan_type,
			plans.max_visits,
			plans.active,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'id', pi.id,
						'name', pi.name,
						'description', pi.description
					)
				) FILTER (WHERE pi.id IS NOT NULL),
				'[]'::jsonb
			) AS items
		`).
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("LEFT JOIN plan_items pi ON pi.plan_id = plans.id").
		Where("plans.id = ?", id).
		Group(`
			plans.id,
			plans.gym_id,
			gyms.name,
			plans.name,
			plans.description,
			plans.price,
			plans.discount,
			plans.duration,
			plans.plan_type,
			plans.max_visits,
			plans.active
		`).
		Scan(&plan).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if plan.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	if len(plan.ItemsRaw) > 0 {
		if err := json.Unmarshal(plan.ItemsRaw, &plan.Items); err != nil {
			return nil, err
		}
		sort.Slice(plan.Items, func(i, j int) bool {
			return plan.Items[i].ID < plan.Items[j].ID
		})
	} else {
		plan.Items = []PlanPublicItem{}
	}

	return &plan, nil
}

func (r *PostgresRepository) GetAllByGymID(
	ctx context.Context,
	gymID uint64,
	activeOnly bool,
) ([]PlanDetailResponse, error) {
	query := r.db.WithContext(ctx).
		Table("plans").
		Select(`
			plans.id,
			plans.gym_id,
			gyms.name AS gym_name,
			plans.name,
			plans.description,
			plans.price,
			plans.discount,
			(plans.price * (1 - (plans.discount / 100.0))) AS final_price,
			plans.duration,
			plans.plan_type,
			plans.max_visits,
			plans.active,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'id', pi.id,
						'name', pi.name,
						'description', pi.description
					)
				) FILTER (WHERE pi.id IS NOT NULL),
				'[]'::jsonb
			) AS items
		`).
		Joins("JOIN gyms ON gyms.id = plans.gym_id").
		Joins("LEFT JOIN plan_items pi ON pi.plan_id = plans.id").
		Where("plans.gym_id = ?", gymID)

	if activeOnly {
		query = query.Where("plans.active = ?", true)
	}

	query = query.
		Group("plans.id, gyms.name").
		Order("plans.price ASC")

	var plans []PlanDetailResponse
	if err := query.Scan(&plans).Error; err != nil {
		return nil, helper.TranslateDBError(err)
	}

	for i := range plans {
		if len(plans[i].ItemsRaw) == 0 {
			plans[i].Items = []PlanPublicItem{}
			continue
		}

		if err := json.Unmarshal(plans[i].ItemsRaw, &plans[i].Items); err != nil {
			return nil, fmt.Errorf(
				"unmarshal items for plan %d: %w",
				plans[i].ID,
				err,
			)
		}

		sort.Slice(plans[i].Items, func(a, b int) bool {
			return plans[i].Items[a].ID < plans[i].Items[b].ID
		})
	}

	return plans, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id uint64, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.Plan{}).
		Where("id = ?", id).
		Updates(data)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
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

func (r *PostgresRepository) CreateItem(ctx context.Context, item *models.PlanItem) error {
	err := r.db.WithContext(ctx).Create(item).Error
	if err != nil {
		return helper.TranslateDBError(err)
	}
	return nil
}

func (r *PostgresRepository) GetItemByID(ctx context.Context, id uint64) (*models.PlanItem, error) {
	var item models.PlanItem
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &item, nil
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, id uint64, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.PlanItem{}).
		Where("id = ?", id).
		Updates(data)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) DeleteItem(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.PlanItem{})

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
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

func (r *PostgresRepository) GetEnrollmentPlans(ctx context.Context, authID, gymID uint64) (*EnrollmentResponse, error) {
	query := `
		SELECT
			g.id AS gym_id,
			g.name AS gym_name,
			CASE WHEN active_sub.id IS NOT NULL THEN true ELSE false END AS has_active_subscription,
			CASE WHEN active_sub.id IS NOT NULL THEN
				jsonb_build_object(
					'id', active_sub.id,
					'slug', active_sub.slug::text,
					'plan_id', active_sub.plan_id,
					'plan_name', active_sub.plan_name,
					'start_date', COALESCE(TO_CHAR(active_sub.start_date, 'YYYY-MM-DD'), ''),
					'end_date', COALESCE(TO_CHAR(active_sub.end_date, 'YYYY-MM-DD'), ''),
					'remaining_days', GREATEST(0, (active_sub.end_date - CURRENT_DATE) + 1),
					'status_name', active_sub.status_name,
					'auto_renew', active_sub.auto_renew
				)
			ELSE NULL END AS active_subscription,
			COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'id', p.id,
						'gym_id', p.gym_id,
						'gym_name', g.name,
						'name', p.name,
						'description', p.description,
						'price', p.price,
						'discount', p.discount,
						'final_price', ROUND((p.price * (1 - (p.discount / 100.0)))::numeric, 2),
						'duration', p.duration,
						'plan_type', p.plan_type,
						'max_visits', p.max_visits,
						'active', p.active,
						'is_current', CASE WHEN active_sub.plan_id = p.id THEN true ELSE false END,
						'items', COALESCE((
							SELECT jsonb_agg(
								jsonb_build_object(
									'id', pi.id,
									'name', pi.name,
									'description', pi.description
								)
								ORDER BY pi.id ASC
							)
							FROM plan_items pi
							WHERE pi.plan_id = p.id
						), '[]'::jsonb)
					)
					ORDER BY p.price ASC
				)
				FROM plans p
				WHERE p.gym_id = g.id AND p.active = true
			), '[]'::jsonb) AS plans
		FROM gyms g
		LEFT JOIN (
			SELECT
				s.id,
				s.slug,
				s.plan_id,
				p.name AS plan_name,
				s.start_date,
				s.end_date,
				ss.name AS status_name,
				s.auto_renew,
				p.gym_id
			FROM subscriptions s
			JOIN users u ON u.id = s.user_id
			JOIN plans p ON p.id = s.plan_id
			JOIN status_subscriptions ss ON ss.id = s.status_id
			WHERE u.auth_id = ? AND p.gym_id = ? AND s.status_id IN (?, ?) AND s.end_date >= CURRENT_DATE
			ORDER BY s.end_date DESC
			LIMIT 1
		) active_sub ON active_sub.gym_id = g.id
		WHERE g.id = ?
	`

	var resp EnrollmentResponse
	err := r.db.WithContext(ctx).Raw(query, authID, gymID, models.StatusSubscriptionActiveID, models.StatusSubscriptionInactiveID, gymID).Scan(&resp).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if resp.GymID == 0 {
		return nil, errorsx.ErrNotFound
	}

	if resp.Plans == nil {
		resp.Plans = EnrollmentPlanList{}
	}

	return &resp, nil
}
