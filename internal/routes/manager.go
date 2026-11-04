package routes

import (
	"fitbase/internal/features/manager"

	"github.com/gin-gonic/gin"
)

func ManagerRoutes(rg *gin.RouterGroup) {
	h := manager.NewHandler(managerUc)

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/managers", h.CreateManager)
		protected.GET("/managers/me", h.GetDetailsMe)
		protected.GET("/gyms/managers", h.GetAllManagersByGym)
		protected.POST("/managers/me/upload-avatar", h.UploadAvatar)
		protected.PUT("/managers/me", h.Update)
		protected.POST("/cash-register/open", h.OpenCashRegister)
		protected.POST("/cash-register/close", h.CloseCashRegister)
		protected.GET("/cash-register/current", h.GetCurrentCashRegister)
	}
}
