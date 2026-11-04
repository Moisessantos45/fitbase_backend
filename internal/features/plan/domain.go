package plan

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fitbase/internal/models"
	"fmt"
	"strings"
)

type PlanItemRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"omitempty,max=255"`
}

type PlanPublicItem struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PlanDetailResponse struct {
	ID          uint64           `json:"id" gorm:"column:id"`
	GymID       uint64           `json:"gym_id" gorm:"column:gym_id"`
	GymName     string           `json:"gym_name" gorm:"column:gym_name"`
	Name        string           `json:"name" gorm:"column:name"`
	Description string           `json:"description" gorm:"column:description"`
	Price       float64          `json:"price" gorm:"column:price"`
	Discount    float64          `json:"discount" gorm:"column:discount"`
	FinalPrice  float64          `json:"final_price" gorm:"column:final_price"`
	Duration    int              `json:"duration" gorm:"column:duration"`
	PlanType    string           `json:"plan_type" gorm:"column:plan_type"`
	MaxVisits   *int             `json:"max_visits" gorm:"column:max_visits"`
	Active      bool             `json:"active" gorm:"column:active"`
	ItemsRaw    []byte           `json:"-" gorm:"column:items"`
	Items       []PlanPublicItem `json:"items" gorm:"-"`
}

type PaginatedPlanSummary struct {
	Data     []PlanDetailResponse `json:"data"`
	Paginate models.Pagination    `json:"paginate"`
}

type EnrollmentActiveSubscription struct {
	ID            uint64 `json:"id"`
	Slug          string `json:"slug"`
	PlanID        uint64 `json:"plan_id"`
	PlanName      string `json:"plan_name"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	RemainingDays int    `json:"remaining_days"`
	StatusName    string `json:"status_name"`
	AutoRenew     bool   `json:"auto_renew"`
}

func (s EnrollmentActiveSubscription) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *EnrollmentActiveSubscription) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, s)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), s)
	}
	return nil
}

type EnrollmentPlanItem struct {
	ID          uint64           `json:"id"`
	GymID       uint64           `json:"gym_id"`
	GymName     string           `json:"gym_name"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Price       float64          `json:"price"`
	Discount    float64          `json:"discount"`
	FinalPrice  float64          `json:"final_price"`
	Duration    int              `json:"duration"`
	PlanType    string           `json:"plan_type"`
	MaxVisits   *int             `json:"max_visits"`
	Active      bool             `json:"active"`
	IsCurrent   bool             `json:"is_current"`
	Items       []PlanPublicItem `json:"items"`
}

type EnrollmentPlanList []EnrollmentPlanItem

func (l EnrollmentPlanList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *EnrollmentPlanList) Scan(value any) error {
	if value == nil {
		*l = EnrollmentPlanList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = EnrollmentPlanList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = EnrollmentPlanList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type EnrollmentResponse struct {
	GymID                 uint64                        `json:"gym_id" gorm:"column:gym_id"`
	GymName               string                        `json:"gym_name" gorm:"column:gym_name"`
	HasActiveSubscription bool                          `json:"has_active_subscription" gorm:"column:has_active_subscription"`
	ActiveSubscription    *EnrollmentActiveSubscription `json:"active_subscription" gorm:"column:active_subscription"`
	Plans                 EnrollmentPlanList            `json:"plans" gorm:"column:plans"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	CreateWithItems(ctx context.Context, plan *models.Plan, items []models.PlanItem) error
	GetByID(ctx context.Context, id uint64) (*models.Plan, error)
	GetDetailsByID(ctx context.Context, id uint64) (*PlanDetailResponse, error)
	GetAllByGymID(ctx context.Context, gymID uint64, activeOnly bool) ([]PlanDetailResponse, error)
	GetEnrollmentPlans(ctx context.Context, authID, gymID uint64) (*EnrollmentResponse, error)
	Update(ctx context.Context, id uint64, data map[string]any) error
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetManagerGymID(ctx context.Context, authID uint64) (uint64, error)
	CreateItem(ctx context.Context, item *models.PlanItem) error
	GetItemByID(ctx context.Context, id uint64) (*models.PlanItem, error)
	UpdateItem(ctx context.Context, id uint64, data map[string]any) error
	DeleteItem(ctx context.Context, id uint64) error
}

