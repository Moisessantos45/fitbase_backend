package subscription

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PurchaseSubscriptionRequest struct {
	PlanID          uint64 `json:"plan_id" binding:"required"`
	PaymentMethodID uint64 `json:"payment_method_id" binding:"required"`
	StartDate       string `json:"start_date" binding:"omitempty"`
	AutoRenew       bool   `json:"auto_renew"`
}

type RenewSubscriptionRequest struct {
	PaymentMethodID uint64 `json:"payment_method_id" binding:"required"`
}

type SubscriptionTicketResponse struct {
	ID            uint64    `json:"id" gorm:"column:id"`
	Slug          string    `json:"slug" gorm:"column:slug"`
	GymID         uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName       string    `json:"gym_name" gorm:"column:gym_name"`
	PlanID        uint64    `json:"plan_id" gorm:"column:plan_id"`
	PlanName      string    `json:"plan_name" gorm:"column:plan_name"`
	PlanPrice     float64   `json:"plan_price" gorm:"column:plan_price"`
	FullName      string    `json:"full_name" gorm:"column:full_name"`
	Email         string    `json:"email,omitempty" gorm:"column:email"`
	StatusID      uint64    `json:"status_id" gorm:"column:status_id"`
	StatusName    string    `json:"status_name" gorm:"column:status_name"`
	StartDate     string    `json:"start_date" gorm:"column:start_date"`
	EndDate       string    `json:"end_date" gorm:"column:end_date"`
	RemainingDays int       `json:"remaining_days" gorm:"-"`
	QRCodeURL     string    `json:"qr_code_url,omitempty" gorm:"-"`
	AutoRenew     bool      `json:"auto_renew" gorm:"column:auto_renew"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
}

type SubscriptionDetailResponse struct {
	ID                uint64    `json:"id" gorm:"column:id"`
	Slug              string    `json:"slug" gorm:"column:slug"`
	UserID            uint64    `json:"user_id" gorm:"column:user_id"`
	UserFullName      string    `json:"user_full_name" gorm:"column:user_full_name"`
	UserEmail         string    `json:"user_email" gorm:"column:user_email"`
	UserPhone         string    `json:"user_phone" gorm:"column:user_phone"`
	GymID             uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName           string    `json:"gym_name" gorm:"column:gym_name"`
	PlanID            uint64    `json:"plan_id" gorm:"column:plan_id"`
	PlanName          string    `json:"plan_name" gorm:"column:plan_name"`
	PlanPrice         float64   `json:"plan_price" gorm:"column:plan_price"`
	StatusID          uint64    `json:"status_id" gorm:"column:status_id"`
	StatusName        string    `json:"status_name" gorm:"column:status_name"`
	StartDate         string    `json:"start_date" gorm:"column:start_date"`
	EndDate           string    `json:"end_date" gorm:"column:end_date"`
	RemainingDays     int       `json:"remaining_days" gorm:"-"`
	PaymentMethodID   uint64    `json:"payment_method_id" gorm:"column:payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name" gorm:"column:payment_method_name"`
	AutoRenew         bool      `json:"auto_renew" gorm:"column:auto_renew"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at"`
}

type SubscriptionUserSummary struct {
	ID            uint64    `json:"id" gorm:"column:id"`
	Slug          string    `json:"slug" gorm:"column:slug"`
	GymID         uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName       string    `json:"gym_name" gorm:"column:gym_name"`
	PlanID        uint64    `json:"plan_id" gorm:"column:plan_id"`
	PlanName      string    `json:"plan_name" gorm:"column:plan_name"`
	PlanPrice     float64   `json:"plan_price" gorm:"column:plan_price"`
	StatusID      uint64    `json:"status_id" gorm:"column:status_id"`
	StatusName    string    `json:"status_name" gorm:"column:status_name"`
	StartDate     string    `json:"start_date" gorm:"column:start_date"`
	EndDate       string    `json:"end_date" gorm:"column:end_date"`
	RemainingDays int       `json:"remaining_days" gorm:"-"`
	AutoRenew     bool      `json:"auto_renew" gorm:"column:auto_renew"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
}

type UserAuthProfile struct {
	ID       uint64 `gorm:"column:id"`
	FullName string `gorm:"column:full_name"`
	Phone    string `gorm:"column:phone"`
	Email    string `gorm:"column:email"`
}

type PaginatedSubscriptionSummary struct {
	Data     []SubscriptionDetailResponse `json:"data"`
	Paginate models.Pagination            `json:"paginate"`
}

type PaginatedUserSubscriptionSummary struct {
	Data     []SubscriptionUserSummary `json:"data"`
	Paginate models.Pagination         `json:"paginate"`
}

type Repository interface {
	WithTransaction(fn func(repo Repository) error) error
	Create(ctx context.Context, sub *models.Subscription) error
	GetByID(ctx context.Context, id uint64) (*models.Subscription, error)
	GetBySlug(ctx context.Context, slug uuid.UUID) (*models.Subscription, error)
	GetTicketBySlug(ctx context.Context, slug uuid.UUID) (*SubscriptionTicketResponse, error)
	GetTicketByID(ctx context.Context, id uint64) (*SubscriptionTicketResponse, error)
	GetActiveByUserID(ctx context.Context, userID, gymID uint64) (*models.Subscription, error)
	GetAllByGymID(ctx context.Context, gymID uint64, statusID uint64, offset, limit int) ([]SubscriptionDetailResponse, int64, error)
	GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]SubscriptionUserSummary, int64, error)
	UpdateStatus(ctx context.Context, id uint64, statusID uint64) error
	Renew(ctx context.Context, id uint64, newStartDate, newEndDate time.Time, paymentMethodID uint64) error
	Cancel(ctx context.Context, id uint64) error
	GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, string, string, string, error)
	GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error)
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetManagerGymID(ctx context.Context, authID uint64) (uint64, error)
}

type Service interface {
	PurchaseSubscription(ctx context.Context, authID uint64, req PurchaseSubscriptionRequest) (*SubscriptionTicketResponse, error)
	RenewSubscription(ctx context.Context, authID, subscriptionID uint64, req RenewSubscriptionRequest) (*SubscriptionTicketResponse, error)
	GetMySubscriptions(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedUserSubscriptionSummary, error)
	GetTicketBySlug(ctx context.Context, slugStr string) (*SubscriptionTicketResponse, error)
	GetAllByGym(ctx context.Context, authID, gymID uint64, statusID uint64, page, pageSize int) (*PaginatedSubscriptionSummary, error)
	CancelSubscription(ctx context.Context, authID, subscriptionID uint64) (*SubscriptionTicketResponse, error)
}

func NewSubscription(userID, planID, statusID, paymentMethodID uint64, startDate, endDate time.Time, autoRenew bool) (*models.Subscription, error) {
	if userID == 0 {
		return nil, fmt.Errorf("invalid userID: %d", userID)
	}

	if planID == 0 {
		return nil, fmt.Errorf("invalid planID: %d", planID)
	}

	if statusID == 0 {
		return nil, fmt.Errorf("invalid statusID: %d", statusID)
	}

	if paymentMethodID == 0 {
		return nil, fmt.Errorf("invalid paymentMethodID: %d", paymentMethodID)
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("endDate cannot be before startDate")
	}

	return &models.Subscription{
		UserID:          userID,
		PlanID:          planID,
		StatusID:        statusID,
		StartDate:       startDate,
		EndDate:         endDate,
		PaymentMethodID: paymentMethodID,
		AutoRenew:       autoRenew,
	}, nil
}
