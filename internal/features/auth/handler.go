package auth

import (
	"errors"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type Handler struct {
	s Service
}

// getCookieDomain devuelve el dominio de las cookies desde COOKIE_DOMAIN.
// No se usa la cabecera Host (la controla el cliente). Si no está definida, la cookie
// es host-only (sin atributo Domain), que es lo más restrictivo.
func getCookieDomain(_ *gin.Context) string {
	return strings.TrimSpace(os.Getenv("COOKIE_DOMAIN"))
}

func isSecureCookie() bool {
	return os.Getenv("GO_ENV") == "prod"
}

func NewAuthHandler(s Service) *Handler {
	return &Handler{
		s: s,
	}
}

// jwtError responde 401 genérico cuando faltan o son inválidos los datos del token en el contexto;
// el detalle real solo va al log.
func jwtError(c *gin.Context, err error) {
	log.Printf("Error extrayendo datos del JWT: %v", err)
	helper.RespondError(c, errorsx.ErrUnauthorized)
}

func (h *Handler) Verify2FALogin(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	ctx := c.Request.Context()

	sessionID, err := c.Cookie("pre_auth_session")
	if err != nil || sessionID == "" {
		c.JSON(401, gin.H{"message": "Sesión de pre-autenticación no encontrada o expirada"})
		return
	}

	type Verify2FARequest struct {
		Code string `json:"code" binding:"required"`
	}

	var req Verify2FARequest
	if !helper.BindJSON(c, &req) {
		return
	}

	user, refreshToken, err := h.s.Verify2FALogin(ctx, sessionID, req.Code)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.SetCookie("pre_auth_session", "", -1, "/", getCookieDomain(c), isSecureCookie(), true)
	c.SetCookie("refresh_token", refreshToken, 8*60*60, "/", getCookieDomain(c), isSecureCookie(), true)

	c.JSON(200, gin.H{"data": user, "message": "Login exitoso"})
}

func (h *Handler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	var req RegisterRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	_, err := h.s.Register(ctx, req.Email, req.Password)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(201, gin.H{"message": "Usuario registrado exitosamente por favor verifica tu correo para activar tu cuenta"})
}

func (h *Handler) RegisterOwner(c *gin.Context) {
	ctx := c.Request.Context()

	var req RegisterRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	_, err := h.s.RegisterOwner(ctx, req.Email, req.Password)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(201, gin.H{"message": "Dueño registrado exitosamente por favor verifica tu correo para activar tu cuenta"})
}

func (h *Handler) Login(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	ctx := c.Request.Context()

	var req LoginRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	user, tokenOrSession, err := h.s.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, errorsx.ErrCodeEmailNotVerified) {
			c.JSON(403, gin.H{
				"message": "Correo no verificado",
				"code":    "EMAIL_NOT_VERIFIED",
			})
			return
		}

		if errors.Is(err, errorsx.ErrPending2FA) {
			c.SetCookie("pre_auth_session", tokenOrSession, 5*60, "/", getCookieDomain(c), isSecureCookie(), true)
			c.JSON(202, gin.H{
				"message": "Se requiere verificación de dos factores",
				"code":    "PENDING_2FA",
			})
			return
		}

		if errors.Is(err, errorsx.ErrInvalidCredentials) {
			log.Printf("Login fallido para un correo (credenciales inválidas)")
			c.JSON(401, gin.H{"message": "Invalid email or password"})
			return
		}

		helper.RespondError(c, err)
		return
	}

	log.Printf("Usuario %d inició sesión correctamente", user.ID)

	c.SetCookie("refresh_token", tokenOrSession, 8*60*60, "/", getCookieDomain(c), isSecureCookie(), true)

	c.JSON(200, gin.H{"data": user, "message": "Login successful"})
}

