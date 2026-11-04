package routes

import (
	"fitbase/config"
	"fitbase/internal/features/storage"
	"fitbase/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func StorageRoutes(rg *gin.RouterGroup) {
	h := storage.NewStorageHandler(storageUc)

	// Public routes (no auth middleware)
	public := rg.Group("/storage/public")
	{
		public.GET("/:slug", h.GetBySlugPublic)
	}

	protected := rg.Group("/storage")
	protected.Use(middleware.StorageAuthMiddleware(maker, config.Rdb))
	// protected.Use(appMiddleware(AppCotizaERP))
	// protected.Use(middleware.RateLimiterMiddleware(rate.Every(time.Minute/300), 300))
	{
		protected.GET("/:slug", h.GetBySlug)
		protected.DELETE("/:slug", h.DeleteBySlug)
	}
}
