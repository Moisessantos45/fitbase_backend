package payment

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"time"
)

type RegisterPaymentRequest struct {
	GymID           uint64  `json:"gym_id" binding:"required"`
	PaymentMethodID uint64  `json:"payment_method_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	SubscriptionID  *uint64 `json:"subscription_id" binding:"omitempty"`
	DayPassID       *uint64 `json:"day_pass_id" binding:"omitempty"`
}

type PaymentReceiptResponse struct {
	ID                uint64    `json:"id" gorm:"column:id"`
	GymID             uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName           string    `json:"gym_name" gorm:"column:gym_name"`
	PaymentMethodID   uint64    `json:"payment_method_id" gorm:"column:payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name" gorm:"column:payment_method_name"`
	PaymentStatusID   uint64    `json:"payment_status_id" gorm:"column:payment_status_id"`
	PaymentStatusName string    `json:"payment_status_name" gorm:"column:payment_status_name"`
	Amount            float64   `json:"amount" gorm:"column:amount"`
	SubscriptionID    *uint64   `json:"subscription_id,omitempty" gorm:"column:subscription_id"`
	DayPassID         *uint64   `json:"day_pass_id,omitempty" gorm:"column:day_pass_id"`
	CashRegisterID    *uint64   `json:"cash_register_id,omitempty" gorm:"column:cash_register_id"`
	PaymentDate       time.Time `json:"payment_date" gorm:"column:payment_date"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at"`
}

type GymRevenueSummaryResponse struct {
	GymID             uint64  `json:"gym_id"`
	GymName           string  `json:"gym_name"`
	TotalRevenue      float64 `json:"total_revenue"`
	TotalTransactions int64   `json:"total_transactions"`
}

type PaymentRevenueSummary struct {
	TotalRevenue      float64 `gorm:"column:total_revenue"`
	TotalTransactions int64   `gorm:"column:total_transactions"`
}

type PaymentListItem struct {
	ID                uint64    `json:"id" gorm:"column:id"`
	GymID             uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName           string    `json:"gym_name" gorm:"column:gym_name"`
	PaymentMethodID   uint64    `json:"payment_method_id" gorm:"column:payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name" gorm:"column:payment_method_name"`
	PaymentStatusID   uint64    `json:"payment_status_id" gorm:"column:payment_status_id"`
	PaymentStatusName string    `json:"payment_status_name" gorm:"column:payment_status_name"`
	Amount            float64   `json:"amount" gorm:"column:amount"`
	SubscriptionID    *uint64   `json:"subscription_id,omitempty" gorm:"column:subscription_id"`
	DayPassID         *uint64   `json:"day_pass_id,omitempty" gorm:"column:day_pass_id"`
	CashRegisterID    *uint64   `json:"cash_register_id,omitempty" gorm:"column:cash_register_id"`
	PaymentDate       time.Time `json:"payment_date" gorm:"column:payment_date"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at"`
}

type CreateCheckoutSessionRequest struct {
	GymID     uint64 `json:"gym_id" binding:"required"`
	PlanID    uint64 `json:"plan_id" binding:"required"`
	Type      string `json:"type" binding:"required,oneof=subscription daypass"`
	VisitDate string `json:"visit_date" binding:"omitempty"`
	AutoRenew bool   `json:"auto_renew"`
}

type CreateGuestCheckoutSessionRequest struct {
	GymID     uint64 `json:"gym_id" binding:"required"`
	PlanID    uint64 `json:"plan_id" binding:"required"`
	FullName  string `json:"full_name" binding:"required,min=2,max=100"`
	Phone     string `json:"phone" binding:"required,min=7,max=20"`
	Email     string `json:"email" binding:"required,email"`
	VisitDate string `json:"visit_date" binding:"omitempty"`
}

type CheckoutSessionResponse struct {
	SessionID   string  `json:"session_id"`
	CheckoutURL string  `json:"checkout_url"`
	PaymentID   uint64  `json:"payment_id"`
	Amount      float64 `json:"amount"`
}

type StripeSessionStatusResponse struct {
	SessionID     string  `json:"session_id"`
	PaymentStatus string  `json:"payment_status"`
	Status        string  `json:"status"`
	PaymentID     uint64  `json:"payment_id"`
	AmountTotal   float64 `json:"amount_total"`
	Currency      string  `json:"currency"`
}

