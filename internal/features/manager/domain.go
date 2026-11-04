package manager

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"strings"
	"time"
)

type CreateRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string `json:"last_name" binding:"required,min=2,max=100"`
	Phone     string `json:"phone" binding:"required,min=7,max=20"`
}

type Items struct {
	ID              uint64    `json:"id"`
	GymID           uint64    `json:"gym_id"`
	GymName         string    `json:"gym_name"`
	AuthID          uint64    `json:"auth_id"`
	CreatedByAuthID uint64    `json:"created_by_auth_id"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	Slug            string    `json:"-"`
	AvatarURL       string    `json:"avatar_url"`
	CreatedAt       time.Time `json:"created_at"`
}

type Detail struct {
	ID        uint64 `json:"id"`
	AuthID    uint64 `json:"auth_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Slug      string `json:"-"`
	AvatarURL string `json:"avatar_url"`
}

type OpenCashRegisterRequest struct {
	GymID    uint64  `json:"gym_id" binding:"required"`
	OpenCash float64 `json:"open_cash" binding:"gte=0"`
}

type CloseCashRegisterRequest struct {
	CloseCash float64 `json:"close_cash" binding:"gte=0"`
}

type CashRegisterStatusResponse struct {
	ID        uint64     `json:"id"`
	GymID     uint64     `json:"gym_id"`
	AuthID    uint64     `json:"auth_id"`
	OpenTime  time.Time  `json:"open_time"`
	CloseTime *time.Time `json:"close_time"`
	OpenCash  float64    `json:"open_cash"`
	CloseCash float64    `json:"close_cash"`
	IsOpen    bool       `json:"is_open"`
}

type PaginatedItems struct {
	Data     []Items           `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	CreateManagerWithAuth(ctx context.Context, authRecord *models.Auth, manager *models.GymManager) error
	GetActiveCashRegister(ctx context.Context, gymID, authID uint64) (*models.CashRegister, error)
	GetActiveCashRegisterByAuthID(ctx context.Context, authID uint64) (*models.CashRegister, error)
	OpenCashRegister(ctx context.Context, register *models.CashRegister) error
	CloseCashRegister(ctx context.Context, registerID uint64, closeCash float64, closeTime time.Time) error
	GetAllByGymID(ctx context.Context, authID uint64, offset int, limit int, query string) ([]Items, int64, error)
	GetDetailByAuthID(ctx context.Context, authID uint64) (*Detail, error)
	GetGymIDByOwnerAuthID(ctx context.Context, authID uint64) (uint64, error)

	ExistsByAuthID(ctx context.Context, authID uint64) (bool, error)
	CreateFile(ctx context.Context, file *models.File) error
	Update(ctx context.Context, authID uint64, data map[string]any) error
	UpdateAvatarByAuthID(ctx context.Context, authID uint64, avatarID uint64) error
	ExistsAuthByEmail(ctx context.Context, email string) (bool, error)
}

type Service interface {
	Create(ctx context.Context, authID uint64, req CreateRequest) error
	GetAllManagersByGym(ctx context.Context, authID uint64, page int, pageSize int, query string) (*PaginatedItems, error)
	OpenCashRegister(ctx context.Context, authID, gymID uint64, openCash float64) (*CashRegisterStatusResponse, error)
	CloseCashRegister(ctx context.Context, authID uint64, closeCash float64) (*CashRegisterStatusResponse, error)
	GetCurrentCashRegister(ctx context.Context, authID uint64) (*CashRegisterStatusResponse, error)
	GetDetailByAuthID(ctx context.Context, authID uint64) (*Detail, error)
	Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error
	UploadAvatar(ctx context.Context, authID uint64, slug, path, mimeType string, size int64) error
}

func NewManager(gymID, createdByAuthID uint64, firstName, lastName, phone string) (*models.GymManager, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	if createdByAuthID == 0 {
		return nil, fmt.Errorf("invalid createdByAuthID: %d", createdByAuthID)
	}

	trimmedFirst := strings.TrimSpace(firstName)
	if trimmedFirst == "" {
		return nil, fmt.Errorf("invalid firstName")
	}

	trimmedLast := strings.TrimSpace(lastName)
	if trimmedLast == "" {
		return nil, fmt.Errorf("invalid lastName")
	}

	trimmedPhone := strings.TrimSpace(phone)
	if trimmedPhone == "" {
		return nil, fmt.Errorf("invalid phone")
	}

	return &models.GymManager{
		GymID:           gymID,
		CreatedByAuthID: createdByAuthID,
		FirstName:       trimmedFirst,
		LastName:        trimmedLast,
		Phone:           trimmedPhone,
	}, nil
}

func BuildUpdateData(firstName *string, lastName *string, phone *string) map[string]any {
	updateData := make(map[string]any)

	if firstName != nil && strings.TrimSpace(*firstName) != "" {
		updateData["first_name"] = *firstName
	}

	if lastName != nil && strings.TrimSpace(*lastName) != "" {
		updateData["last_name"] = *lastName
	}

	if phone != nil && strings.TrimSpace(*phone) != "" {
		updateData["phone"] = *phone
	}

	return updateData
}

func NewCashRegister(gymID, authID uint64, openCash float64) (*models.CashRegister, error) {
	if gymID == 0 {
		return nil, fmt.Errorf("invalid gymID: %d", gymID)
	}

	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	if openCash < 0 {
		return nil, fmt.Errorf("invalid openCash: %f", openCash)
	}

	return &models.CashRegister{
		GymID:    gymID,
		AuthID:   authID,
		OpenTime: time.Now(),
		OpenCash: openCash,
	}, nil
}
