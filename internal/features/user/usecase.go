package user

import (
	"context"
	"encoding/json"
	"fitbase/internal/features/auth"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/utils"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CACHE_KEY_USER_PROFILE_PREFIX = "user_profile_"
)

type UseCase struct {
	repo   Repository
	ucAuth auth.Service
	rd     *redis.Client
}

func NewUseCase(repo Repository, ucAuth auth.Service, rd *redis.Client) Service {
	return &UseCase{repo: repo, ucAuth: ucAuth, rd: rd}
}

func invalidateProfileCache(ctx context.Context, rd *redis.Client, authID uint64) {
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_USER_PROFILE_PREFIX, authID)
	_ = rd.Del(ctx, cacheKey)
}

func (uc *UseCase) GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	if authID == 0 {
		return 0, nil
	}

	return uc.repo.GetIDByAuthID(ctx, authID)
}

func (uc *UseCase) GetByAuthID(ctx context.Context, authID uint64) (*UserDetail, error) {
	if authID == 0 {
		return nil, nil
	}

	var user *UserDetail
	cacheKey := fmt.Sprintf("%s%d", CACHE_KEY_USER_PROFILE_PREFIX, authID)
	cachedUser, err := uc.rd.Get(ctx, cacheKey).Result()
	if err == nil && cachedUser != "" {
		if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
			return user, nil
		}
	}

	user, err = uc.repo.GetByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, nil
	}

	user.AvatarURL = utils.BuildEvidenceURL(user.Slug)

	go func(key string, u *UserDetail) {

		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered in cache goroutine: %v", r)
			}
		}()

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		userJSON, err := json.Marshal(u)
		if err == nil {
			uc.rd.Set(ctxTimeout, key, userJSON, 10*60*1e9) // Cache for 10 minutes
		}

	}(cacheKey, user)

	return user, nil
}

func (uc *UseCase) GetEmailByID(ctx context.Context, id uint64) (string, error) {
	if id == 0 {
		return "", nil
	}

	return uc.repo.GetEmailByID(ctx, id)
}

func (uc *UseCase) CompleteProfile(ctx context.Context, authID uint64, firstName, lastName string, phone string) (*models.User, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	newUser, err := New(authID, firstName, lastName, phone)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}

	log.Printf("Creating user with AuthID: %d", newUser.AuthID)

	err = uc.repo.WithTransaction(func(repo *PostgresRepository) error {
		if err := uc.repo.Create(ctx, newUser); err != nil {
			return err
		}

		if err := repo.ChangeCompletedProfile(newUser.AuthID, true); err != nil {
			return errorsx.Internal("Error al actualizar el perfil", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error completing profile: %v", err)
		return nil, err
	}

	invalidateProfileCache(ctx, uc.rd, authID)

	return newUser, nil
}

func (uc *UseCase) Update(ctx context.Context, authID uint64, firstName *string, lastName *string, phone *string) error {
	if authID == 0 {
		return nil
	}

	updateData := BuildUserUpdateData(firstName, lastName, phone)

	err := uc.repo.Update(ctx, authID, updateData)
	if err != nil {
		return err
	}

	invalidateProfileCache(ctx, uc.rd, authID)

	return nil
}

func (uc *UseCase) UploadAvatar(ctx context.Context, authID uint64, slug string, path, mimeType string, size int64) error {
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

		return txRepo.UpdateAvatarURL(ctx, authID, file.ID)
	})

	if err != nil {
		return err
	}

	invalidateProfileCache(ctx, uc.rd, authID)
	return nil
}
