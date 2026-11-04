package user

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CompleteProfileRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string `json:"last_name" binding:"required,min=2,max=100"`
	Phone     string `json:"phone" binding:"required,min=7,max=20"`
}

type UpdateRequest struct {
	FirstName string `json:"first_name" binding:"omitempty,min=2,max=100"`
	LastName  string `json:"last_name" binding:"omitempty,min=2,max=100"`
	Phone     string `json:"phone" binding:"omitempty,min=7,max=20"`
}

type Handler struct {
	s Service
}

func NewHandler(uc Service) *Handler {
	return &Handler{s: uc}
}

func (h *Handler) CompleteProfile(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CompleteProfileRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	profile, err := h.s.CompleteProfile(ctx, authID, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Perfil de dueño completado exitosamente",
		"data":    profile,
	})
}

func (h *Handler) GetByAuthID(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		log.Printf("Error extrayendo datos del JWT: %v", err)
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	user, err := h.s.GetByAuthID(c.Request.Context(), authID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"data": user, "message": "User retrieved successfully"})
}

func (h *Handler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		log.Printf("Error extrayendo datos del JWT: %v", err)
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req UpdateRequest
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

	slug, path, err := utils.SaveFile("users/avatars", "", file, c)
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
