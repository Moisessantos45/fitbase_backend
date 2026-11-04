package user

import (
	"context"
	"fitbase/internal/models"
	"fmt"
	"strings"
)

type UserDetail struct {
	ID        uint64 `json:"id" gorm:"column:id"`
	FirstName string `json:"first_name" gorm:"column:first_name"`
	LastName  string `json:"last_name" gorm:"column:last_name"`
	Email     string `json:"email" gorm:"column:email"`
	Phone     string `json:"phone"`
	Slug      string `json:"-"`
	AvatarURL string `json:"avatar_url"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetByAuthID(ctx context.Context, authID uint64) (*UserDetail, error)
	GetEmailByID(ctx context.Context, id uint64) (string, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, authID uint64, data map[string]any) error
	UpdateAvatarURL(ctx context.Context, userID uint64, avatarID uint64) error
	ChangeCompletedProfile(id uint64, complete bool) error
}

type Service interface {
	GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetByAuthID(ctx context.Context, authID uint64) (*UserDetail, error)
	GetEmailByID(ctx context.Context, id uint64) (string, error)
	CompleteProfile(ctx context.Context, authID uint64, firstName string, lastName string, phone string) (*models.User, error)
	Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error
	UploadAvatar(ctx context.Context, authID uint64, slug string, path, mimeType string, size int64) error
}

func New(authID uint64, firstName string, lastName string, phone string) (*models.User, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	if strings.TrimSpace(firstName) == "" {
		return nil, fmt.Errorf("invalid firstName: %s", firstName)
	}

	if strings.TrimSpace(lastName) == "" {
		return nil, fmt.Errorf("invalid lastName: %s", lastName)
	}

	if strings.TrimSpace(phone) == "" {
		return nil, fmt.Errorf("invalid phone: %s", phone)
	}

	return &models.User{
		AuthID:    authID,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

func BuildUserUpdateData(firstName *string, lastName *string, phone *string) map[string]any {
	updateData := make(map[string]any)

	if firstName != nil && *firstName != "" {
		updateData["first_name"] = *firstName
	}

	if lastName != nil && *lastName != "" {
		updateData["last_name"] = *lastName
	}

	if phone != nil && *phone != "" {
		updateData["phone"] = *phone
	}

	return updateData
}
