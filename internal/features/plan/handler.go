package plan

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreatePlanRequest struct {
	GymID       uint64            `json:"gym_id" binding:"required"`
	Name        string            `json:"name" binding:"required,min=2,max=100"`
	Description string            `json:"description" binding:"required,max=255"`
	Price       float64           `json:"price" binding:"required,gt=0"`
	Discount    float64           `json:"discount" binding:"gte=0,lte=100"`
	Duration    int               `json:"duration" binding:"required,gte=1"`
	PlanType    string            `json:"plan_type" binding:"required"`
	MaxVisits   *int              `json:"max_visits" binding:"omitempty"`
	Items       []PlanItemRequest `json:"items" binding:"omitempty,dive"`
}

type UpdatePlanRequest struct {
	Name        string   `json:"name" binding:"omitempty,min=2,max=100"`
	Description string   `json:"description" binding:"omitempty,max=255"`
	Price       *float64 `json:"price" binding:"omitempty,gt=0"`
	Discount    *float64 `json:"discount" binding:"omitempty,gte=0,lte=100"`
	Duration    *int     `json:"duration" binding:"omitempty,gte=1"`
	PlanType    string   `json:"plan_type" binding:"omitempty"`
	MaxVisits   *int     `json:"max_visits" binding:"omitempty"`
	Active      *bool    `json:"active" binding:"omitempty"`
}

type UpdatePlanItemRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

type Handler struct {
	s Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CreatePlanRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	plan, err := h.s.Create(ctx, authID, req.GymID, req.Name, req.Description, req.PlanType, req.Price, req.Discount, req.Duration, req.MaxVisits, req.Items)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Plan creado exitosamente",
		"data":    plan,
	})
}

func (h *Handler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	planID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de plan inválido"))
		return
	}

	var req UpdatePlanRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.Update(ctx, authID, planID, req.Name, req.Description, req.PlanType, req.Price, req.Discount, req.Duration, req.MaxVisits, req.Active)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plan actualizado exitosamente"})
}

func (h *Handler) GetDetailsByID(c *gin.Context) {
	ctx := c.Request.Context()

	planID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de plan inválido"))
		return
	}

	plan, err := h.s.GetDetailsByID(ctx, planID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *Handler) GetAllActiveByGymID(c *gin.Context) {
	ctx := c.Request.Context()

	gymID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de gimnasio inválido"))
		return
	}

	plans, err := h.s.GetAllByGymID(ctx, 0, gymID, true)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}

func (h *Handler) GetAllByGymID(c *gin.Context) {
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

	plans, err := h.s.GetAllByGymID(ctx, authID, gymID, false)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}

func (h *Handler) GetEnrollmentPlans(c *gin.Context) {
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

	response, err := h.s.GetEnrollmentPlans(ctx, authID, gymID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (h *Handler) AddItem(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	planID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de plan inválido"))
		return
	}

	var req PlanItemRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	item, err := h.s.AddItem(ctx, authID, planID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Beneficio agregado exitosamente",
		"data":    item,
	})
}

func (h *Handler) UpdateItem(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	planID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de plan inválido"))
		return
	}

	itemID, err := utils.ValidateParamsId(c, "item_id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de beneficio inválido"))
		return
	}

	var req UpdatePlanItemRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.UpdateItem(ctx, authID, planID, itemID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Beneficio actualizado exitosamente"})
}

func (h *Handler) DeleteItem(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	planID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de plan inválido"))
		return
	}

	itemID, err := utils.ValidateParamsId(c, "item_id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de beneficio inválido"))
		return
	}

	err = h.s.DeleteItem(ctx, authID, planID, itemID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Beneficio eliminado exitosamente"})
}
