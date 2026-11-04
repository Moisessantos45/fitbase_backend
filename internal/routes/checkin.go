package routes

import (
	"fitbase/internal/features/checkin"

	"github.com/gin-gonic/gin"
)

func CheckInRoutes(rg *gin.RouterGroup) {
	h := checkin.NewHandler(checkInUc)

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/checkin/scan", h.ValidateAccess)
		protected.GET("/checkin/my-history", h.GetMyHistory)
		protected.GET("/gyms/:id/checkins", h.GetAllByGym)
	}
}
