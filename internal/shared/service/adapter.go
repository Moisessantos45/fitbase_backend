package service

import "context"

type CacheRepoAdapter[T any] struct {
	repo interface {
		GetAll(ctx context.Context, offset int, limit int) ([]T, int64, error)
		GetAllPublic(ctx context.Context, offset int, limit int) ([]T, int64, error)
		GetAllRecents(ctx context.Context, max int) ([]T, error)
		GetByID(ctx context.Context, id uint64) (*T, error)
		ExistsById(ctx context.Context, id uint64) (bool, error)
		Create(ctx context.Context, data *T) error
		Update(ctx context.Context, id uint64, data *T) error
	}
}

func NewCacheRepoAdapter[T any](repo interface {
	GetAll(ctx context.Context, offset int, limit int) ([]T, int64, error)
	GetAllPublic(ctx context.Context, offset int, limit int) ([]T, int64, error)
	GetAllRecents(ctx context.Context, max int) ([]T, error)
	GetByID(ctx context.Context, id uint64) (*T, error)
	ExistsById(ctx context.Context, id uint64) (bool, error)
	Create(ctx context.Context, data *T) error
	Update(ctx context.Context, id uint64, data *T) error
}) *CacheRepoAdapter[T] {
	return &CacheRepoAdapter[T]{repo: repo}
}

func (a *CacheRepoAdapter[T]) GetAll(ctx context.Context, offset int, limit int) ([]T, int64, error) {
	return a.repo.GetAll(ctx, offset, limit)
}

func (a *CacheRepoAdapter[T]) GetAllPublic(ctx context.Context, offset int, limit int) ([]T, int64, error) {
	return a.repo.GetAllPublic(ctx, offset, limit)
}

func (a *CacheRepoAdapter[T]) GetAllRecents(ctx context.Context, max int) ([]T, error) {
	return a.repo.GetAllRecents(ctx, max)
}

func (a *CacheRepoAdapter[T]) GetByID(ctx context.Context, id uint64) (*T, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *CacheRepoAdapter[T]) ExistsById(ctx context.Context, id uint64) (bool, error) {
	return a.repo.ExistsById(ctx, id)
}

func (a *CacheRepoAdapter[T]) Create(ctx context.Context, data *T) error {
	return a.repo.Create(ctx, data)
}

func (a *CacheRepoAdapter[T]) Update(ctx context.Context, id uint64, data *T) error {
	return a.repo.Update(ctx, id, data)
}
