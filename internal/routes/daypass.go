package routes

import (
	"fitbase/internal/features/daypass"

	"github.com/gin-gonic/gin"
)

func DayPassRoutes(rg *gin.RouterGroup) {
	h := daypass.NewHandler(dayPassUc)

	public := rg.Group("")
	public.Use(idempotencyMiddleware())
	{
		public.GET("/daypass/:slug/ticket", h.GetTicketBySlug)
		public.POST("/daypass/guest", h.CreatePublicDayPass)
	}

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/daypass/purchase", h.PurchaseDayPass)
		protected.POST("/daypass/register-guest", h.RegisterGuest)
		protected.GET("/gyms/:id/daypasses", h.GetAllByGym)
	}
}
