package daypass

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateDayPassRequest struct {
	GymID     uint64 `json:"gym_id" binding:"required"`
	PlanID    uint64 `json:"plan_id" binding:"required"`
	VisitDate string `json:"visit_date" binding:"required"`
}

type RegisterGuestRequest struct {
	GymID     uint64 `json:"gym_id" binding:"required"`
	PlanID    uint64 `json:"plan_id" binding:"required"`
	FullName  string `json:"full_name" binding:"required,min=2,max=100"`
	Phone     string `json:"phone" binding:"required,min=7,max=20"`
	Email     string `json:"email" binding:"required,email"`
	VisitDate string `json:"visit_date" binding:"omitempty"`
}

type DayPassTicketResponse struct {
	ID        uint64    `json:"id" gorm:"column:id"`
	Slug      string    `json:"slug" gorm:"column:slug"`
	GymID     uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName   string    `json:"gym_name" gorm:"column:gym_name"`
	PlanID    uint64    `json:"plan_id" gorm:"column:plan_id"`
	PlanName  string    `json:"plan_name" gorm:"column:plan_name"`
	PlanPrice float64   `json:"plan_price" gorm:"column:plan_price"`
	IsPaid    bool      `json:"is_paid" gorm:"column:is_paid"`
	FullName  string    `json:"full_name" gorm:"column:full_name"`
	Phone     string    `json:"phone" gorm:"column:phone"`
	Email     string    `json:"email" gorm:"column:email"`
	VisitDate string    `json:"visit_date" gorm:"column:visit_date"`
	IsUsed    bool      `json:"is_used" gorm:"column:is_used"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

type DayPassDetailResponse struct {
	ID        uint64    `json:"id" gorm:"column:id"`
	Slug      string    `json:"slug" gorm:"column:slug"`
	UserID    *uint64   `json:"user_id" gorm:"column:user_id"`
	GymID     uint64    `json:"gym_id" gorm:"column:gym_id"`
	GymName   string    `json:"gym_name" gorm:"column:gym_name"`
	PlanID    uint64    `json:"plan_id" gorm:"column:plan_id"`
	PlanName  string    `json:"plan_name" gorm:"column:plan_name"`
	PlanPrice float64   `json:"plan_price" gorm:"column:plan_price"`
	IsPaid    bool      `json:"is_paid" gorm:"column:is_paid"`
	FullName  string    `json:"full_name" gorm:"column:full_name"`
	Phone     string    `json:"phone" gorm:"column:phone"`
	Email     string    `json:"email" gorm:"column:email"`
	VisitDate string    `json:"visit_date" gorm:"column:visit_date"`
	IsUsed    bool      `json:"is_used" gorm:"column:is_used"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

type UserAuthProfile struct {
	ID       uint64 `gorm:"column:id"`
	FullName string `gorm:"column:full_name"`
	Phone    string `gorm:"column:phone"`
	Email    string `gorm:"column:email"`
}

type PaginatedDayPassSummary struct {
	Data     []DayPassDetailResponse `json:"data"`
	Paginate models.Pagination       `json:"paginate"`
}

type Repository interface {
	Create(ctx context.Context, dayPass *models.DayPass) error
	GetBySlug(ctx context.Context, slug uuid.UUID) (*models.DayPass, error)
	GetTicketBySlug(ctx context.Context, slug uuid.UUID) (*DayPassTicketResponse, error)
	MarkAsUsed(ctx context.Context, id uint64) error
	GetAllByGymID(ctx context.Context, gymID uint64, date time.Time, search string, offset, limit int) ([]DayPassDetailResponse, int64, error)
	GetSchedulesByGymID(ctx context.Context, gymID uint64) ([]models.GymSchedule, error)
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetManagerGymID(ctx context.Context, authID uint64) (uint64, error)
	GetUserDetailsByAuthID(ctx context.Context, authID uint64) (uint64, string, string, string, error)
	GetPlanByID(ctx context.Context, planID uint64) (*models.Plan, error)
}

type Service interface {
	PurchaseDayPass(ctx context.Context, authID uint64, req CreateDayPassRequest) (*DayPassTicketResponse, error)
	RegisterGuest(ctx context.Context, authID uint64, req RegisterGuestRequest) (*DayPassTicketResponse, error)
	CreatePublicDayPass(ctx context.Context, req RegisterGuestRequest) (*DayPassTicketResponse, error)
	GetTicketBySlug(ctx context.Context, slugStr string) (*DayPassTicketResponse, error)
	GetAllByGym(ctx context.Context, authID, gymID uint64, dateStr, search string, page, pageSize int) (*PaginatedDayPassSummary, error)
}

func NewDayPass(gymID, planID uint64, userID *uint64, fullName, phone, email string, visitDate time.Time) (*models.DayPass, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	if planID == 0 {
		return nil, fmt.Errorf("invalid planID: %d", planID)
	}

	trimmedName := strings.TrimSpace(fullName)
	if trimmedName == "" {
		return nil, fmt.Errorf("invalid fullName")
	}

	trimmedPhone := strings.TrimSpace(phone)
	if trimmedPhone == "" {
		return nil, fmt.Errorf("invalid phone")
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, fmt.Errorf("invalid email")
	}

	return &models.DayPass{
		GymID:     gymID,
		PlanID:    planID,
		UserID:    userID,
		FullName:  trimmedName,
		Phone:     trimmedPhone,
		Email:     trimmedEmail,
		VisitDate: visitDate,
		IsUsed:    false,
	}, nil
}
