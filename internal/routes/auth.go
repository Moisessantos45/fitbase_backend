package routes

import (
	"fitbase/config"
	"fitbase/internal/features/auth"
	"fitbase/internal/shared/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func AuthRoutes(rg *gin.RouterGroup) {
	s := authUc
	h := auth.NewAuthHandler(s)

	// Límite estricto por IP para endpoints sensibles (fuerza bruta, spam de correos, registros masivos):
	// 10 peticiones/minuto con ráfaga de 5, compartido entre instancias vía Redis. Independiente del global.
	strict := middleware.RateLimiterRedis(config.Rdb, "auth", rate.Every(time.Minute/10), 5)

	public := rg.Group("")
	public.Use(idempotencyMiddleware())
	{
		public.POST("/login", strict, h.Login)
		public.POST("/login-mobile", strict, h.LoginMobile)
		public.POST("/forward-email-verification", strict, h.ForwardEmailVerification)
		public.POST("/forgot-password", strict, h.SendPasswordReset)
		public.POST("/register", strict, h.Register)
		public.POST("/register-owner", strict, h.RegisterOwner)
		public.POST("/logout", h.Logout)
		public.POST("/logout-mobile", h.Logout)
		public.POST("/refresh-token", h.RefreshToken)
		public.POST("/refresh-token-mobile", h.RefreshTokenMobile)
	}

	protected := rg.Group("/")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	// protected.Use(middleware.RateLimiterMiddleware(rate.Every(time.Minute/300), 300))
	{
		protected.GET("/confirm-account", h.ConfirmAccount)
		protected.GET("/session", h.GetSession)
		protected.POST("/verify-email", h.VerifyEmail)
		protected.PATCH("/reset-password", h.ResetPassword)
		protected.PATCH("/change-password", h.UpdatePassword)

	}
}
