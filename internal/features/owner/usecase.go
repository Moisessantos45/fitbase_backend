package owner

import (
	"context"
	"encoding/json"
	"errors"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/utils"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CACHE_KEY_OWNER_PROFILE_PREFIX = "owner_profile_"
)

type UseCase struct {
	repo Repository
	rd   *redis.Client
}

func NewUseCase(repo Repository, rd *redis.Client) Service {
	return &UseCase{repo: repo, rd: rd}
}

func invalidateOwnerProfileCache(ctx context.Context, rd *redis.Client, authID uint64) {
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_OWNER_PROFILE_PREFIX, authID)
	_ = rd.Del(ctx, cacheKey)
}

func (uc *UseCase) CompleteProfile(ctx context.Context, authID uint64, firstName, lastName, phone string) (*Detail, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("ID de autenticación inválido")
	}

	existing, err := uc.repo.ExistsByAuthID(ctx, authID)
	if err != nil && !errors.Is(err, errorsx.ErrNotFound) {
		return nil, err
	}

	if existing {
		return nil, errorsx.Conflict("El perfil de dueño ya ha sido completado previamente")
	}

	newOwner, err := New(authID, firstName, lastName, phone)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	err = uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.Create(ctx, newOwner); err != nil {
			return err
		}

		if err := txRepo.MarkAuthProfileCompleted(ctx, authID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	invalidateOwnerProfileCache(ctx, uc.rd, authID)
	return uc.repo.GetByAuthID(ctx, authID)
}

func (uc *UseCase) GetProfile(ctx context.Context, authID uint64) (*Detail, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("ID de autenticación inválido")
	}

	var data *Detail
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_OWNER_PROFILE_PREFIX, authID)
	cachedData, err := uc.rd.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		if err = json.Unmarshal([]byte(cachedData), &data); err == nil {
			return data, nil
		}
	}

	user, err := uc.repo.GetByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errorsx.NotFound("Usuario no encontrado")
	}

	user.AvatarURL = utils.BuildEvidenceURL(user.Slug)

	go func(k string, u *Detail) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in cache goroutine:", r)
			}
		}()

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dataBytes, err := json.Marshal(u)
		if err == nil {
			uc.rd.Set(ctxTimeout, k, dataBytes, 10*time.Minute)
		}
	}(cacheKey, user)

	return user, nil
}

func (uc *UseCase) Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error {
	if authID == 0 {
		return errorsx.BadRequest("ID de autenticación inválido")
	}

	updateData := BuildUpdateData(firstName, lastName, phone)
	if len(updateData) == 0 {
		return nil
	}

	invalidateOwnerProfileCache(ctx, uc.rd, authID)
	return uc.repo.Update(ctx, authID, updateData)
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
