package gym

import (
	"context"
	"encoding/json"
	"errors"
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

func (r *PostgresRepository) Create(ctx context.Context, gym *models.Gym) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(gym).Error)
}

func (r *PostgresRepository) CreateSchedule(ctx context.Context, schedule *models.GymSchedule) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(schedule).Error)
}

func (r *PostgresRepository) CreateSchedules(ctx context.Context, schedules []models.GymSchedule) error {
	if len(schedules) == 0 {
		return nil
	}
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(&schedules).Error)
}

func (r *PostgresRepository) CreateFile(ctx context.Context, file *models.File) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(file).Error)
}

func (r *PostgresRepository) CreatePhoto(ctx context.Context, photo *models.GymPhoto) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Create(photo).Error)
}

func (r *PostgresRepository) CreateGymPhoto(ctx context.Context, photo *models.GymPhoto) error {
	return r.CreatePhoto(ctx, photo)
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uint64) (*models.Gym, error) {
	var gym models.Gym
	err := r.db.WithContext(ctx).First(&gym, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &gym, nil
}

func (r *PostgresRepository) getDetailsWhere(ctx context.Context, whereClause string, args ...any) (*GymDetailResponse, error) {
	var gym GymDetailResponse
	err := r.db.WithContext(ctx).
		Table("gyms").
		Select(`
			gyms.id,
			-- gyms.owner_id,
			-- gyms.background_id,
			COALESCE(f_bg.slug, '') AS background_slug,
			gyms.name,
			gyms.address,
			gyms.phone,
			gyms.email,
			gyms.latitude,
			gyms.longitude,
			gyms.created_at,
			gyms.updated_at,
			COALESCE(
				(
					SELECT jsonb_agg(
						jsonb_build_object(
							'id', gs.id,
							'gym_id', gs.gym_id,
							'day_of_week', gs.day_of_week,
							'open_minutes', gs.open_minutes,
							'close_minutes', gs.close_minutes,
							'is_closed', gs.is_closed
						)
					)
					FROM gym_schedules gs
					WHERE gs.gym_id = gyms.id
				),
				'[]'::jsonb
			) AS schedules_raw,
			COALESCE(
				(
					SELECT jsonb_agg(
						jsonb_build_object(
							'id', gp.id,
							'slug', f.slug
						)
					)
					FROM gym_photos gp
					LEFT JOIN files f ON f.id = gp.file_id
					WHERE gp.gym_id = gyms.id
				),
				'[]'::jsonb
			) AS photos_raw
		`).
		Joins("LEFT JOIN files f_bg ON f_bg.id = gyms.background_id").
		Where(whereClause, args...).
		Scan(&gym).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if gym.ID == 0 {
		return nil, errorsx.ErrNotFound
	}

	if len(gym.SchedulesRaw) > 0 && string(gym.SchedulesRaw) != "null" {
		var rawSchedules []GymScheduleItem
		if err := json.Unmarshal(gym.SchedulesRaw, &rawSchedules); err != nil {
			return nil, err
		}
		gym.Schedules = rawSchedules
	} else {
		gym.Schedules = []GymScheduleItem{}
	}

	if len(gym.PhotosRaw) > 0 && string(gym.PhotosRaw) != "null" {
		var rawPhotos []GymPhotoDetail
		if err := json.Unmarshal(gym.PhotosRaw, &rawPhotos); err != nil {
			return nil, err
		}
		gym.Photos = rawPhotos
	} else {
		gym.Photos = []GymPhotoDetail{}
	}

	return &gym, nil
}

func (r *PostgresRepository) GetDetailsByID(ctx context.Context, id uint64) (*GymDetailResponse, error) {
	return r.getDetailsWhere(ctx, "gyms.id = ?", id)
}

func (r *PostgresRepository) GetDetailsByAuthID(ctx context.Context, authID uint64) (*GymDetailResponse, error) {
	return r.getDetailsWhere(ctx, `gyms.id = COALESCE(
		(SELECT gm.gym_id FROM gym_managers gm WHERE gm.auth_id = ? LIMIT 1),
		(SELECT g2.id FROM gyms g2 INNER JOIN owners o ON o.id = g2.owner_id WHERE o.auth_id = ? ORDER BY g2.id ASC LIMIT 1)
	)`, authID, authID)
}

func (r *PostgresRepository) GetAllByOwnerID(ctx context.Context, ownerID uint64, offset, limit int) ([]GymPublicSummary, int64, error) {
	baseQuery := r.db.WithContext(ctx).Table("gyms").Where("gyms.owner_id = ?", ownerID)

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var list []GymPublicSummary
	err := dataQuery.
		Joins("LEFT JOIN files f_bg ON f_bg.id = gyms.background_id").
		Select(`
			gyms.id,
			gyms.name,
			gyms.address,
			gyms.phone,
			gyms.email,
			gyms.latitude,
			gyms.longitude,
			COALESCE(f_bg.slug, '') AS background_slug
		`).
		Order("gyms.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&list).Error

	if err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return list, total, nil
}

func (r *PostgresRepository) GetAllPublic(ctx context.Context, query string, lat, lng float64, offset, limit int) ([]GymPublicSummary, int64, error) {
	baseQuery := r.db.WithContext(ctx).Table("gyms")

	if query != "" {
		baseQuery = baseQuery.Where("gyms.name ILIKE ? OR gyms.address ILIKE ?", "%"+query+"%", "%"+query+"%")
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	dataQuery := baseQuery.Session(&gorm.Session{})

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	var list []GymPublicSummary
	if lat != 0 || lng != 0 {
		err := dataQuery.
			Joins("LEFT JOIN files f_bg ON f_bg.id = gyms.background_id").
			Select(`
			gyms.id,
			gyms.name,
			gyms.address,
			gyms.phone,
			gyms.email,
			gyms.latitude,
			gyms.longitude,
			COALESCE(f_bg.slug, '') AS background_slug,
			ROUND(ST_Distance(
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
				ST_SetSRID(ST_MakePoint(gyms.longitude, gyms.latitude), 4326)::geography
			)::numeric, 2) AS distance_meters
		`, lng, lat).
			Order("distance_meters ASC").
			Offset(offset).
			Limit(limit).
			Scan(&list).Error

		if err != nil {
			return nil, 0, helper.TranslateDBError(err)
		}
	} else {
		err := dataQuery.
			Joins("LEFT JOIN files f_bg ON f_bg.id = gyms.background_id").
			Select(`
			gyms.id,
			gyms.name,
			gyms.address,
			gyms.phone,
			gyms.email,
			gyms.latitude,
			gyms.longitude,
			COALESCE(f_bg.slug, '') AS background_slug
		`).
			Order("gyms.created_at DESC").
			Offset(offset).
			Limit(limit).
			Scan(&list).Error

		if err != nil {
			return nil, 0, helper.TranslateDBError(err)
		}
	}

	return list, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id uint64, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.Gym{}).
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

func (r *PostgresRepository) UpsertSchedule(ctx context.Context, schedule *models.GymSchedule) error {
	var existing models.GymSchedule
	err := r.db.WithContext(ctx).
		Where("gym_id = ? AND day_of_week = ?", schedule.GymID, schedule.DayOfWeek).
		First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.TranslateDBError(r.db.WithContext(ctx).Create(schedule).Error)
		}
		return helper.TranslateDBError(err)
	}

	return helper.TranslateDBError(r.db.WithContext(ctx).
		Model(&existing).
		Updates(map[string]any{
			"open_minutes":  schedule.OpenMinutes,
			"close_minutes": schedule.CloseMinutes,
			"is_closed":     schedule.IsClosed,
		}).Error)
}

func (r *PostgresRepository) GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]GymScheduleItem, error) {
	var items []GymScheduleItem
	err := r.db.WithContext(ctx).
		Model(&models.GymSchedule{}).
		Where("gym_id = ?", gymID).
		Order("day_of_week ASC").
		Scan(&items).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return items, nil
}

