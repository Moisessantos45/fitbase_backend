package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/templates"
	"fitbase/internal/shared/utils"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CACHE_KEY_MANAGER_PROFILE_PREFIX = "manager_profile_"
)

type UseCase struct {
	repo Repository
	rd   *redis.Client
	mk   *utils.PasetoMaker
	host string
}

func NewUseCase(repo Repository, rd *redis.Client, mk *utils.PasetoMaker, host string) Service {
	return &UseCase{
		repo: repo,
		rd:   rd,
		mk:   mk,
		host: host,
	}
}

func invalidateOwnerProfileCache(ctx context.Context, rd *redis.Client, authID uint64) {
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_MANAGER_PROFILE_PREFIX, authID)
	_ = rd.Del(ctx, cacheKey)
}

func (uc *UseCase) Create(ctx context.Context, authID uint64, req CreateRequest) error {
	if authID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	if !utils.EmailRegex.MatchString(req.Email) {
		return errorsx.BadRequest("Correo inválido")
	}

	gymID, err := uc.repo.GetGymIDByOwnerAuthID(ctx, authID)
	if err != nil {
		if errors.Is(err, errorsx.ErrNotFound) {
			return errorsx.Forbidden("No tienes un gimnasio asignado o no eres dueño")
		}
		return err
	}

	exists, err := uc.repo.ExistsAuthByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if exists {
		return errorsx.Conflict("El correo ya se encuentra registrado")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return errorsx.Internal("Error procesando contraseña", err)
	}

	authRecord := &models.Auth{
		RoleID:         models.RoleGymManagerID,
		Email:          req.Email,
		PasswordHash:   hashedPassword,
		FullProfile:    true,
		EmailConfirmed: false,
		Active:         true,
	}

	managerModel, err := NewManager(gymID, authID, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		return err
	}

	if err := uc.repo.CreateManagerWithAuth(ctx, authRecord, managerModel); err != nil {
		return err
	}

	token, err := uc.mk.NewToken(fmt.Sprintf("%d", authRecord.ID), 24*time.Hour)
	if err == nil {
		confirmKey := fmt.Sprintf("%s:%d", templates.CONFIRM_ACCOUNT_PATH, authRecord.ID)
		_ = uc.rd.Set(ctx, confirmKey, token, 24*time.Hour).Err()

		renderer, err := templates.NewEmailRenderer()
		if err == nil {
			data := templates.ConfirmAccountData{
				Name:        managerModel.FirstName,
				ConfirmLink: fmt.Sprintf("%s/%s/%s", uc.host, templates.CONFIRM_ACCOUNT_PATH, token),
			}
			htmlContent, err := renderer.RenderConfirmAccount(data)
			if err == nil {
				_ = utils.EnqueueEmail([]string{authRecord.Email}, "Confirmación de cuenta como Encargado", htmlContent)
			}
		}
	}

	return nil
}

func (uc *UseCase) GetAllManagersByGym(ctx context.Context, authID uint64, page int, pageSize int, query string) (*PaginatedItems, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	results, total, err := uc.repo.GetAllByGymID(ctx, authID, offset, pageSize, query)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errors.New("page out of range")
	}

	for i := range results {
		if results[i].Slug == "" {
			continue
		}

		results[i].AvatarURL = utils.BuildEvidenceURL(results[i].Slug)
	}

	return &PaginatedItems{
		Data: results,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) OpenCashRegister(ctx context.Context, authID, gymID uint64, openCash float64) (*CashRegisterStatusResponse, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	active, err := uc.repo.GetActiveCashRegister(ctx, gymID, authID)
	if err != nil {
		return nil, err
	}

	if active != nil {
		return nil, errorsx.Conflict("Ya existe una caja abierta para este turno en este gimnasio")
	}

	register, err := NewCashRegister(gymID, authID, openCash)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.OpenCashRegister(ctx, register); err != nil {
		return nil, err
	}

	return &CashRegisterStatusResponse{
		ID:        register.ID,
		GymID:     register.GymID,
		AuthID:    register.AuthID,
		OpenTime:  register.OpenTime,
		CloseTime: register.CloseTime,
		OpenCash:  register.OpenCash,
		CloseCash: register.CloseCash,
		IsOpen:    true,
	}, nil
}

func (uc *UseCase) CloseCashRegister(ctx context.Context, authID uint64, closeCash float64) (*CashRegisterStatusResponse, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	active, err := uc.repo.GetActiveCashRegisterByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, errorsx.NotFound("No hay ninguna caja abierta activa para este usuario")
	}

	closeTime := time.Now()
	if err := uc.repo.CloseCashRegister(ctx, active.ID, closeCash, closeTime); err != nil {
		return nil, err
	}

	return &CashRegisterStatusResponse{
		ID:        active.ID,
		GymID:     active.GymID,
		AuthID:    active.AuthID,
		OpenTime:  active.OpenTime,
		CloseTime: &closeTime,
		OpenCash:  active.OpenCash,
		CloseCash: closeCash,
		IsOpen:    false,
	}, nil
}

func (uc *UseCase) GetCurrentCashRegister(ctx context.Context, authID uint64) (*CashRegisterStatusResponse, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	active, err := uc.repo.GetActiveCashRegisterByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, errorsx.NotFound("No hay ninguna caja abierta activa")
	}

	return &CashRegisterStatusResponse{
		ID:        active.ID,
		GymID:     active.GymID,
		AuthID:    active.AuthID,
		OpenTime:  active.OpenTime,
		CloseTime: active.CloseTime,
		OpenCash:  active.OpenCash,
		CloseCash: active.CloseCash,
		IsOpen:    true,
	}, nil
}

func (uc *UseCase) GetDetailByAuthID(ctx context.Context, authID uint64) (*Detail, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	var detail *Detail
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_MANAGER_PROFILE_PREFIX, authID)
	cachedData, err := uc.rd.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		if err = json.Unmarshal([]byte(cachedData), &detail); err == nil {
			return detail, nil
		}
	}

	detail, err = uc.repo.GetDetailByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}

	if detail == nil {
		return nil, errorsx.NotFound("No se encontró el detalle del encargado")
	}

	if detail.Slug != "" {
		detail.AvatarURL = utils.BuildEvidenceURL(detail.Slug)

	}

	go func(data *Detail, key string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in cache goroutine:", r)
			}
		}()

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		jsonData, err := json.Marshal(data)
		if err == nil {
			uc.rd.Set(ctxTimeout, key, jsonData, 10*time.Minute)
		}

	}(detail, cacheKey)

	return detail, nil
}

func (uc *UseCase) Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error {
	if authID == 0 {
		return nil
	}

	updateData := BuildUpdateData(firstName, lastName, phone)

	err := uc.repo.Update(ctx, authID, updateData)
	if err != nil {
		return err
	}

	invalidateOwnerProfileCache(ctx, uc.rd, authID)
	return nil
}

func (uc *UseCase) UploadAvatar(ctx context.Context, authID uint64, slug, path, mimeType string, size int64) error {
	if authID == 0 {
		return errorsx.BadRequest("ID de autenticación inválido")
	}

	optimizedPath, optimizedSize, optErr := utils.OptimizeImage(path)
	if optErr == nil && optimizedPath != "" {
		path = optimizedPath
		size = optimizedSize
		mimeType = "image/jpeg"
	}

	err := uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
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

		return txRepo.UpdateAvatarByAuthID(ctx, authID, file.ID)
	})

	if err != nil {
		return err
	}

	invalidateOwnerProfileCache(ctx, uc.rd, authID)
	return nil
}
