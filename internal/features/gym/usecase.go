package gym

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/service"
	"fitbase/internal/shared/utils"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

type UseCase struct {
	repo  Repository
	cache *service.CacheService
}

func NewUseCase(repo Repository, rd *redis.Client) Service {
	return &UseCase{
		repo:  repo,
		cache: service.NewCacheService(rd),
	}
}

func calculateIsOpen(schedules []GymScheduleItem, t time.Time) bool {
	dayOfWeek := int(t.Weekday())
	currentMinutes := t.Hour()*60 + t.Minute()

	for _, s := range schedules {
		if s.DayOfWeek == dayOfWeek {
			if s.IsClosed {
				return false
			}
			return currentMinutes >= s.OpenMinutes && currentMinutes <= s.CloseMinutes
		}
	}

	return false
}

func (uc *UseCase) Create(
	ctx context.Context,
	authID uint64,
	name, address, phone, email string,
	lat, lng float64,
	schedules []GymScheduleItemRequest,
	cover *SavedFile,
	photos []SavedFile,
) (*models.Gym, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("ID de autenticación inválido")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de dueño no encontrado")
	}

	gym, err := New(ownerID, name, address, phone, email, lat, lng)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if cover != nil {
		optPath, optSize, optErr := utils.OptimizeImage(cover.Path)
		if optErr == nil && optPath != "" {
			cover.Path = optPath
			cover.Size = optSize
			cover.MimeType = "image/jpeg"
		}
	}

	for i := range photos {
		optimizedPath, optimizedSize, optErr := utils.OptimizeImage(photos[i].Path)
		if optErr == nil && optimizedPath != "" {
			photos[i].Path = optimizedPath
			photos[i].Size = optimizedSize
			photos[i].MimeType = "image/jpeg"
		}
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		// 1. Crear el gimnasio
		if err := txRepo.Create(ctx, gym); err != nil {
			return err
		}

		// 2. Crear los 7 horarios semanales (0 a 6). Si algún día no se envió, se inicializa como cerrado.
		scheduleMap := make(map[int]GymScheduleItemRequest)
		for _, s := range schedules {
			scheduleMap[s.DayOfWeek] = s
		}

		var scheduleModels []models.GymSchedule
		for day := 0; day <= 6; day++ {
			if s, found := scheduleMap[day]; found {
				scheduleModels = append(scheduleModels, models.GymSchedule{
					GymID:        gym.ID,
					DayOfWeek:    day,
					OpenMinutes:  s.OpenMinutes,
					CloseMinutes: s.CloseMinutes,
					IsClosed:     s.IsClosed,
				})
			} else {
				scheduleModels = append(scheduleModels, models.GymSchedule{
					GymID:        gym.ID,
					DayOfWeek:    day,
					OpenMinutes:  0,
					CloseMinutes: 0,
					IsClosed:     true,
				})
			}
		}

		if err := txRepo.CreateSchedules(ctx, scheduleModels); err != nil {
			return err
		}
		gym.GymSchedules = scheduleModels

		// 3. Crear foto de portada si se proporcionó
		if cover != nil {
			coverFile := &models.File{
				AuthorID: authID,
				Slug:     cover.FileID,
				Path:     cover.Path,
				MimeType: cover.MimeType,
				Size:     cover.Size,
			}
			if err := txRepo.CreateFile(ctx, coverFile); err != nil {
				return err
			}

			if err := txRepo.Update(ctx, gym.ID, map[string]any{"background_id": coverFile.ID}); err != nil {
				return err
			}
			gym.BackgroundID = &coverFile.ID
			gym.BackgroundURL = utils.BuildEvidenceURL(cover.FileID)
		}

		// 4. Crear los archivos y las fotos de la galería
		for _, p := range photos {
			file := &models.File{
				AuthorID: authID,
				Slug:     p.FileID,
				Path:     p.Path,
				MimeType: p.MimeType,
				Size:     p.Size,
			}
			if err := txRepo.CreateFile(ctx, file); err != nil {
				return err
			}

			photo := &models.GymPhoto{
				GymID:  gym.ID,
				FileID: file.ID,
			}
			if err := txRepo.CreatePhoto(ctx, photo); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if cover != nil {
			_ = utils.DeleteFile(cover.Path)
		}
		for _, p := range photos {
			_ = utils.DeleteFile(p.Path)
		}
		return nil, err
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return gym, nil
}

func (uc *UseCase) Update(ctx context.Context, authID, gymID uint64, name, address, phone, email string, lat, lng *float64) error {
	if authID == 0 || gymID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return err
	}
	if !isStaff {
		return errorsx.Forbidden("No tienes permiso para modificar este gimnasio")
	}

	modelData := &models.Gym{
		Name:    name,
		Address: address,
		Phone:   phone,
		Email:   email,
	}

	if lat != nil {
		if *lat < -90 || *lat > 90 {
			return errorsx.BadRequest("Latitud inválida (-90 a 90)")
		}
		modelData.Latitude = *lat
	}

	if lng != nil {
		if *lng < -180 || *lng > 180 {
			return errorsx.BadRequest("Longitud inválida (-180 a 180)")
		}
		modelData.Longitude = *lng
	}

	updateMap := BuildGymUpdateData(modelData)
	if len(updateMap) == 0 {
		return nil
	}

	if err := uc.repo.Update(ctx, gymID, updateMap); err != nil {
		return err
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return nil
}

func enrichGymDetail(detail *GymDetailResponse) {
	detail.IsOpen = calculateIsOpen(detail.Schedules, time.Now())

	if detail.BackgroundSlug != "" {
		detail.BackgroundURL = utils.BuildEvidenceURL(detail.BackgroundSlug)
	}

	for i := range detail.Photos {
		if detail.Photos[i].Slug != "" {
			detail.Photos[i].URL = utils.BuildEvidenceURL(detail.Photos[i].Slug)
		}
	}
}

func (uc *UseCase) GetDetailsByID(ctx context.Context, id uint64) (*GymDetailResponse, error) {
	if id == 0 {
		return nil, errorsx.BadRequest("ID de gimnasio inválido")
	}

	detail, err := uc.repo.GetDetailsByID(ctx, id)
	if err != nil {
		return nil, err
	}

	enrichGymDetail(detail)

	return detail, nil
}


func (uc *UseCase) GetMyGymDetails(ctx context.Context, authID uint64) (*GymDetailResponse, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("ID de autenticación inválido")
	}

	detail, err := uc.repo.GetDetailsByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Gimnasio no encontrado para el usuario actual")
	}

	enrichGymDetail(detail)

	return detail, nil
}

