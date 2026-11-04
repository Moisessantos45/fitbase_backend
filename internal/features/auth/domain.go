package auth

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/utils"

	"fmt"

	"strings"
)

type UserBasic struct {
	ID    string `gorm:"column:id"`
	Email string `gorm:"column:email"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	Register(user *models.Auth) error
	GetByEmail(email string) (*models.Auth, error)
	GetByEmailAny(email string) (*models.Auth, error)
	ExistsByEmail(email string) (bool, error)
	GetActiveUserByEmail(context context.Context, email string) (*models.Auth, error)
	GetByID(id uint64) (*models.Auth, error)
	GetBasicByID(id uint64) (*UserBasic, error)
	GetBasicByEmail(email string) (*UserBasic, error)

	UpdateEmailConfirm(id uint64) error
	UpdateEmail(id uint64, email string) error
	UpdatePassword(id uint64, newPassword string) error
	SetTwoFactor(id uint64, enabled bool, secret string) error
	ChangeStatus(id uint64, active bool) error
	DeleteByID(id uint64) error
}

type Service interface {
	Register(ctx context.Context, email string, password string) (*models.Auth, error)
	RegisterOwner(ctx context.Context, email string, password string) (*models.Auth, error)
	Login(ctx context.Context, email string, password string) (*models.Auth, string, error)
	LoginMobile(ctx context.Context, email string, password string) (*models.Auth, string, error)
	GetSession(userId uint64) (*models.Auth, error)
	GetBasicByID(id uint64) (*UserBasic, error)
	RefreshToken(ctx context.Context, token string) (string, error)
	RefreshTokenMobile(ctx context.Context, token string) (string, string, error)
	VerifyEmail(email string) error
	ExistsByEmail(email string) (bool, error)
	ForwardEmailVerification(ctx context.Context, email string) error
	ConfirmAccount(ctx context.Context, userId uint64, token string) error
	SendPasswordReset(ctx context.Context, email string) error
	ResetPassword(userId uint64, newPassword string) error
	UpdatePassword(userId uint64, currentPassword string, newPassword string) error
	UpdateEmail(ctx context.Context, authID uint64, newEmail string) error
	Deactivate(ctx context.Context, userId uint64) error
	GetSessionByUserID(ctx context.Context, userID uint64) (map[string]string, error)
	EnableTWOFA(userId uint64) (string, string, error)
	DisableTWOFA(userId uint64) error
	VerifyTOTP(userId uint64, code string) (bool, error)
	Verify2FALogin(ctx context.Context, sessionID string, code string) (*models.Auth, string, error)
	Logout(ctx context.Context, token string) error
}

func NewUser(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		RoleID:       models.RoleUserID,
		Email:        strings.TrimSpace(email),
		PasswordHash: password,
		FullProfile:  false,
	}, nil
}

func NewOwner(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		RoleID:       models.RoleOwnerID,
		Email:        strings.TrimSpace(email),
		PasswordHash: password,
		FullProfile:  false,
	}, nil
}

func NewUserAdmin(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		RoleID:       models.RoleAdminID,
		Email:        email,
		PasswordHash: password,
		FullProfile:  true,
	}, nil
}
