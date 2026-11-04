package routes

import (
	"fitbase/internal/features/subscription"

	"github.com/gin-gonic/gin"
)

func SubscriptionRoutes(rg *gin.RouterGroup) {
	h := subscription.NewHandler(subscriptionUc)

	public := rg.Group("")
	public.Use(idempotencyMiddleware())
	{
		public.GET("/subscriptions/:slug/ticket", h.GetTicketBySlug)
	}

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/subscriptions/subscribe", h.PurchaseSubscription)
		protected.GET("/subscriptions/my-subscriptions", h.GetMySubscriptions)
		protected.GET("/gyms/:id/subscriptions", h.GetAllByGym)
		protected.PUT("/subscriptions/:id/cancel", h.CancelSubscription)
	}
}