func (uc *UseCase) GetAllByOwner(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedGymSummary, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("ID de autenticación inválido")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de dueño no encontrado")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	gyms, total, err := uc.repo.GetAllByOwnerID(ctx, ownerID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range gyms {
		schedules, err := uc.repo.GetSchedulesByGymID(ctx, gyms[i].ID)
		if err == nil {
			gyms[i].IsOpen = calculateIsOpen(schedules, now)
		}
		if gyms[i].BackgroundSlug != "" {
			gyms[i].BackgroundURL = utils.BuildEvidenceURL(gyms[i].BackgroundSlug)
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedGymSummary{
		Data: gyms,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) GetAllPublic(ctx context.Context, query string, lat, lng float64, page, pageSize int) (*PaginatedGymSummary, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	cacheKey := fmt.Sprintf("gyms:public:%s:%.4f:%.4f:%d:%d", query, lat, lng, page, pageSize)
	var cached PaginatedGymSummary
	if found, err := service.Get(ctx, uc.cache, cacheKey, &cached); err == nil && found {
		return &cached, nil
	}

	offset := (page - 1) * pageSize

	gyms, total, err := uc.repo.GetAllPublic(ctx, query, lat, lng, offset, pageSize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range gyms {
		schedules, err := uc.repo.GetSchedulesByGymID(ctx, gyms[i].ID)
		if err == nil {
			gyms[i].IsOpen = calculateIsOpen(schedules, now)
		}
		if gyms[i].BackgroundSlug != "" {
			gyms[i].BackgroundURL = utils.BuildEvidenceURL(gyms[i].BackgroundSlug)
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	result := &PaginatedGymSummary{
		Data: gyms,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	go func(res *PaginatedGymSummary, key string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = service.Set(bgCtx, uc.cache, key, res, 2*time.Minute)
	}(result, cacheKey)

	return result, nil
}

func (uc *UseCase) SetSchedules(ctx context.Context, authID, gymID uint64, schedules []GymScheduleItemRequest) error {
	if authID == 0 || gymID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return err
	}
	if !isStaff {
		return errorsx.Forbidden("No tienes permiso para configurar los horarios de este gimnasio")
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		for _, s := range schedules {
			scheduleModel := &models.GymSchedule{
				GymID:        gymID,
				DayOfWeek:    s.DayOfWeek,
				OpenMinutes:  s.OpenMinutes,
				CloseMinutes: s.CloseMinutes,
				IsClosed:     s.IsClosed,
			}
			if err := txRepo.UpsertSchedule(ctx, scheduleModel); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return nil
}

func (uc *UseCase) UpdateSchedule(ctx context.Context, authID, gymID, scheduleID uint64, req UpdateGymScheduleRequest) error {
	if authID == 0 || gymID == 0 || scheduleID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return err
	}
	if !isStaff {
		return errorsx.Forbidden("No tienes permiso para modificar los horarios de este gimnasio")
	}

	schedule, err := uc.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	if schedule.GymID != gymID {
		return errorsx.BadRequest("El horario no pertenece a este gimnasio")
	}

	updateData := BuildScheduleUpdateData(req)
	if len(updateData) == 0 {
		return nil
	}

	if err := uc.repo.UpdateSchedule(ctx, scheduleID, updateData); err != nil {
		return err
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return nil
}

func (uc *UseCase) UploadPhotos(ctx context.Context, authID, gymID uint64, photos []SavedFile) error {
	if authID == 0 || gymID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}
	if len(photos) == 0 {
		return errorsx.BadRequest("No se proporcionaron imágenes para subir")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return err
	}
	if !isStaff {
		return errorsx.Forbidden("No tienes permiso para agregar imágenes a este gimnasio")
	}

	gym, err := uc.repo.GetByID(ctx, gymID)
	if err != nil {
		return err
	}

	for i := range photos {
		optimizedPath, optimizedSize, optErr := utils.OptimizeImage(photos[i].Path)
		if optErr == nil && optimizedPath != "" {
			photos[i].Path = optimizedPath
			photos[i].Size = optimizedSize
			photos[i].MimeType = "image/jpeg"
		}
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		for _, p := range photos {
			file := &models.File{
				AuthorID: authID,
				Slug:     p.FileID,
				Path:     p.Path,
				MimeType: p.MimeType,
				Size:     p.Size,
			}
			if err := txRepo.CreateFile(ctx, file); err != nil {
				return err
			}

			photo := &models.GymPhoto{
				GymID:  gym.ID,
				FileID: file.ID,
			}
			if err := txRepo.CreatePhoto(ctx, photo); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		for _, p := range photos {
			_ = utils.DeleteFile(p.Path)
		}
		return err
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return nil
}

func (uc *UseCase) DeletePhoto(ctx context.Context, authID, gymID, photoID uint64) error {
	if authID == 0 || gymID == 0 || photoID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return err
	}
	if !isStaff {
		return errorsx.Forbidden("No tienes permiso para eliminar fotos de este gimnasio")
	}

	gym, err := uc.repo.GetByID(ctx, gymID)
	if err != nil {
		return err
	}

	photo, err := uc.repo.GetPhotoByID(ctx, photoID)
	if err != nil {
		return err
	}

	if photo.GymID != gymID {
		return errorsx.BadRequest("La foto no pertenece a este gimnasio")
	}

	file, err := uc.repo.GetFileByID(ctx, photo.FileID)
	if err != nil {
		return err
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		// 1. Si era el fondo del gimnasio, reasignar a la siguiente foto o nil
		if gym.BackgroundID != nil && *gym.BackgroundID == file.ID {
			nextPhoto, nextErr := txRepo.GetFirstPhotoByGymID(ctx, gymID)
			var newBackgroundID *uint64
			if nextErr == nil && nextPhoto != nil && nextPhoto.ID != photoID {
				newBackgroundID = &nextPhoto.FileID
			}
			if err := txRepo.Update(ctx, gym.ID, map[string]any{"background_id": newBackgroundID}); err != nil {
				return err
			}
		}

		// 2. Eliminar registro de GymPhoto
		if err := txRepo.DeletePhoto(ctx, photoID); err != nil {
			return err
		}

		// 3. Eliminar registro de File
		if err := txRepo.DeleteFile(ctx, file.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 4. Eliminar el archivo físico de disco
	_ = utils.DeleteFile(file.Path)

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return nil
}

func (uc *UseCase) UploadCover(ctx context.Context, authID, gymID uint64, slug, path, mimeType string, size int64) (string, error) {
	if authID == 0 || gymID == 0 {
		return "", errorsx.BadRequest("Parámetros inválidos")
	}

	isStaff, err := uc.repo.IsGymStaff(ctx, authID, gymID)
	if err != nil {
		return "", err
	}
	if !isStaff {
		return "", errorsx.Forbidden("No tienes permiso para actualizar este gimnasio")
	}

	gym, err := uc.repo.GetByID(ctx, gymID)
	if err != nil {
		return "", err
	}

	optimizedPath, optimizedSize, optErr := utils.OptimizeImage(path)
	if optErr == nil && optimizedPath != "" {
		path = optimizedPath
		size = optimizedSize
		mimeType = "image/jpeg"
	}

	oldBackgroundID := gym.BackgroundID
	var oldFilePath string
	if oldBackgroundID != nil {
		if oldFile, err := uc.repo.GetFileByID(ctx, *oldBackgroundID); err == nil && oldFile != nil {
			oldFilePath = oldFile.Path
		}
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		file := &models.File{
			AuthorID: authID,
			Slug:     slug,
			Path:     path,
			MimeType: mimeType,
			Size:     size,
		}
		if err := txRepo.CreateFile(ctx, file); err != nil {
			return err
		}

		if err := txRepo.Update(ctx, gym.ID, map[string]any{"background_id": file.ID}); err != nil {
			return err
		}

		if oldBackgroundID != nil {
			_ = txRepo.DeleteFile(ctx, *oldBackgroundID)
		}

		return nil
	})

	if err != nil {
		_ = utils.DeleteFile(path)
		return "", err
	}

	if oldFilePath != "" {
		_ = utils.DeleteFile(oldFilePath)
	}

	service.DeleteByPattern(ctx, uc.cache, "gyms:public:*")

	return utils.BuildEvidenceURL(slug), nil
}

