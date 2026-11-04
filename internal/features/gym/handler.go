package gym

import (
	"encoding/json"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var req struct {
	Name      string                   `json:"name" binding:"required,min=2,max=100"`
	Address   string                   `json:"address" binding:"required,min=5,max=255"`
	Phone     string                   `json:"phone" binding:"required,min=7,max=20"`
	Email     string                   `json:"email" binding:"required,email"`
	Latitude  float64                  `json:"latitude" binding:"required"`
	Longitude float64                  `json:"longitude" binding:"required"`
	Schedules []GymScheduleItemRequest `json:"schedules"`
}

type CreateGymRequest struct {
	Name      string  `json:"name" form:"name" binding:"required,min=2,max=100"`
	Address   string  `json:"address" form:"address" binding:"required,min=5,max=255"`
	Phone     string  `json:"phone" form:"phone" binding:"required,min=7,max=20"`
	Email     string  `json:"email" form:"email" binding:"required,email"`
	Latitude  float64 `json:"latitude" form:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" form:"longitude" binding:"required"`
}

type UpdateGymRequest struct {
	Name      string   `json:"name" binding:"omitempty,min=2,max=100"`
	Address   string   `json:"address" binding:"omitempty,min=5,max=255"`
	Phone     string   `json:"phone" binding:"omitempty,min=7,max=20"`
	Email     string   `json:"email" binding:"omitempty,email"`
	Latitude  *float64 `json:"latitude" binding:"omitempty"`
	Longitude *float64 `json:"longitude" binding:"omitempty"`
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

	var name, address, phone, email string
	var lat, lng float64
	var schedules []GymScheduleItemRequest

	if strings.Contains(c.ContentType(), "application/json") {

		if !helper.BindJSON(c, &req) {
			return
		}
		name = req.Name
		address = req.Address
		phone = req.Phone
		email = req.Email
		lat = req.Latitude
		lng = req.Longitude
		schedules = req.Schedules
	} else {
		name = strings.TrimSpace(c.PostForm("name"))
		address = strings.TrimSpace(c.PostForm("address"))
		phone = strings.TrimSpace(c.PostForm("phone"))
		email = strings.TrimSpace(c.PostForm("email"))

		if name == "" || address == "" || phone == "" || email == "" {
			helper.RespondError(c, errorsx.BadRequest("Nombre, dirección, teléfono y correo electrónico son requeridos"))
			return
		}

		latStr := c.PostForm("latitude")
		if latStr == "" {
			latStr = c.PostForm("lat")
		}
		lat, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			helper.RespondError(c, errorsx.BadRequest("Latitud inválida"))
			return
		}

		lngStr := c.PostForm("longitude")
		if lngStr == "" {
			lngStr = c.PostForm("lng")
		}
		lng, err = strconv.ParseFloat(lngStr, 64)
		if err != nil {
			helper.RespondError(c, errorsx.BadRequest("Longitud inválida"))
			return
		}

		schedulesJSON := strings.TrimSpace(c.PostForm("schedules"))
		if schedulesJSON != "" {
			if err := json.Unmarshal([]byte(schedulesJSON), &schedules); err != nil {
				helper.RespondError(c, errorsx.BadRequest("Formato JSON inválido para 'schedules': "+err.Error()))
				return
			}
		} else {
			scheduleCountStr := c.PostForm("schedule_count")
			scheduleCount := 0
			if scheduleCountStr != "" {
				scheduleCount, _ = strconv.Atoi(scheduleCountStr)
			} else {
				for i := range [7]int{} {
					if c.PostForm("schedules["+strconv.Itoa(i)+"][day_of_week]") != "" {
						scheduleCount = i + 1
					}
				}
			}

			for i := 0; i < scheduleCount; i++ {
				idx := strconv.Itoa(i)
				dayOfWeekStr := c.PostForm("schedules[" + idx + "][day_of_week]")
				if dayOfWeekStr != "" {
					dayOfWeek, err := strconv.Atoi(dayOfWeekStr)
					if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
						helper.RespondError(c, errorsx.BadRequest(fmt.Sprintf("Día de la semana inválido en índice %d (0 a 6)", i)))
						return
					}

					openMin, _ := strconv.Atoi(c.PostForm("schedules[" + idx + "][open_minutes]"))
					closeMin, _ := strconv.Atoi(c.PostForm("schedules[" + idx + "][close_minutes]"))
					isClosedStr := c.PostForm("schedules[" + idx + "][is_closed]")
					isClosed := isClosedStr == "true" || isClosedStr == "1"

					schedules = append(schedules, GymScheduleItemRequest{
						DayOfWeek:    dayOfWeek,
						OpenMinutes:  openMin,
						CloseMinutes: closeMin,
						IsClosed:     isClosed,
					})
				}
			}
		}
	}

	// 1. Extraer imagen de portada independiente (opcional)
	var coverSaved *SavedFile
	coverHeader, coverSize, coverMime, coverErr := utils.ExtractImage(c, "cover")
	if coverErr != nil {
		coverHeader, coverSize, coverMime, coverErr = utils.ExtractImage(c, "background")
	}
	if coverErr == nil && coverHeader != nil {
		fileID, path, saveErr := utils.SaveFile("gyms/covers", "", coverHeader, c)
		if saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error al guardar imagen de portada: " + saveErr.Error()})
			return
		}
		coverSaved = &SavedFile{
			FileID:   fileID,
			Path:     path,
			MimeType: coverMime,
			Size:     coverSize,
		}
	}

	// 2. Extraer imágenes de la galería (opcional)
	var savedFiles []SavedFile
	var uploadedFiles []models.UploadedFile
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for key, headers := range form.File {
			if key == "cover" || key == "background" {
				continue
			}
			for _, file := range headers {
				if file == nil || file.Size <= 0 {
					continue
				}
				if file.Size > utils.MaxImageSize {
					if coverSaved != nil {
						_ = utils.DeleteFile(coverSaved.Path)
					}
					helper.RespondError(c, errorsx.BadRequest(fmt.Sprintf("La imagen '%s' supera el tamaño máximo permitido (5MB)", file.Filename)))
					return
				}
				mimeType, vErr := utils.ValidateImageHeader(file)
				if vErr != nil {
					if coverSaved != nil {
						_ = utils.DeleteFile(coverSaved.Path)
					}
					helper.RespondError(c, errorsx.BadRequest(fmt.Sprintf("Archivo '%s': %s", file.Filename, vErr.Error())))
					return
				}
				uploadedFiles = append(uploadedFiles, models.UploadedFile{
					File:     file,
					Size:     file.Size,
					MimeType: mimeType,
				})
			}
		}
	}

	if len(uploadedFiles) > 0 {
		for _, f := range uploadedFiles {
			fileID, path, saveErr := utils.SaveFile("gyms/photos", "", f.File, c)
			if saveErr != nil {
				if coverSaved != nil {
					_ = utils.DeleteFile(coverSaved.Path)
				}
				for _, sf := range savedFiles {
					_ = utils.DeleteFile(sf.Path)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error al guardar foto de galería: " + saveErr.Error()})
				return
			}

			savedFiles = append(savedFiles, SavedFile{
				FileID:   fileID,
				Path:     path,
				MimeType: f.MimeType,
				Size:     f.Size,
			})
		}
	}

	gym, err := h.s.Create(ctx, authID, name, address, phone, email, lat, lng, schedules, coverSaved, savedFiles)
	if err != nil {
		if coverSaved != nil {
			_ = utils.DeleteFile(coverSaved.Path)
		}
		for _, sf := range savedFiles {
			_ = utils.DeleteFile(sf.Path)
		}
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Gimnasio creado exitosamente",
		"data":    gym,
	})
}

