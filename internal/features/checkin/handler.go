package checkin

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

func (h *Handler) ValidateAccess(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req ValidateAccessRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	result, err := h.s.ValidateAccess(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Acceso validado exitosamente",
		"data":    result,
	})
}

func (h *Handler) GetMyHistory(c *gin.Context) {
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

	result, err := h.s.GetMyHistory(ctx, authID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
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

	fromStr := c.Query("from")
	toStr := c.Query("to")

	result, err := h.s.GetAllByGym(ctx, authID, gymID, fromStr, toStr, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
