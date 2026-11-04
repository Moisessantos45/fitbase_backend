package daypass

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	s Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) PurchaseDayPass(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CreateDayPassRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	ticket, err := h.s.PurchaseDayPass(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pase de un día generado exitosamente",
		"data":    ticket,
	})
}

func (h *Handler) RegisterGuest(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req RegisterGuestRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	ticket, err := h.s.RegisterGuest(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Invitado registrado exitosamente",
		"data":    ticket,
	})
}

func (h *Handler) CreatePublicDayPass(c *gin.Context) {
	ctx := c.Request.Context()

	var req RegisterGuestRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	ticket, err := h.s.CreatePublicDayPass(ctx, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pase diario generado exitosamente",
		"data":    ticket,
	})
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

	dateStr := c.DefaultQuery("date", "")
	search := c.DefaultQuery("search", c.DefaultQuery("query", ""))

	result, err := h.s.GetAllByGym(ctx, authID, gymID, dateStr, search, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
