package routes

import (
	"fitbase/internal/features/owner"

	"github.com/gin-gonic/gin"
)

func OwnerRoutes(rg *gin.RouterGroup) {
	h := owner.NewHandler(ownerUc)

	protected := rg.Group("/owner")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/complete-profile", h.CompleteProfile)
		protected.GET("/me", h.GetProfile)
		protected.PUT("/me", h.UpdateProfile)
		protected.POST("/me/upload-avatar", h.UploadAvatar)
	}
}