type PaginatedPaymentSummary struct {
	Data     []PaymentListItem `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type UserAuthProfile struct {
	ID        uint64 `gorm:"column:id"`
	AuthID    uint64 `gorm:"column:auth_id"`
	FirstName string `gorm:"column:first_name"`
	LastName  string `gorm:"column:last_name"`
	FullName  string `gorm:"column:full_name"`
	Phone     string `gorm:"column:phone"`
	Email     string `gorm:"column:email"`
}

type DayPassPaymentDetails struct {
	ID        uint64 `gorm:"column:id"`
	Slug      string `gorm:"column:slug"`
	GymID     uint64 `gorm:"column:gym_id"`
	GymName   string `gorm:"column:gym_name"`
	PlanID    uint64 `gorm:"column:plan_id"`
	PlanName  string `gorm:"column:plan_name"`
	FullName  string `gorm:"column:full_name"`
	Email     string `gorm:"column:email"`
	VisitDate string `gorm:"column:visit_date"`
	IsUsed    bool   `gorm:"column:is_used"`
}

type SubscriptionPaymentDetails struct {
	ID        uint64  `gorm:"column:id"`
	Slug      string  `gorm:"column:slug"`
	GymID     uint64  `gorm:"column:gym_id"`
	GymName   string  `gorm:"column:gym_name"`
	PlanID    uint64  `gorm:"column:plan_id"`
	PlanName  string  `gorm:"column:plan_name"`
	PlanPrice float64 `gorm:"column:plan_price"`
	FullName  string  `gorm:"column:full_name"`
	Email     string  `gorm:"column:email"`
	StartDate string  `gorm:"column:start_date"`
	EndDate   string  `gorm:"column:end_date"`
	AutoRenew bool    `gorm:"column:auto_renew"`
}

type Repository interface {
	WithTransaction(fn func(repo Repository) error) error
	Create(ctx context.Context, payment *models.Payment) error
	GetTotalRevenueByGymID(ctx context.Context, gymID uint64) (float64, int64, error)
	GetAllByGymID(ctx context.Context, gymID uint64, offset, limit int) ([]PaymentListItem, int64, error)
	GetAllByUserID(ctx context.Context, userID uint64, offset, limit int) ([]PaymentListItem, int64, error)
	GetActiveCashRegisterByAuthID(ctx context.Context, authID, gymID uint64) (*models.CashRegister, error)
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetGymName(ctx context.Context, gymID uint64) (string, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetManagerGymID(ctx context.Context, authID uint64) (uint64, error)
	GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetPaymentMethodByID(ctx context.Context, methodID uint64) (*models.PaymentMethod, error)
	GetReceiptByID(ctx context.Context, paymentID uint64) (*PaymentReceiptResponse, error)
	GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error)
	GetPaymentByID(ctx context.Context, paymentID uint64) (*models.Payment, error)
	Update(ctx context.Context, paymentID uint64, updates map[string]any) error
	CreateSubscription(ctx context.Context, sub *models.Subscription) error
	UpdateSubscription(ctx context.Context, subID uint64, updates map[string]any) error
	CreateDayPass(ctx context.Context, dp *models.DayPass) error
	GetDayPassDetailsByID(ctx context.Context, dayPassID uint64) (*DayPassPaymentDetails, error)
	GetSubscriptionDetailsByID(ctx context.Context, subID uint64) (*SubscriptionPaymentDetails, error)
	GetUserProfileByAuthID(ctx context.Context, authID uint64) (*UserAuthProfile, error)
	GetGymByID(ctx context.Context, gymID uint64) (*models.Gym, error)
	GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error)
	IsDayPassPaid(ctx context.Context, dayPassID uint64) (bool, error)
}

type Service interface {
	RegisterCashPayment(ctx context.Context, authID uint64, req RegisterPaymentRequest) (*PaymentReceiptResponse, error)
	GetRevenueSummary(ctx context.Context, authID, gymID uint64) (*GymRevenueSummaryResponse, error)
	GetAllByGym(ctx context.Context, authID, gymID uint64, page, pageSize int) (*PaginatedPaymentSummary, error)
	GetMyPayments(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedPaymentSummary, error)
	CreateCheckoutSession(ctx context.Context, authID uint64, req CreateCheckoutSessionRequest) (*CheckoutSessionResponse, error)
	CreateGuestCheckoutSession(ctx context.Context, req CreateGuestCheckoutSessionRequest) (*CheckoutSessionResponse, error)
	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error
	GetSessionStatus(ctx context.Context, authID uint64, sessionID string) (*StripeSessionStatusResponse, error)
}

func NewPayment(gymID, paymentMethodID, statusID uint64, amount float64, subID, dayPassID, cashRegisterID *uint64) (*models.Payment, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	if paymentMethodID == 0 {
		return nil, fmt.Errorf("invalid paymentMethodID: %d", paymentMethodID)
	}

	if statusID == 0 {
		return nil, fmt.Errorf("invalid statusID: %d", statusID)
	}

	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	return &models.Payment{
		GymID:           gymID,
		PaymentMethodID: paymentMethodID,
		PaymentStatusID: statusID,
		Amount:          amount,
		SubscriptionID:  subID,
		DayPassID:       dayPassID,
		CashRegisterID:  cashRegisterID,
		PaymentDate:     time.Now(),
	}, nil
}
