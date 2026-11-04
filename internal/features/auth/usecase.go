package auth

import (
	"context"
	"errors"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/templates"
	"fitbase/internal/shared/utils"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthUseCase struct {
	rd   *redis.Client
	repo Repository
	mk   *utils.PasetoMaker
	host string
}

func NewUseCase(repo Repository, rd *redis.Client, mk *utils.PasetoMaker, host string) Service {
	return &AuthUseCase{
		repo: repo,
		rd:   rd,
		mk:   mk,
		host: host,
	}
}

func (a *AuthUseCase) createSession(ctx context.Context, user *models.Auth, accessToken, refreshToken, platform string) (string, error) {
	platformPrefix := platform
	if platformPrefix == "" {
		platformPrefix = "web"
	}

	userSessionKey := fmt.Sprintf("user_session:%s:%d", platformPrefix, user.ID)

	oldSessionID, err := a.rd.Get(ctx, userSessionKey).Result()
	if err == nil && oldSessionID != "" {
		_ = a.rd.Del(ctx, fmt.Sprintf("session:%s", oldSessionID)).Err()
	}

	sessionID, err := utils.GenerateSessionID()
	if err != nil {
		return "", err
	}

	sessionData := utils.NewSession(user)
	sessionData["access_token"] = accessToken
	sessionData["refresh_token"] = refreshToken
	sessionData["platform"] = platformPrefix

	key := fmt.Sprintf("session:%s", sessionID)
	ttl := 10 * time.Hour

	err = a.rd.HSet(ctx, key, sessionData).Err()
	if err != nil {
		return "", err
	}

	err = a.rd.Expire(ctx, key, ttl).Err()
	if err != nil {
		return "", err
	}

	a.rd.Set(ctx, userSessionKey, sessionID, ttl)

	return sessionID, nil
}

func (a *AuthUseCase) Register(ctx context.Context, email string, password string) (*models.Auth, error) {
	existingUser, err := a.repo.ExistsByEmail(email)
	if err != nil {
		log.Printf("Error checking if user exists: %v", err)
		return nil, errorsx.Internal("No se pudo completar el registro, intente más tarde", err)
	}

	if existingUser {
		return nil, errorsx.Conflict("El correo electrónico ya se encuentra registrado")
	}

	newUser, err := NewUser(email, password)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := utils.HashPassword(newUser.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	newUser.PasswordHash = hashedPassword

	err = a.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.Register(newUser); err != nil {
			if errors.Is(err, errorsx.ErrAlreadyExists) {
				return errorsx.Conflict("El correo electrónico ya se encuentra registrado")
			}
			return errorsx.Internal("No se pudo registrar el usuario", err)
		}

		if newUser.ID == 0 {
			return errorsx.Internal("No se pudo registrar el usuario", err)
		}

		token, err := a.mk.NewToken(fmt.Sprintf("%d", newUser.ID), 15*time.Minute)
		if err != nil {
			return fmt.Errorf("Error generating token")
		}

		confirmKey := fmt.Sprintf("%s:%d", templates.CONFIRM_ACCOUNT_PATH, newUser.ID)
		err = a.rd.Set(ctx, confirmKey, token, 15*time.Minute).Err()
		if err != nil {
			return fmt.Errorf("error caching token: %w", err)
		}

		renderer, err := templates.NewEmailRenderer()
		if err != nil {
			return err
		}

		data := templates.ConfirmAccountData{
			Name:        newUser.Email,
			ConfirmLink: fmt.Sprintf("%s/%s/%s", a.host, templates.CONFIRM_ACCOUNT_PATH, token),
		}

		htmlContent, err := renderer.RenderConfirmAccount(data)
		if err != nil {
			return err
		}

		err = utils.EnqueueEmail([]string{newUser.Email}, "Confirmacion de cuenta", htmlContent)
		if err != nil {
			return fmt.Errorf("error sending email: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (a *AuthUseCase) RegisterOwner(ctx context.Context, email string, password string) (*models.Auth, error) {
	existingUser, err := a.repo.ExistsByEmail(email)
	if err != nil {
		log.Printf("Error checking if user exists: %v", err)
		return nil, errorsx.Internal("No se pudo completar el registro, intente más tarde", err)
	}

	if existingUser {
		return nil, errorsx.Conflict("El correo electrónico ya se encuentra registrado")
	}

	newOwner, err := NewOwner(email, password)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := utils.HashPassword(newOwner.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	newOwner.PasswordHash = hashedPassword

	err = a.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.Register(newOwner); err != nil {
			if errors.Is(err, errorsx.ErrAlreadyExists) {
				return errorsx.Conflict("El correo electrónico ya se encuentra registrado")
			}
			return errorsx.Internal("No se pudo registrar el usuario", err)
		}

		if newOwner.ID == 0 {
			return errorsx.Internal("No se pudo registrar el usuario", err)
		}

		token, err := a.mk.NewToken(fmt.Sprintf("%d", newOwner.ID), 15*time.Minute)
		if err != nil {
			return fmt.Errorf("Error generating token")
		}

		confirmKey := fmt.Sprintf("%s:%d", templates.CONFIRM_ACCOUNT_PATH, newOwner.ID)
		err = a.rd.Set(ctx, confirmKey, token, 15*time.Minute).Err()
		if err != nil {
			return fmt.Errorf("error caching token: %w", err)
		}

		renderer, err := templates.NewEmailRenderer()
		if err != nil {
			return err
		}

		data := templates.ConfirmAccountData{
			Name:        newOwner.Email,
			ConfirmLink: fmt.Sprintf("%s/%s/%s", a.host, templates.CONFIRM_ACCOUNT_PATH, token),
		}

		htmlContent, err := renderer.RenderConfirmAccount(data)
		if err != nil {
			return err
		}

		err = utils.EnqueueEmail([]string{newOwner.Email}, "Confirmacion de cuenta", htmlContent)
		if err != nil {
			return fmt.Errorf("error sending email: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newOwner, nil
}

func (a *AuthUseCase) Login(ctx context.Context, email string, password string) (*models.Auth, string, error) {
	return a.loginWithPlatform(ctx, email, password, "web")
}

func (a *AuthUseCase) LoginMobile(ctx context.Context, email string, password string) (*models.Auth, string, error) {
	return a.loginWithPlatform(ctx, email, password, "mobile")
}

func (a *AuthUseCase) loginWithPlatform(ctx context.Context, email string, password string, platform string) (*models.Auth, string, error) {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, "", errorsx.ErrInvalidCredentials
	}

	if password == "" || len(password) < 6 {
		return nil, "", errorsx.ErrInvalidCredentials
	}

	user, err := a.repo.GetActiveUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errorsx.ErrCodeEmailNotVerified) {
			return nil, "", errorsx.ErrCodeEmailNotVerified
		}
		if errors.Is(err, errorsx.ErrCodeAccountInactive) {
			utils.BurnPasswordCheck(password)
			return nil, "", errorsx.ErrInvalidCredentials
		}

		return nil, "", err
	}

	if user == nil {
		utils.BurnPasswordCheck(password)
		return nil, "", errorsx.ErrInvalidCredentials
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", errorsx.ErrInvalidCredentials
	}

	if user.TwoFactorEnabled {
		sessionID, err := utils.GenerateSessionID()
		if err != nil {
			return nil, "", fmt.Errorf("error generando sesión temporal")
		}

		key := fmt.Sprintf("preauth:%s", sessionID)
		preAuthData := map[string]any{
			"user_id":            fmt.Sprintf("%d", user.ID),
			"two_factor_enabled": true,
			"auth_stage":         "pending_2fa",
			"mfa_verified":       false,
			"platform":           platform,
		}

		if err := a.rd.HSet(ctx, key, preAuthData).Err(); err != nil {
			return nil, "", fmt.Errorf("error guardando sesión temporal: %w", err)
		}

		if err := a.rd.Expire(ctx, key, 5*time.Minute).Err(); err != nil {
			return nil, "", fmt.Errorf("error configurando TTL de sesión temporal: %w", err)
		}

		return nil, sessionID, errorsx.ErrPending2FA
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 10 * time.Hour

	accessToken, err := a.mk.NewTokenWithPlatform(fmt.Sprintf("%d", user.ID), platform, accessTTL)
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := a.mk.NewTokenWithPlatform(fmt.Sprintf("%d", user.ID), platform, refreshTTL)
	if err != nil {
		return nil, "", err
	}

	_, err = a.createSession(ctx, user, accessToken, refreshToken, platform)
	if err != nil {
		return nil, "", err
	}

	user.Token = accessToken
	user.PasswordHash = ""

	return user, refreshToken, nil
}

func (a *AuthUseCase) Verify2FALogin(ctx context.Context, sessionID string, code string) (*models.Auth, string, error) {
	if sessionID == "" {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	key := fmt.Sprintf("preauth:%s", sessionID)

	vals, err := a.rd.HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	authStage, ok := vals["auth_stage"]
	if !ok || authStage != "pending_2fa" {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	userIDStr, ok := vals["user_id"]
	if !ok {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	var userID uint64
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	user, err := a.repo.GetByID(userID)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", errorsx.Unauthorized("Sesión temporal inválida o expirada")
	}

	if !utils.VerifyTOTP(user.TwoFactorSecret, code) {
		return nil, "", errorsx.Unauthorized("Código 2FA inválido")
	}

	platform, _ := vals["platform"]
	if platform == "" {
		platform = "web"
	}

	a.rd.Del(ctx, key)

	accessToken, err := a.mk.NewTokenWithPlatform(fmt.Sprintf("%d", user.ID), platform, 15*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("error generando access token")
	}

	refreshToken, err := a.mk.NewTokenWithPlatform(fmt.Sprintf("%d", user.ID), platform, 10*time.Hour)
	if err != nil {
		return nil, "", fmt.Errorf("error generando refresh token")
	}

	_, err = a.createSession(ctx, user, accessToken, refreshToken, platform)
	if err != nil {
		return nil, "", err
	}

	user.Token = accessToken
	user.PasswordHash = ""

	return user, refreshToken, nil
}

func (a *AuthUseCase) VerifyEmail(email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return errorsx.BadRequest("Email inválido")
	}

	exists, err := a.repo.ExistsByEmail(email)
	if err != nil {
		return fmt.Errorf("error verificando email: %w", err)
	}
	if !exists {
		return errorsx.NotFound("No se encontró el correo electrónico")
	}

	return nil
}

func (a *AuthUseCase) ExistsByEmail(email string) (bool, error) {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return false, errorsx.BadRequest("Email inválido")
	}

	exists, err := a.repo.ExistsByEmail(email)
	if err != nil {
		return false, fmt.Errorf("error checking email existence: %w", err)
	}

	return exists, nil
}

func (a *AuthUseCase) GetSessionByUserID(ctx context.Context, userID uint64) (map[string]string, error) {
	keys := []string{
		fmt.Sprintf("user_session:web:%d", userID),
		fmt.Sprintf("user_session:mobile:%d", userID),
		fmt.Sprintf("user_session:%d", userID),
	}

	var sessionID string
	var err error
	for _, k := range keys {
		sessionID, err = a.rd.Get(ctx, k).Result()
		if err == nil && sessionID != "" {
			break
		}
	}

	if sessionID == "" {
		return nil, errorsx.Unauthorized("Sesión no encontrada o expirada")
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	vals, err := a.rd.HGetAll(ctx, sessionKey).Result()
	if err != nil || len(vals) == 0 {
		return nil, errorsx.Unauthorized("Sesión no encontrada o expirada")
	}

	return vals, nil
}

func (a *AuthUseCase) deleteSessionByPlatform(ctx context.Context, platform string, userID uint64) {
	userSessionKey := fmt.Sprintf("user_session:%s:%d", platform, userID)
	sessionID, err := a.rd.Get(ctx, userSessionKey).Result()
	if err == nil && sessionID != "" {
		sessionKey := fmt.Sprintf("session:%s", sessionID)

		vals, err := a.rd.HMGet(ctx, sessionKey, "access_token", "refresh_token").Result()
		if err == nil {
			for _, v := range vals {
				if t, ok := v.(string); ok && t != "" {
					p, err := a.mk.VerifyToken(t)
					if err == nil && p != nil {
						ttl := time.Until(p.Exp)
						if ttl > 0 {
							_ = a.rd.Set(ctx, fmt.Sprintf("blacklist:%s", t), "revoked", ttl).Err()
						}
					}
				}
			}
		}

		_ = a.rd.Del(ctx, sessionKey).Err()
	}
	_ = a.rd.Del(ctx, userSessionKey).Err()
}

func (a *AuthUseCase) deleteSessionByUserID(ctx context.Context, userID uint64) {
	a.deleteSessionByPlatform(ctx, "web", userID)
	a.deleteSessionByPlatform(ctx, "mobile", userID)

	legacyKey := fmt.Sprintf("user_session:%d", userID)
	sessionID, err := a.rd.Get(ctx, legacyKey).Result()
	if err == nil && sessionID != "" {
		_ = a.rd.Del(ctx, fmt.Sprintf("session:%s", sessionID)).Err()
	}
	_ = a.rd.Del(ctx, legacyKey).Err()
}

func (a *AuthUseCase) GetSession(userId uint64) (*models.Auth, error) {
	if userId == 0 {
		return nil, errorsx.BadRequest("ID de usuario inválido")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}
	if user == nil {
		return nil, errorsx.ErrNotFound
	}

	user.PasswordHash = ""

	return user, nil
}

func (a *AuthUseCase) GetBasicByID(id uint64) (*UserBasic, error) {
	if id == 0 {
		return nil, errorsx.BadRequest("ID de usuario inválido")
	}

	return a.repo.GetBasicByID(id)
}

func (a *AuthUseCase) RefreshToken(ctx context.Context, token string) (string, error) {
	payload, err := a.mk.VerifyToken(token)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errorsx.ErrUnauthorized, err)
	}

	userId := payload.UserID
	platform := payload.Platform
	if platform == "" {
		platform = "web"
	}

	userSessionKey := fmt.Sprintf("user_session:%s:%s", platform, userId)
	sessionID, err := a.rd.Get(ctx, userSessionKey).Result()
	if err != nil || sessionID == "" {
		// Fallback por si era clave sin prefijo
		legacyKey := fmt.Sprintf("user_session:%s", userId)
		sessionID, _ = a.rd.Get(ctx, legacyKey).Result()
	}
	if sessionID == "" {
		return "", errorsx.Unauthorized("Sesión no encontrada o expirada")
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	storedRefreshToken, err := a.rd.HGet(ctx, sessionKey, "refresh_token").Result()
	if err != nil || storedRefreshToken == "" || storedRefreshToken != token {
		return "", errorsx.Unauthorized("Token de refresco inválido o expirado")
	}

	newToken, err := a.mk.NewTokenWithPlatform(userId, platform, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("error generating new token: %w", err)
	}

	_ = a.rd.HSet(ctx, sessionKey, "access_token", newToken).Err()

	return newToken, nil
}

func (a *AuthUseCase) RefreshTokenMobile(ctx context.Context, token string) (string, string, error) {
	payload, err := a.mk.VerifyToken(token)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", errorsx.ErrUnauthorized, err)
	}

	if payload == nil {
		return "", "", errorsx.Unauthorized("Token inválido: payload vacío")
	}

	userId := payload.UserID
	platform := payload.Platform
	if platform == "" {
		platform = "mobile"
	}

	userSessionKey := fmt.Sprintf("user_session:%s:%s", platform, userId)
	sessionID, err := a.rd.Get(ctx, userSessionKey).Result()
	if err != nil || sessionID == "" {
		// Fallback por si era clave sin prefijo
		legacyKey := fmt.Sprintf("user_session:%s", userId)
		sessionID, _ = a.rd.Get(ctx, legacyKey).Result()
	}

	if sessionID == "" {
		return "", "", errorsx.Unauthorized("Sesión no encontrada o expirada")
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	storedRefreshToken, err := a.rd.HGet(ctx, sessionKey, "refresh_token").Result()
	if err != nil || storedRefreshToken == "" || storedRefreshToken != token {
		return "", "", errorsx.Unauthorized("Token de refresco inválido o expirado")
	}

	newAccessToken, err := a.mk.NewTokenWithPlatform(userId, platform, 15*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("error generating access token: %w", err)
	}

	newRefreshToken, err := a.mk.NewTokenWithPlatform(userId, platform, 10*time.Hour)
	if err != nil {
		return "", "", fmt.Errorf("error generating refresh token: %w", err)
	}

	_ = a.rd.HSet(ctx, sessionKey, "access_token", newAccessToken, "refresh_token", newRefreshToken).Err()
	_ = a.rd.Expire(ctx, sessionKey, 10*time.Hour).Err()
	_ = a.rd.Expire(ctx, userSessionKey, 10*time.Hour).Err()

	return newAccessToken, newRefreshToken, nil
}

func (a *AuthUseCase) ForwardEmailVerification(ctx context.Context, email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return errorsx.BadRequest("Email inválido")
	}

	user, err := a.repo.GetByEmailAny(email)
	if err != nil {
		return errorsx.NotFound("No se encontró el correo electrónico")
	}

	if user == nil {
		return errorsx.NotFound("No se encontró el correo electrónico")
	}

	token, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), 15*time.Minute)
	if err != nil {
		return err
	}

	confirmKey := fmt.Sprintf("%s:%d", templates.CONFIRM_ACCOUNT_PATH, user.ID)
	if err := a.rd.Set(ctx, confirmKey, token, 15*time.Minute).Err(); err != nil {
		return fmt.Errorf("error caching token: %w", err)
	}

	renderer, err := templates.NewEmailRenderer()
	if err != nil {
		return err
	}

	data := templates.ConfirmAccountData{
		Name:        user.Email,
		ConfirmLink: fmt.Sprintf("%s/%s/%s", a.host, templates.CONFIRM_ACCOUNT_PATH, token),
	}

	htmlContent, err := renderer.RenderConfirmAccount(data)
	if err != nil {
		return err
	}

	err = utils.EnqueueEmail([]string{email}, "Confirmacion de cuenta", htmlContent)
	if err != nil {
		return errorsx.Internal("No se pudo enviar el correo electrónico de confirmación", err)
	}

	return nil
}

func (a *AuthUseCase) ConfirmAccount(ctx context.Context, userId uint64, token string) error {
	if userId == 0 {
		return errorsx.BadRequest("ID de usuario inválido")
	}

	if token == "" {
		return errorsx.BadRequest("Token inválido o expirado")
	}

	confirmKey := fmt.Sprintf("%s:%d", templates.CONFIRM_ACCOUNT_PATH, userId)

	storedToken, err := a.rd.Get(ctx, confirmKey).Result()
	if err != nil {
		return errorsx.BadRequest("Token inválido o expirado")
	}

	if storedToken != token {
		return errorsx.BadRequest("Token inválido o expirado")
	}

	if err := a.repo.UpdateEmailConfirm(userId); err != nil {
		return errorsx.Internal("No se pudo confirmar el correo electrónico", err)
	}

	a.rd.Del(ctx, confirmKey)

	return nil
}

func (a *AuthUseCase) SendPasswordReset(ctx context.Context, email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return errorsx.BadRequest("Email inválido")
	}

	user, err := a.repo.GetBasicByEmail(email)
	if err != nil {
		log.Printf("Error buscando email para restablecer contraseña: %v", err)
		return nil
	}
	if user == nil {
		// respuesta idéntica exista o no el correo (evita enumeración)
		log.Printf("Password reset requested for non-existent email")
		return nil
	}

	token, err := a.mk.NewToken(fmt.Sprintf("%s", user.ID), 15*time.Minute)
	if err != nil {
		return err
	}

	resetKey := fmt.Sprintf("reset:%s", user.ID)
	if err := a.rd.Set(ctx, resetKey, token, 15*time.Minute).Err(); err != nil {
		return fmt.Errorf("error caching reset token: %w", err)
	}

	renderer, err := templates.NewEmailRenderer()
	if err != nil {
		return err
	}

	data := templates.PasswordResetData{
		Name:      user.Email,
		ResetLink: fmt.Sprintf("%s/%s/%s", a.host, templates.FORGOT_PASSWORD_PATH, token),
	}

	htmlContent, err := renderer.RenderPasswordReset(data)
	if err != nil {
		return err
	}

	err = utils.EnqueueEmail([]string{email}, "Restablece tu contraseña", htmlContent)
	if err != nil {
		return errorsx.Internal("No se pudo enviar el correo electrónico de restablecimiento de contraseña", err)
	}

	return nil
}

func (a *AuthUseCase) ResetPassword(userId uint64, newPassword string) error {
	if newPassword == "" || len(newPassword) < 6 {
		return errorsx.BadRequest("La contraseña debe tener al menos 6 caracteres")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errorsx.Internal("No se pudo restablecer la contraseña", err)
	}

	err = a.repo.UpdatePassword(userId, hashedPassword)
	if err != nil {
		return errorsx.Internal("No se pudo restablecer la contraseña", err)
	}

	resetKey := fmt.Sprintf("reset:%d", userId)
	_ = a.rd.Del(context.Background(), resetKey).Err()

	return nil
}

func (a *AuthUseCase) UpdatePassword(userId uint64, currentPassword string, newPassword string) error {
	if userId == 0 {
		return errorsx.BadRequest("ID de usuario inválido")
	}

	if newPassword == "" || len(newPassword) < 6 {
		return errorsx.BadRequest("La contraseña debe tener al menos 6 caracteres")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return err
	}
	if user == nil {
		return errorsx.ErrNotFound
	}

	if !utils.CheckPasswordHash(currentPassword, user.PasswordHash) {
		return errorsx.BadRequest("Contraseña actual incorrecta")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errorsx.Internal("No se pudo actualizar la contraseña", err)
	}

	err = a.repo.UpdatePassword(userId, hashedPassword)
	if err != nil {
		return errorsx.Internal("No se pudo actualizar la contraseña", err)
	}

	return nil
}

func (a *AuthUseCase) Deactivate(ctx context.Context, userId uint64) error {
	if userId == 0 {
		return errorsx.BadRequest("ID de usuario inválido")
	}

	err := a.repo.ChangeStatus(userId, false)
	if err != nil {
		return errorsx.Internal("No se pudo desactivar el usuario", err)
	}

	a.deleteSessionByUserID(ctx, userId)

	return nil
}

func (a *AuthUseCase) EnableTWOFA(userId uint64) (string, string, error) {
	if userId == 0 {
		return "", "", errorsx.BadRequest("ID de usuario inválido")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errorsx.ErrNotFound
	}

	uri, secret, err := utils.GenerateTOTP(user.Email, "SpinLuck")
	if err != nil {
		return "", "", errorsx.Internal("No se pudo generar el secreto de 2FA", err)
	}

	err = a.repo.SetTwoFactor(userId, true, secret)
	if err != nil {
		return "", "", errorsx.Internal("No se pudo habilitar la autenticación de dos factores", err)
	}

	return uri, secret, nil
}

func (a *AuthUseCase) VerifyTOTP(userId uint64, code string) (bool, error) {
	if userId == 0 {
		return false, errorsx.BadRequest("ID de usuario inválido")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errorsx.ErrNotFound
	}

	if !user.TwoFactorEnabled {
		return false, errorsx.BadRequest("La autenticación de dos factores no está habilitada para este usuario")
	}

	return utils.VerifyTOTP(user.TwoFactorSecret, code), nil
}

func (a *AuthUseCase) DisableTWOFA(userId uint64) error {
	if userId == 0 {
		return errorsx.BadRequest("ID de usuario inválido")
	}

	err := a.repo.SetTwoFactor(userId, false, "")
	if err != nil {
		return errorsx.Internal("No se pudo deshabilitar la autenticación de dos factores", err)
	}

	return nil
}

func (a *AuthUseCase) Logout(ctx context.Context, token string) error {
	if token == "" || len(token) < 7 {
		return nil
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload, err := a.mk.VerifyToken(token)
	if err == nil && payload != nil {
		var userID uint64
		fmt.Sscanf(payload.UserID, "%d", &userID)
		if userID > 0 {
			if payload.Platform != "" {
				a.deleteSessionByPlatform(ctxTimeout, payload.Platform, userID)
			} else {
				a.deleteSessionByUserID(ctxTimeout, userID)
			}
		}

		ttl := time.Until(payload.Exp)
		if ttl > 0 {
			blacklistKey := fmt.Sprintf("blacklist:%s", token)
			_ = a.rd.Set(ctxTimeout, blacklistKey, "revoked", ttl).Err()
		}
	}

	return nil
}

func (a *AuthUseCase) UpdateEmail(ctx context.Context, authID uint64, newEmail string) error {
	if authID == 0 {
		return errorsx.BadRequest("ID de usuario inválido")
	}

	if newEmail == "" || len(newEmail) <= 10 || !utils.EmailRegex.MatchString(newEmail) {
		return errorsx.BadRequest("Email inválido")
	}

	exists, err := a.repo.ExistsByEmail(newEmail)
	if err != nil {
		return fmt.Errorf("error checking email: %w", err)
	}

	if exists {
		return errorsx.Conflict("El correo electrónico ya se encuentra registrado")
	}

	newEmail = strings.TrimSpace(newEmail)

	if err := a.repo.UpdateEmail(authID, newEmail); err != nil {
		return errorsx.Internal("No se pudo actualizar el correo electrónico", err)
	}

	if err := a.ForwardEmailVerification(ctx, newEmail); err != nil {
		return fmt.Errorf("se actualizó el email pero no se pudo enviar el correo de verificación: %w", err)
	}

	return nil
}
