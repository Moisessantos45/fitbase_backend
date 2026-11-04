package owner

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

func (h *Handler) CompleteProfile(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req CompleteOwnerProfileRequest
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

func (h *Handler) GetProfile(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	profile, err := h.s.GetProfile(ctx, authID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	var req UpdateRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	if err := h.s.Update(ctx, authID, &req.FirstName, &req.LastName, &req.Phone); err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Perfil de dueño actualizado exitosamente"})
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

	slug, path, err := utils.SaveFile("owners/avatars", "", file, c)
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