type Service interface {
	Create(ctx context.Context, authID, gymID uint64, name, description, planType string, price, discount float64, duration int, maxVisits *int, items []PlanItemRequest) (*PlanDetailResponse, error)
	Update(ctx context.Context, authID, planID uint64, name, description, planType string, price, discount *float64, duration *int, maxVisits *int, active *bool) error
	GetDetailsByID(ctx context.Context, id uint64) (*PlanDetailResponse, error)
	GetAllByGymID(ctx context.Context, authID, gymID uint64, activeOnly bool) ([]PlanDetailResponse, error)
	GetEnrollmentPlans(ctx context.Context, authID, gymID uint64) (*EnrollmentResponse, error)
	AddItem(ctx context.Context, authID, planID uint64, req PlanItemRequest) (*PlanPublicItem, error)
	UpdateItem(ctx context.Context, authID, planID, itemID uint64, req UpdatePlanItemRequest) error
	DeleteItem(ctx context.Context, authID, planID, itemID uint64) error
}

func New(gymID uint64, name, description, planType string, price, discount float64, duration int, maxVisits *int) (*models.Plan, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("invalid name")
	}

	trimmedDescription := strings.TrimSpace(description)
	if trimmedDescription == "" {
		return nil, fmt.Errorf("invalid description")
	}

	trimmedPlanType := strings.TrimSpace(planType)
	if trimmedPlanType == "" {
		return nil, fmt.Errorf("invalid planType")
	}

	if price <= 0 {
		return nil, fmt.Errorf("invalid price: %f", price)
	}

	if discount < 0 || discount > 100 {
		return nil, fmt.Errorf("invalid discount: %f", discount)
	}

	if duration < 1 {
		return nil, fmt.Errorf("invalid duration: %d", duration)
	}

	return &models.Plan{
		GymID:       gymID,
		Name:        trimmedName,
		Description: trimmedDescription,
		Price:       price,
		Discount:    discount,
		Duration:    duration,
		PlanType:    trimmedPlanType,
		MaxVisits:   maxVisits,
		Active:      true,
	}, nil
}

func BuildPlanUpdateData(name, description, planType string, price, discount *float64, duration *int, maxVisits *int, active *bool) map[string]any {
	updateData := make(map[string]any)

	trimmedName := strings.TrimSpace(name)
	if trimmedName != "" {
		updateData["name"] = trimmedName
	}

	trimmedDescription := strings.TrimSpace(description)
	if trimmedDescription != "" {
		updateData["description"] = trimmedDescription
	}

	trimmedPlanType := strings.TrimSpace(planType)
	if trimmedPlanType != "" {
		updateData["plan_type"] = trimmedPlanType
	}

	if price != nil && *price > 0 {
		updateData["price"] = *price
	}

	if discount != nil && *discount >= 0 && *discount <= 100 {
		updateData["discount"] = *discount
	}

	if duration != nil && *duration >= 1 {
		updateData["duration"] = *duration
	}

	if maxVisits != nil && *maxVisits >= 0 {
		updateData["max_visits"] = *maxVisits
	}

	if maxVisits != nil && *maxVisits == 0 {
		updateData["max_visits"] = nil
	}

	if active != nil {
		updateData["active"] = *active
	}

	return updateData
}

func BuildPlanItemUpdateData(req UpdatePlanItemRequest) map[string]any {
	data := make(map[string]any)
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed != "" {
			data["name"] = trimmed
		}
	}
	if req.Description != nil {
		data["description"] = strings.TrimSpace(*req.Description)
	}
	return data
}
