package checkin

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ValidateAccessRequest struct {
	GymID uint64 `json:"gym_id" binding:"required"`
	Slug  string `json:"slug"`
	Email string `json:"email"`
}

type AccessResultResponse struct {
	Allowed     bool      `json:"allowed"`
	Message     string    `json:"message"`
	AccessType  string    `json:"access_type"`
	FullName    string    `json:"full_name"`
	PlanName    string    `json:"plan_name"`
	CheckedInAt time.Time `json:"checked_in_at"`
}

type CheckInSummaryItem struct {
	ID               uint64    `json:"id"`
	GymID            uint64    `json:"gym_id"`
	GymName          string    `json:"gym_name"`
	AccessTypeID     uint64    `json:"access_type_id"`
	AccessTypeName   string    `json:"access_type_name"`
	UserID           *uint64   `json:"user_id"`
	UserFullName     string    `json:"user_full_name"`
	DayPassID        *uint64   `json:"day_pass_id"`
	SubscriptionID   *uint64   `json:"subscription_id"`
	RegisteredByID   *uint64   `json:"registered_by_id"`
	RegisteredByName string    `json:"registered_by_name"`
	CheckedInAt      time.Time `json:"checked_in_at"`
}

type PaginatedCheckInSummary struct {
	Data     []CheckInSummaryItem `json:"data"`
	Paginate models.Pagination    `json:"paginate"`
}

type UserAuthSummary struct {
	ID       uint64 `gorm:"column:id"`
	FullName string `gorm:"column:full_name"`
}

type Repository interface {
	WithTransaction(fn func(repo Repository) error) error
	Register(ctx context.Context, checkIn *models.CheckIn) error
	GetAllByGymID(ctx context.Context, gymID uint64, from, to time.Time, offset, limit int) ([]CheckInSummaryItem, int64, error)
	GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]CheckInSummaryItem, int64, error)
	GetDayPassBySlug(ctx context.Context, slug uuid.UUID) (*models.DayPass, error)
	GetSubscriptionBySlug(ctx context.Context, slug uuid.UUID) (*models.Subscription, error)
	GetSubscriptionByEmail(ctx context.Context, email string, gymID uint64) (*models.Subscription, error)
	GetDayPassByEmail(ctx context.Context, email string, gymID uint64) (*models.DayPass, error)
	IsDayPassPaid(ctx context.Context, dayPassID uint64) (bool, error)
	MarkDayPassAsUsed(ctx context.Context, id uint64) error
	GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error)
	GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error)
	GetUserDetailsByUserID(ctx context.Context, userID uint64) (string, error)
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetManagerGymID(ctx context.Context, authID uint64) (uint64, error)
	GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, string, error)
}

type Service interface {
	ValidateAccess(ctx context.Context, authID uint64, req ValidateAccessRequest) (*AccessResultResponse, error)
	GetMyHistory(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedCheckInSummary, error)
	GetAllByGym(ctx context.Context, authID, gymID uint64, fromStr, toStr string, page, pageSize int) (*PaginatedCheckInSummary, error)
}

func NewCheckIn(gymID, accessTypeID uint64, userID, dayPassID, subID, registeredByID *uint64) (*models.CheckIn, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	if accessTypeID == 0 {
		return nil, fmt.Errorf("invalid accessTypeID: %d", accessTypeID)
	}

	return &models.CheckIn{
		GymID:          gymID,
		AccessTypeID:   accessTypeID,
		UserID:         userID,
		DayPassID:      dayPassID,
		SubscriptionID: subID,
		RegisteredByID: registeredByID,
		CheckedInAt:    time.Now(),
	}, nil
}
