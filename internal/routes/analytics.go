package routes

import (
	"fitbase/internal/features/analytics"

	"github.com/gin-gonic/gin"
)

func AnalyticsRoutes(rg *gin.RouterGroup) {
	h := analytics.NewHandler(analyticsUc)

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.GET("/gyms/:id/analytics/shifts", h.GetManagerShifts)
		protected.GET("/gyms/:id/analytics/attendance", h.GetAttendanceMetrics)
		protected.GET("/gyms/:id/analytics/revenue", h.GetCrossedRevenue)
		protected.GET("/gyms/:id/analytics/clients/:user_id", h.GetClientHistory)
		protected.GET("/gyms/:id/analytics/dashboard-summary", h.GetDashboardSummary)
	}
}