func (r *PostgresRepository) GetScheduleByID(ctx context.Context, id uint64) (*models.GymSchedule, error) {
	var schedule models.GymSchedule
	err := r.db.WithContext(ctx).First(&schedule, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &schedule, nil
}

func (r *PostgresRepository) UpdateSchedule(ctx context.Context, id uint64, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.GymSchedule{}).
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

func (r *PostgresRepository) GetPhotoByID(ctx context.Context, id uint64) (*models.GymPhoto, error) {
	var photo models.GymPhoto
	err := r.db.WithContext(ctx).First(&photo, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &photo, nil
}

func (r *PostgresRepository) GetFileByID(ctx context.Context, id uint64) (*models.File, error) {
	var file models.File
	err := r.db.WithContext(ctx).First(&file, "id = ?", id).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &file, nil
}

func (r *PostgresRepository) DeletePhoto(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.GymPhoto{}, "id = ?", id)
	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteFile(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.File{}, "id = ?", id)
	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetFirstPhotoByGymID(ctx context.Context, gymID uint64) (*models.GymPhoto, error) {
	var photo models.GymPhoto
	err := r.db.WithContext(ctx).
		Where("gym_id = ?", gymID).
		Order("id ASC").
		First(&photo).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}
	return &photo, nil
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

func (r *PostgresRepository) GetGymIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var gymID uint64

	// 1. Intentar como encargado (GymManager)
	err := r.db.WithContext(ctx).
		Model(&models.GymManager{}).
		Select("gym_id").
		Where("auth_id = ?", authID).
		Scan(&gymID).Error

	if err == nil && gymID > 0 {
		return gymID, nil
	}

	// 2. Intentar como dueño (Owner)
	err = r.db.WithContext(ctx).
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

func (r *PostgresRepository) IsGymStaff(ctx context.Context, authID, gymID uint64) (bool, error) {
	var isStaff bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM gyms g
			JOIN owners o ON o.id = g.owner_id
			WHERE g.id = ? AND o.auth_id = ?
			UNION ALL
			SELECT 1 FROM gym_managers gm
			WHERE gm.gym_id = ? AND gm.auth_id = ?
		)
	`
	err := r.db.WithContext(ctx).Raw(query, gymID, authID, gymID, authID).Scan(&isStaff).Error
	if err != nil {
		return false, helper.TranslateDBError(err)
	}
	return isStaff, nil
}

