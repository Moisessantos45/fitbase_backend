package routes

import (
	"fitbase/internal/features/user"

	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup) {
	h := user.NewHandler(userUc)

	protected := rg.Group("/users")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.GET("/me", h.GetByAuthID)
		protected.POST("", h.CompleteProfile)
		protected.POST("/me/upload-avatar", h.UploadAvatar)
		protected.POST("/users/me/upload-avatar", h.UploadAvatar)
		protected.PUT("/me", h.Update)
		// protected.PATCH("/me/update-access-count", h.UpdateAccessCount)
	}
}
