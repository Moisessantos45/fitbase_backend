package owner

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"strings"
)

type Detail struct {
	ID        uint64 `json:"id"`
	AuthID    uint64 `json:"auth_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Slug      string `json:"-"`
	AvatarURL string `json:"avatar_url"`
}

type UpdateRequest struct {
	FirstName string `json:"first_name" binding:"omitempty,min=2,max=100"`
	LastName  string `json:"last_name" binding:"omitempty,min=2,max=100"`
	Phone     string `json:"phone" binding:"omitempty,min=7,max=20"`
}

type CompleteOwnerProfileRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string `json:"last_name" binding:"required,min=2,max=100"`
	Phone     string `json:"phone" binding:"required,min=7,max=20"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetByAuthID(ctx context.Context, authID uint64) (*Detail, error)
	ExistsByAuthID(ctx context.Context, authID uint64) (bool, error)
	Create(ctx context.Context, owner *models.Owner) error
	Update(ctx context.Context, authID uint64, data map[string]any) error
	MarkAuthProfileCompleted(ctx context.Context, authID uint64) error
	CreateFile(ctx context.Context, file *models.File) error
	UpdateAvatarByAuthID(ctx context.Context, authID uint64, avatarID uint64) error
}

type Service interface {
	CompleteProfile(ctx context.Context, authID uint64, firstName, lastName, phone string) (*Detail, error)
	GetProfile(ctx context.Context, authID uint64) (*Detail, error)
	Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error
	UploadAvatar(ctx context.Context, authID uint64, slug string, path string, mimeType string, size int64) error
}

func New(authID uint64, firstName, lastName, phone string) (*models.Owner, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
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

	return &models.Owner{
		AuthID:    authID,
		FirstName: trimmedFirst,
		LastName:  trimmedLast,
		Phone:     trimmedPhone,
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
