package routes

import (
	"fitbase/internal/features/plan"

	"github.com/gin-gonic/gin"
)

func PlanRoutes(rg *gin.RouterGroup) {
	h := plan.NewHandler(planUc)

	public := rg.Group("")
	public.Use(idempotencyMiddleware())
	{
		public.GET("/plans/:id", h.GetDetailsByID)
		public.GET("/gyms/:id/plans", h.GetAllActiveByGymID)
	}

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/plans", h.Create)
		protected.PUT("/plans/:id", h.Update)
		protected.GET("/gyms/:id/all-plans", h.GetAllByGymID)
		protected.GET("/gyms/:id/plans/enrollment", h.GetEnrollmentPlans)

		protected.POST("/plans/:id/items", h.AddItem)
		protected.PUT("/plans/:id/items/:item_id", h.UpdateItem)
		protected.DELETE("/plans/:id/items/:item_id", h.DeleteItem)
	}
}
