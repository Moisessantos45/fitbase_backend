package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

func Get[T any](ctx context.Context, cache *CacheService, key string, result *T) (bool, error) {
	cachedData, err := cache.rd.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := json.Unmarshal(cachedData, result); err != nil {
		return false, err
	}
	return true, nil
}

func Set[T any](ctx context.Context, cache *CacheService, key string, data T, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return cache.rd.Set(ctx, key, jsonData, ttl).Err()
}

func BuildCacheKey(prefix string, parts ...any) string {
	key := prefix
	for _, part := range parts {
		key = fmt.Sprintf("%s:%v", key, part)
	}
	return key
}

func DeleteByPattern(ctx context.Context, cache *CacheService, pattern string) {
	var cursor uint64
	for {
		keys, nextCursor, err := cache.rd.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			cache.rd.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func Paginate[T any](
	ctx context.Context,
	cache *CacheService,
	cacheKeyPrefix string,
	page int,
	pageSize int,
	fetch func(context.Context, int, int) ([]T, int64, error),
	newResult func([]T, int64, int, int, int) *PaginatedResult[T],
) (*PaginatedResult[T], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	cacheKey := BuildCacheKey(cacheKeyPrefix, page, pageSize)

	var cachedResult PaginatedResult[T]
	if found, err := Get(ctx, cache, cacheKey, &cachedResult); err == nil && found {
		return &cachedResult, nil
	}

	results, total, err := fetch(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return newResult([]T{}, 0, 0, 0, 0), nil
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if page > totalPages && totalPages > 0 {
		return nil, fmt.Errorf("la página solicitada excede el número total de páginas disponibles")
	}

	result := newResult(results, total, totalPages, page, pageSize)

	go func(res *PaginatedResult[T], key string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = Set(bgCtx, cache, key, res, 1*time.Minute)
	}(result, cacheKey)

	return result, nil
}

func GetRecent[T any](
	ctx context.Context,
	cache *CacheService,
	cacheKeyPrefix string,
	max int,
	fetch func(context.Context, int) ([]T, error),
	newResult func([]T, int64, int, int, int) *PaginatedResult[T],
) (*PaginatedResult[T], error) {
	cacheKey := BuildCacheKey(cacheKeyPrefix, "recent", max)

	var cachedResult PaginatedResult[T]
	if found, err := Get(ctx, cache, cacheKey, &cachedResult); err == nil && found {
		return &cachedResult, nil
	}

	results, err := fetch(ctx, max)
	if err != nil {
		return nil, err
	}

	result := newResult(results, 0, 0, 0, 0)

	go func(res *PaginatedResult[T], key string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = Set(bgCtx, cache, key, res, 1*time.Minute)
	}(result, cacheKey)

	return result, nil
}
