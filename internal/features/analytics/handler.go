package analytics

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

func (h *Handler) GetManagerShifts(c *gin.Context) {
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

	var query AnalyticsFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		helper.RespondError(c, errorsx.BadRequest("Parámetros de consulta inválidos"))
		return
	}
	query.Normalize()

	result, err := h.s.GetManagerShifts(ctx, authID, gymID, query.From, query.To, query.Page, query.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetAttendanceMetrics(c *gin.Context) {
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

	var query AnalyticsFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		helper.RespondError(c, errorsx.BadRequest("Parámetros de consulta inválidos"))
		return
	}
	query.Normalize()

	result, err := h.s.GetAttendanceMetrics(ctx, authID, gymID, query.From, query.To)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetCrossedRevenue(c *gin.Context) {
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

	var query AnalyticsFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		helper.RespondError(c, errorsx.BadRequest("Parámetros de consulta inválidos"))
		return
	}
	query.Normalize()

	result, err := h.s.GetCrossedRevenue(ctx, authID, gymID, query.From, query.To)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetClientHistory(c *gin.Context) {
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

	targetUserID, err := utils.ValidateParamsId(c, "user_id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de usuario inválido"))
		return
	}

	result, err := h.s.GetClientHistory(ctx, authID, gymID, targetUserID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetDashboardSummary(c *gin.Context) {
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

	result, err := h.s.GetDashboardSummary(ctx, authID, gymID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

