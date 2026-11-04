package storage

import (
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UpdatePermissionRequest struct {
	PermissionID uint64 `json:"permission_id" binding:"required"`
}

type Handler struct {
	uc Service
}

func NewStorageHandler(uc Service) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	file, err := h.uc.GetBySlug(c.Request.Context(), slug)
	if err != nil || file == nil {
		helper.RespondError(c, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	utils.ServeFileSecurely(c, file.Path, "")
}

func (h *Handler) GetBySlugPublic(c *gin.Context) {
	slug := c.Param("slug")

	file, err := h.uc.GetBySlugPublic(c.Request.Context(), slug)
	if err != nil {
		if err == ErrFileNotFound || err == ErrFileIsPrivate {
			helper.RespondError(c, ErrFileNotFound)
			return
		}
		helper.RespondError(c, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	utils.ServeFileSecurely(c, file.Path, "")
}

func (h *Handler) GetBySlugPrivate(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	fileId := c.Param("slug")

	file, err := h.uc.GetBySlugPrivate(c.Request.Context(), authID, fileId)
	if err != nil {
		if err == ErrFileNotFound || err == ErrUnauthorizedFile {
			helper.RespondError(c, ErrFileNotFound)
			return
		}
		helper.RespondError(c, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	utils.ServeFileSecurely(c, file.Path, "")
}

func (h *Handler) DeleteBySlug(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	fileId := c.Param("slug")

	err = h.uc.Delete(c.Request.Context(), authID, fileId)
	if err != nil {
		if err == ErrFileNotFound || err == ErrUnauthorizedFile {
			helper.RespondError(c, ErrFileNotFound)
			return
		}
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