func (h *Handler) LoginMobile(c *gin.Context) {
	ctx := c.Request.Context()

	var req LoginRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	user, tokenOrSession, err := h.s.LoginMobile(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, errorsx.ErrCodeEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Correo no verificado",
				"code":    "EMAIL_NOT_VERIFIED",
			})
			return
		}

		if errors.Is(err, errorsx.ErrPending2FA) {
			c.JSON(http.StatusAccepted, gin.H{
				"message": "Se requiere verificación de dos factores",
				"code":    "PENDING_2FA",
				"data": gin.H{
					"session_id": tokenOrSession,
				},
			})
			return
		}

		if errors.Is(err, errorsx.ErrInvalidCredentials) {
			log.Printf("Login móvil fallido para un correo (credenciales inválidas)")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
			return
		}

		helper.RespondError(c, err)
		return
	}

	log.Printf("Usuario %d inició sesión correctamente (móvil)", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"user":          user,
			"token":         user.Token,
			"access_token":  user.Token,
			"refresh_token": tokenOrSession,
		},
	})
}

func (h *Handler) GetSession(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	session, err := h.s.GetSession(userID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"data": session, "message": "Session retrieved successfully"})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	token, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(400, gin.H{"message": "No refresh token provided"})
		return
	}

	newToken, err := h.s.RefreshToken(ctx, token)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Token refreshed successfully", "data": map[string]string{"token": newToken}})
}

func (h *Handler) RefreshTokenMobile(c *gin.Context) {
	ctx := c.Request.Context()

	token, _, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		} else {
			jwtError(c, err)
			return
		}
	}

	newAccessToken, newRefreshToken, err := h.s.RefreshTokenMobile(ctx, token)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"data": gin.H{
			"token":         newAccessToken,
			"access_token":  newAccessToken,
			"refresh_token": newRefreshToken,
		},
	})
}

func (h *Handler) ForwardEmailVerification(c *gin.Context) {
	ctx := c.Request.Context()
	type ForwardEmailRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req ForwardEmailRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err := h.s.ForwardEmailVerification(ctx, req.Email)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Verification email forwarded successfully"})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	type VerifyEmailRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req VerifyEmailRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err := h.s.VerifyEmail(req.Email)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Verification email sent successfully", "data": map[string]bool{"exist": true}})
}

func (h *Handler) ConfirmAccount(c *gin.Context) {
	ctx := c.Request.Context()
	token, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	err = h.s.ConfirmAccount(ctx, userID, token)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Tu cuenta ha sido confirmada exitosamente, ya puedes iniciar sesión"})
}

func (h *Handler) SendPasswordReset(c *gin.Context) {
	ctx := c.Request.Context()

	type SendPasswordResetRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req SendPasswordResetRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err := h.s.SendPasswordReset(ctx, req.Email)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Se ha enviado un correo para restablecer tu contraseña, por favor revisa tu bandeja de entrada"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	_, userId, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	type ResetPasswordRequest struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	var req ResetPasswordRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.ResetPassword(userId, req.NewPassword)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Tu contraseña ha sido restablecida exitosamente, ya puedes iniciar sesión con tu nueva contraseña"})
}

func (h *Handler) UpdatePassword(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	var req UpdatePasswordRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.UpdatePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Tu contraseña ha sido actualizada exitosamente, ya puedes iniciar sesión con tu nueva contraseña"})
}

func (h *Handler) EnableTWOFA(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	uri, secret, err := h.s.EnableTWOFA(userID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "2FA habilitado exitosamente", "data": map[string]string{"secret": secret, "uri": uri}})
}

func (h *Handler) DisableTWOFA(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	err = h.s.DisableTWOFA(userID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "2FA deshabilitado exitosamente"})
}

func (h *Handler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	token := ""
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(authHeader[7:])
	}

	if token == "" {
		if cookieToken, err := c.Cookie("refresh_token"); err == nil && cookieToken != "" {
			token = cookieToken
		}
	}

	if token != "" {
		_ = h.s.Logout(ctx, token)
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", "", -1, "/", getCookieDomain(c), isSecureCookie(), true)
	c.SetCookie("pre_auth_session", "", -1, "/", getCookieDomain(c), isSecureCookie(), true)

	c.JSON(http.StatusOK, gin.H{"message": "Sesión cerrada exitosamente"})
}
