package payment

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	s Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) RegisterCashPayment(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req RegisterPaymentRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	receipt, err := h.s.RegisterCashPayment(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pago registrado exitosamente",
		"data":    receipt,
	})
}

func (h *Handler) GetRevenueSummary(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	gymID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de gimnasio inválido"))
		return
	}

	summary, err := h.s.GetRevenueSummary(ctx, authID, gymID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

func (h *Handler) GetAllByGym(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	gymID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de gimnasio inválido"))
		return
	}

	pagination, err := utils.ParsePagination(c)
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest(err.Error()))
		return
	}

	result, err := h.s.GetAllByGym(ctx, authID, gymID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetMyPayments(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	pagination, err := utils.ParsePagination(c)
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest(err.Error()))
		return
	}

	result, err := h.s.GetMyPayments(ctx, authID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) CreateCheckoutSession(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CreateCheckoutSessionRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	result, err := h.s.CreateCheckoutSession(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sesión de checkout creada exitosamente",
		"data":    result,
	})
}

func (h *Handler) CreateGuestCheckoutSession(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateGuestCheckoutSessionRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	result, err := h.s.CreateGuestCheckoutSession(ctx, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sesión de checkout para invitado creada exitosamente",
		"data":    result,
	})
}

func (h *Handler) HandleStripeWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	signature := c.GetHeader("Stripe-Signature")
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("Error al leer el cuerpo del webhook"))
		return
	}

	if err := h.s.HandleStripeWebhook(ctx, payload, signature); err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *Handler) GetSessionStatus(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, _ := utils.ExtractedParamsJwt(c)

	sessionID := c.Param("session_id")
	if sessionID == "" {
		helper.RespondError(c, errorsx.BadRequest("ID de sesión requerido"))
		return
	}

	status, err := h.s.GetSessionStatus(ctx, authID, sessionID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}
