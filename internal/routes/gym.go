package routes

import (
	"fitbase/internal/features/gym"

	"github.com/gin-gonic/gin"
)

func GymRoutes(rg *gin.RouterGroup) {
	h := gym.NewHandler(gymUc)

	public := rg.Group("/gyms")
	public.Use(idempotencyMiddleware())
	{
		public.GET("", h.GetAllPublic)
		public.GET("/:id", h.GetDetailsByID)
	}

	protected := rg.Group("/gyms")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.POST("", h.Create)
		protected.GET("/my-gym", h.GetMyGymDetails)
		protected.GET("/my-gyms", h.GetAllByOwner)
		protected.PUT("/:id", h.Update)
		protected.POST("/:id/schedules", h.SetSchedules)
		protected.PUT("/:id/schedules/:schedule_id", h.UpdateSchedule)
		protected.POST("/:id/photos", h.UploadPhotos)
		protected.DELETE("/:id/photos/:photo_id", h.DeletePhoto)
		protected.POST("/:id/cover", h.UploadCover)
		protected.PUT("/:id/cover", h.UploadCover)
	}
}
