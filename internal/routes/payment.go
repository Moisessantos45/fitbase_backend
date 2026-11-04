package routes

import (
	"fitbase/internal/features/payment"

	"github.com/gin-gonic/gin"
)

func PaymentRoutes(rg *gin.RouterGroup) {
	h := payment.NewHandler(paymentUc)

	public := rg.Group("")
	{
		public.POST("/payments/stripe/webhook", h.HandleStripeWebhook)
		public.POST("/payments/stripe/guest-checkout", h.CreateGuestCheckoutSession)
		public.GET("/payments/stripe/session/:session_id", h.GetSessionStatus)
	}

	protected := rg.Group("")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("/payments/cash", h.RegisterCashPayment)
		protected.GET("/gyms/:id/payments/revenue", h.GetRevenueSummary)
		protected.GET("/gyms/:id/payments", h.GetAllByGym)
		protected.GET("/payments/my-payments", h.GetMyPayments)
		protected.POST("/payments/stripe/checkout-session", h.CreateCheckoutSession)
	}
}
