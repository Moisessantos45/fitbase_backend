package manager

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	s Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s: s}
}

func (h *Handler) CreateManager(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CreateRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.Create(ctx, authID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Encargado registrado exitosamente. Se ha enviado un correo de confirmación.",
	})
}

func (h *Handler) GetAllManagersByGym(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	query, err := utils.ParseManagerQuery(c)
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest(err.Error()))
		return
	}

	managers, err := h.s.GetAllManagersByGym(ctx, authID, query.Page, query.PageSize, query.Query)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": managers})
}

func (h *Handler) OpenCashRegister(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req OpenCashRegisterRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	register, err := h.s.OpenCashRegister(ctx, authID, req.GymID, req.OpenCash)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Caja abierta exitosamente",
		"data":    register,
	})
}

func (h *Handler) CloseCashRegister(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CloseCashRegisterRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	register, err := h.s.CloseCashRegister(ctx, authID, req.CloseCash)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Caja cerrada exitosamente",
		"data":    register,
	})
}

func (h *Handler) GetCurrentCashRegister(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	register, err := h.s.GetCurrentCashRegister(ctx, authID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": register})
}

func (h *Handler) GetDetailsMe(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	manager, err := h.s.GetDetailByAuthID(ctx, authID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": manager})
}

func (h *Handler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		log.Printf("Error extrayendo datos del JWT: %v", err)
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req struct {
		FirstName string `json:"first_name" binding:"omitempty,max=100"`
		LastName  string `json:"last_name" binding:"omitempty,max=100"`
		Phone     string `json:"phone" binding:"omitempty,max=20"`
	}
	if !helper.BindJSON(c, &req) {
		return
	}

	if err = h.s.Update(c.Request.Context(), authID, &req.FirstName, &req.LastName, &req.Phone); err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "User updated successfully"})
}

func (h *Handler) UploadAvatar(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	file, size, mimeType, err := utils.ExtractImage(c, "image")
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	slug, path, err := utils.SaveFile("managers/avatars", "", file, c)
	if err != nil {
		_ = utils.DeleteFile(path)
		helper.RespondError(c, err)
		return
	}

	err = h.s.UploadAvatar(c.Request.Context(), authID, slug, path, mimeType, size)
	if err != nil {
		_ = utils.DeleteFile(path)
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Avatar uploaded successfully"})
}
