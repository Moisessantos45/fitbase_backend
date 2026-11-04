package gym

import (
	"context"
	"encoding/json"
	"fitbase/internal/models"
	"fitbase/internal/shared/utils"
	"fmt"
	"strings"
	"time"
)

type GymScheduleItemRequest struct {
	DayOfWeek    int  `json:"day_of_week" binding:"gte=0,lte=6"`
	OpenMinutes  int  `json:"open_minutes" binding:"gte=0,lte=1440"`
	CloseMinutes int  `json:"close_minutes" binding:"gte=0,lte=1440"`
	IsClosed     bool `json:"is_closed"`
}

type SetScheduleRequest struct {
	Schedules []GymScheduleItemRequest `json:"schedules" binding:"required,dive"`
}

type GymScheduleItem struct {
	ID           uint64 `json:"id" gorm:"column:id"`
	GymID        uint64 `json:"gym_id" gorm:"column:gym_id"`
	DayOfWeek    int    `json:"day_of_week" gorm:"column:day_of_week"`
	OpenMinutes  int    `json:"open_minutes" gorm:"column:open_minutes"`
	CloseMinutes int    `json:"close_minutes" gorm:"column:close_minutes"`
	IsClosed     bool   `json:"is_closed" gorm:"column:is_closed"`
}

type SavedFile struct {
	FileID   string `json:"file_id"`
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type SearchGymQuery struct {
	Search    string  `form:"search"`
	Latitude  float64 `form:"lat"`
	Longitude float64 `form:"lng"`
	Page      int     `form:"page"`
	PageSize  int     `form:"page_size"`
}

func (q *SearchGymQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 10
	}
	q.Search = strings.TrimSpace(q.Search)
}

type GymPublicSummary struct {
	ID             uint64   `json:"id"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Phone          string   `json:"phone"`
	Email          string   `json:"email"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	IsOpen         bool     `json:"is_open"`
	BackgroundSlug string   `json:"-" gorm:"column:background_slug"`
	BackgroundURL  string   `json:"background_url" gorm:"-"`
	DistanceMeters *float64 `json:"distance_meters,omitempty" gorm:"column:distance_meters"`
}

type GymPhotoDetail struct {
	ID   uint64 `json:"id"`
	Slug string `json:"-"`
	URL  string `json:"url"`
}

func (p *GymPhotoDetail) UnmarshalJSON(data []byte) error {
	type Alias GymPhotoDetail
	aux := &struct {
		Slug string `json:"slug"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	p.Slug = aux.Slug
	return nil
}

type GymDetailResponse struct {
	ID uint64 `json:"id" gorm:"column:id"`
	// OwnerID        uint64            `json:"owner_id" gorm:"column:owner_id"`
	// BackgroundID   *uint64           `json:"background_id" gorm:"column:background_id"`
	BackgroundSlug string            `json:"-" gorm:"column:background_slug"`
	BackgroundURL  string            `json:"background_url" gorm:"-"`
	Name           string            `json:"name" gorm:"column:name"`
	Address        string            `json:"address" gorm:"column:address"`
	Phone          string            `json:"phone" gorm:"column:phone"`
	Email          string            `json:"email" gorm:"column:email"`
	Latitude       float64           `json:"latitude" gorm:"column:latitude"`
	Longitude      float64           `json:"longitude" gorm:"column:longitude"`
	IsOpen         bool              `json:"is_open" gorm:"-"`
	CreatedAt      time.Time         `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time         `json:"updated_at" gorm:"column:updated_at"`
	SchedulesRaw   []byte            `json:"-" gorm:"column:schedules_raw"`
	Schedules      []GymScheduleItem `json:"schedules" gorm:"-"`
	PhotosRaw      []byte            `json:"-" gorm:"column:photos_raw"`
	Photos         []GymPhotoDetail  `json:"photos" gorm:"-"`
}

type PaginatedGymSummary struct {
	Data     []GymPublicSummary `json:"data"`
	Paginate models.Pagination  `json:"paginate"`
}

type UpdateGymScheduleRequest struct {
	DayOfWeek    *int  `json:"day_of_week" binding:"omitempty,gte=0,lte=6"`
	OpenMinutes  *int  `json:"open_minutes" binding:"omitempty,gte=0,lte=1440"`
	CloseMinutes *int  `json:"close_minutes" binding:"omitempty,gte=0,lte=1440"`
	IsClosed     *bool `json:"is_closed"`
}

func BuildScheduleUpdateData(req UpdateGymScheduleRequest) map[string]any {
	data := make(map[string]any)
	if req.DayOfWeek != nil {
		data["day_of_week"] = *req.DayOfWeek
	}
	if req.OpenMinutes != nil {
		data["open_minutes"] = *req.OpenMinutes
	}
	if req.CloseMinutes != nil {
		data["close_minutes"] = *req.CloseMinutes
	}
	if req.IsClosed != nil {
		data["is_closed"] = *req.IsClosed
	}
	return data
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	Create(ctx context.Context, gym *models.Gym) error
	CreateSchedule(ctx context.Context, schedule *models.GymSchedule) error
	CreateSchedules(ctx context.Context, schedules []models.GymSchedule) error
	CreateFile(ctx context.Context, file *models.File) error
	CreatePhoto(ctx context.Context, photo *models.GymPhoto) error
	CreateGymPhoto(ctx context.Context, photo *models.GymPhoto) error
	GetByID(ctx context.Context, id uint64) (*models.Gym, error)
	GetDetailsByID(ctx context.Context, id uint64) (*GymDetailResponse, error)
	GetDetailsByAuthID(ctx context.Context, authID uint64) (*GymDetailResponse, error)
	GetAllByOwnerID(ctx context.Context, ownerID uint64, offset, limit int) ([]GymPublicSummary, int64, error)
	GetAllPublic(ctx context.Context, query string, lat, lng float64, offset, limit int) ([]GymPublicSummary, int64, error)
	Update(ctx context.Context, id uint64, data map[string]any) error
	UpsertSchedule(ctx context.Context, schedule *models.GymSchedule) error
	GetScheduleByID(ctx context.Context, id uint64) (*models.GymSchedule, error)
	UpdateSchedule(ctx context.Context, id uint64, data map[string]any) error
	GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]GymScheduleItem, error)
	GetPhotoByID(ctx context.Context, id uint64) (*models.GymPhoto, error)
	GetFileByID(ctx context.Context, id uint64) (*models.File, error)
	DeletePhoto(ctx context.Context, id uint64) error
	DeleteFile(ctx context.Context, id uint64) error

	GetFirstPhotoByGymID(ctx context.Context, gymID uint64) (*models.GymPhoto, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetGymIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	IsGymStaff(ctx context.Context, authID, gymID uint64) (bool, error)
}

