package subscription

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	s Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) PurchaseSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req PurchaseSubscriptionRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	ticket, err := h.s.PurchaseSubscription(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Suscripción adquirida exitosamente",
		"data":    ticket,
	})
}

func (h *Handler) GetMySubscriptions(c *gin.Context) {
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

	result, err := h.s.GetMySubscriptions(ctx, authID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetTicketBySlug(c *gin.Context) {
	ctx := c.Request.Context()

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.BadRequest("Código de ticket no proporcionado"))
		return
	}

	ticket, err := h.s.GetTicketBySlug(ctx, slug)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ticket})
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

	var statusID uint64
	if statusStr := c.Query("status_id"); statusStr != "" {
		if parsed, err := strconv.ParseUint(statusStr, 10, 64); err == nil {
			statusID = parsed
		}
	}

	result, err := h.s.GetAllByGym(ctx, authID, gymID, statusID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	subID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de suscripción inválido"))
		return
	}

	ticket, err := h.s.CancelSubscription(ctx, authID, subID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	msg := "Suscripción cancelada exitosamente"
	if ticket.RemainingDays > 0 {
		msg = fmt.Sprintf("Suscripción cancelada exitosamente. Cuentas con %d días de acceso restantes hasta el %s", ticket.RemainingDays, ticket.EndDate)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": msg,
		"data":    ticket,
	})
}