func (h *Handler) Update(c *gin.Context) {
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

	var req UpdateGymRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.Update(ctx, authID, gymID, req.Name, req.Address, req.Phone, req.Email, req.Latitude, req.Longitude)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gimnasio actualizado exitosamente"})
}

func (h *Handler) SetSchedules(c *gin.Context) {
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

	var req SetScheduleRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.SetSchedules(ctx, authID, gymID, req.Schedules)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Horarios actualizados exitosamente"})
}

func (h *Handler) GetDetailsByID(c *gin.Context) {
	ctx := c.Request.Context()

	gymID, err := utils.ValidateParamsId(c, "id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de gimnasio inválido"))
		return
	}

	detail, err := h.s.GetDetailsByID(ctx, gymID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func (h *Handler) GetMyGymDetails(c *gin.Context) {
	ctx := c.Request.Context()

	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		helper.RespondError(c, errorsx.ErrUnauthorized)
		return
	}

	detail, err := h.s.GetMyGymDetails(ctx, authID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func (h *Handler) GetAllByOwner(c *gin.Context) {
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

	result, err := h.s.GetAllByOwner(ctx, authID, pagination.Page, pagination.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetAllPublic(c *gin.Context) {
	ctx := c.Request.Context()

	var query SearchGymQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		helper.RespondError(c, errorsx.BadRequest("Parámetros de búsqueda inválidos"))
		return
	}
	query.Normalize()

	result, err := h.s.GetAllPublic(ctx, query.Search, query.Latitude, query.Longitude, query.Page, query.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) UpdateSchedule(c *gin.Context) {
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

	scheduleID, err := utils.ValidateParamsId(c, "schedule_id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de horario inválido"))
		return
	}

	var req UpdateGymScheduleRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.s.UpdateSchedule(ctx, authID, gymID, scheduleID, req)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Horario actualizado exitosamente"})
}

func (h *Handler) UploadPhotos(c *gin.Context) {
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

	form, err := c.MultipartForm()
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("No se pudo procesar el formulario de imágenes: "+err.Error()))
		return
	}

	var uploadedFiles []models.UploadedFile
	for _, headers := range form.File {
		for _, file := range headers {
			if file == nil || file.Size <= 0 {
				continue
			}
			if file.Size > utils.MaxImageSize {
				helper.RespondError(c, errorsx.BadRequest(fmt.Sprintf("La imagen '%s' supera el tamaño máximo permitido (5MB)", file.Filename)))
				return
			}
			mimeType, vErr := utils.ValidateImageHeader(file)
			if vErr != nil {
				helper.RespondError(c, errorsx.BadRequest(fmt.Sprintf("Archivo '%s': %s", file.Filename, vErr.Error())))
				return
			}
			uploadedFiles = append(uploadedFiles, models.UploadedFile{
				File:     file,
				Size:     file.Size,
				MimeType: mimeType,
			})
		}
	}

	if len(uploadedFiles) == 0 {
		helper.RespondError(c, errorsx.BadRequest("No se enviaron imágenes válidas (JPG, PNG o WEBP)"))
		return
	}

	var savedFiles []SavedFile
	for _, f := range uploadedFiles {
		fileID, path, saveErr := utils.SaveFile("gyms/photos", "", f.File, c)
		if saveErr != nil {
			for _, sf := range savedFiles {
				_ = utils.DeleteFile(sf.Path)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error al guardar archivo: " + saveErr.Error()})
			return
		}

		savedFiles = append(savedFiles, SavedFile{
			FileID:   fileID,
			Path:     path,
			MimeType: f.MimeType,
			Size:     f.Size,
		})
	}

	err = h.s.UploadPhotos(ctx, authID, gymID, savedFiles)
	if err != nil {
		for _, sf := range savedFiles {
			_ = utils.DeleteFile(sf.Path)
		}
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Imágenes subidas exitosamente"})
}

func (h *Handler) DeletePhoto(c *gin.Context) {
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

	photoID, err := utils.ValidateParamsId(c, "photo_id")
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("ID de fotografía inválido"))
		return
	}

	err = h.s.DeletePhoto(ctx, authID, gymID, photoID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fotografía eliminada exitosamente"})
}

func (h *Handler) UploadCover(c *gin.Context) {
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

	file, size, mimeType, err := utils.ExtractImage(c, "image")
	if err != nil {
		file, size, mimeType, err = utils.ExtractImage(c, "cover")
		if err != nil {
			file, size, mimeType, err = utils.ExtractImage(c, "background")
		}
	}
	if err != nil {
		helper.RespondError(c, errorsx.BadRequest("No se proporcionó una imagen válida para la portada (image, cover o background)"))
		return
	}

	slug, path, err := utils.SaveFile("gyms/covers", "", file, c)
	if err != nil {
		_ = utils.DeleteFile(path)
		helper.RespondError(c, err)
		return
	}

	backgroundURL, err := h.s.UploadCover(c.Request.Context(), authID, gymID, slug, path, mimeType, size)
	if err != nil {
		_ = utils.DeleteFile(path)
		helper.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Foto de portada actualizada exitosamente",
		"data": gin.H{
			"background_url": backgroundURL,
		},
	})
}
