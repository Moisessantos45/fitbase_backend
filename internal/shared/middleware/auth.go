package middleware

import (
	"context"
	"fitbase/internal/shared/templates"
	"fitbase/internal/shared/utils"

	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware(maker *utils.PasetoMaker, rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		token := strings.TrimSpace(authHeader[7:])
		if token == "" || len(token) < 20 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		if rd != nil {
			blacklisted, err := rd.Exists(c.Request.Context(), fmt.Sprintf("blacklist:%s", token)).Result()
			if err == nil && blacklisted > 0 {
				c.AbortWithStatusJSON(401, gin.H{"msg": "token revoked"})
				return
			}
		}

		payload, err := maker.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		if payload == nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		if !validateSessionAccessToken(c.Request.Context(), rd, payload.UserID, payload.Platform, token) {
			c.AbortWithStatusJSON(401, gin.H{"msg": "session expired or invalid"})
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("payload", payload)
		c.Set("token", token)
		c.Next()
	}

}

func StorageAuthMiddleware(maker *utils.PasetoMaker, rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := ""

		if len(authHeader) >= 7 && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		} else {
			cookieToken, err := c.Cookie("refresh_token")
			if err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		if token == "" || len(token) < 20 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		if rd != nil {
			blacklisted, err := rd.Exists(c.Request.Context(), fmt.Sprintf("blacklist:%s", token)).Result()
			if err == nil && blacklisted > 0 {
				c.AbortWithStatusJSON(401, gin.H{"msg": "token revoked"})
				return
			}
		}

		payload, err := maker.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		if !validateStorageSessionToken(c.Request.Context(), rd, payload.UserID, payload.Platform, token) {
			c.AbortWithStatusJSON(401, gin.H{"msg": "session expired or invalid"})
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("payload", payload)
		c.Set("token", token)
		c.Next()
	}
}

// validateSessionAccessToken valida que para peticiones normales el token enviado sea estrictamente el token de corta duración (access_token)
func validateSessionAccessToken(ctx context.Context, rd *redis.Client, userID, platform, token string) bool {
	if rd == nil {
		return true
	}

	var sessionID string
	var err error

	if platform != "" {
		userSessionKey := fmt.Sprintf("user_session:%s:%s", platform, userID)
		sessionID, err = rd.Get(ctx, userSessionKey).Result()
	}

	if sessionID == "" {
		candidateKeys := []string{
			fmt.Sprintf("user_session:mobile:%s", userID),
			fmt.Sprintf("user_session:web:%s", userID),
			fmt.Sprintf("user_session:%s", userID),
		}
		for _, k := range candidateKeys {
			sessionID, err = rd.Get(ctx, k).Result()
			if err == nil && sessionID != "" {
				break
			}
		}
	}

	if sessionID == "" {
		// Validar tokens temporales de un solo uso (confirmación o reseteo de contraseña)
		confirmKey := fmt.Sprintf("%s:%s", templates.CONFIRM_ACCOUNT_PATH, userID)
		if storedToken, cErr := rd.Get(ctx, confirmKey).Result(); cErr == nil && storedToken == token {
			return true
		}

		resetKey := fmt.Sprintf("reset:%s", userID)
		if storedToken, rErr := rd.Get(ctx, resetKey).Result(); rErr == nil && storedToken == token {
			return true
		}

		return false
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	accToken, err := rd.HGet(ctx, sessionKey, "access_token").Result()
	if err != nil || accToken == "" {
		return false
	}

	return token == accToken
}

// validateStorageSessionToken valida el token para almacenamiento estático, permitiendo access_token o refresh_token (cookie de navegador)
func validateStorageSessionToken(ctx context.Context, rd *redis.Client, userID, platform, token string) bool {
	if rd == nil {
		return true
	}

	var sessionID string
	var err error

	if platform != "" {
		userSessionKey := fmt.Sprintf("user_session:%s:%s", platform, userID)
		sessionID, err = rd.Get(ctx, userSessionKey).Result()
	}

	if sessionID == "" {
		candidateKeys := []string{
			fmt.Sprintf("user_session:web:%s", userID),
			fmt.Sprintf("user_session:mobile:%s", userID),
			fmt.Sprintf("user_session:%s", userID),
		}
		for _, k := range candidateKeys {
			sessionID, err = rd.Get(ctx, k).Result()
			if err == nil && sessionID != "" {
				break
			}
		}
	}

	if sessionID == "" {
		return false
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	storedTokens, err := rd.HMGet(ctx, sessionKey, "access_token", "refresh_token").Result()
	if err != nil || len(storedTokens) < 2 {
		return false
	}

	accToken, _ := storedTokens[0].(string)
	refToken, _ := storedTokens[1].(string)

	return token == accToken || token == refToken
}

func PreAuthMiddleware(rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("pre_auth_session")
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación requerida"})
			return
		}

		key := fmt.Sprintf("preauth:%s", sessionID)
		vals, err := rd.HGetAll(context.Background(), key).Result()
		if err != nil || len(vals) == 0 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación expirada o inválida"})
			return
		}

		authStage, ok := vals["auth_stage"]
		if !ok || authStage != "pending_2fa" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "estado de autenticación inválido"})
			return
		}

		c.Set("pre_auth_session_id", sessionID)
		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = utils.GenerateID()
		}
		c.Set("RequestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Next()
	}
}