type Service interface {
	Create(ctx context.Context, authID uint64, name, address, phone, email string, lat, lng float64, schedules []GymScheduleItemRequest, cover *SavedFile, photos []SavedFile) (*models.Gym, error)
	UploadCover(ctx context.Context, authID, gymID uint64, slug, path, mimeType string, size int64) (string, error)
	Update(ctx context.Context, authID, gymID uint64, name, address, phone, email string, lat, lng *float64) error
	GetDetailsByID(ctx context.Context, id uint64) (*GymDetailResponse, error)
	GetMyGymDetails(ctx context.Context, authID uint64) (*GymDetailResponse, error)
	GetAllByOwner(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedGymSummary, error)
	GetAllPublic(ctx context.Context, query string, lat, lng float64, page, pageSize int) (*PaginatedGymSummary, error)
	SetSchedules(ctx context.Context, authID, gymID uint64, schedules []GymScheduleItemRequest) error
	UpdateSchedule(ctx context.Context, authID, gymID, scheduleID uint64, req UpdateGymScheduleRequest) error
	UploadPhotos(ctx context.Context, authID, gymID uint64, photos []SavedFile) error
	DeletePhoto(ctx context.Context, authID, gymID, photoID uint64) error
}

func New(ownerID uint64, name, address, phone, email string, lat, lng float64) (*models.Gym, error) {
	if ownerID == 0 {
		return nil, fmt.Errorf("invalid ownerID: %d", ownerID)
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("invalid name")
	}

	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		return nil, fmt.Errorf("invalid address")
	}

	trimmedPhone := strings.TrimSpace(phone)
	if trimmedPhone == "" {
		return nil, fmt.Errorf("invalid phone")
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" || !utils.EmailRegex.MatchString(trimmedEmail) {
		return nil, fmt.Errorf("invalid email")
	}

	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("invalid latitude: %f", lat)
	}

	if lng < -180 || lng > 180 {
		return nil, fmt.Errorf("invalid longitude: %f", lng)
	}

	return &models.Gym{
		OwnerID:   ownerID,
		Name:      trimmedName,
		Address:   trimmedAddress,
		Phone:     trimmedPhone,
		Email:     trimmedEmail,
		Latitude:  lat,
		Longitude: lng,
	}, nil
}

func BuildGymUpdateData(data *models.Gym) map[string]any {
	updateData := make(map[string]any)

	trimmedName := strings.TrimSpace(data.Name)
	if trimmedName != "" {
		updateData["name"] = trimmedName
	}

	trimmedAddress := strings.TrimSpace(data.Address)
	if trimmedAddress != "" {
		updateData["address"] = trimmedAddress
	}

	trimmedPhone := strings.TrimSpace(data.Phone)
	if trimmedPhone != "" {
		updateData["phone"] = trimmedPhone
	}

	trimmedEmail := strings.TrimSpace(data.Email)
	if trimmedEmail != "" {
		updateData["email"] = trimmedEmail
	}

	if data.Latitude != 0 {
		updateData["latitude"] = data.Latitude
	}

	if data.Longitude != 0 {
		updateData["longitude"] = data.Longitude
	}

	return updateData
}
