package main

import (
	"context"
	"fitbase/config"
	"fitbase/config/db"
	"fitbase/internal/routes"
	"fitbase/internal/shared/helper"
	"fitbase/internal/shared/middleware"
	"fitbase/internal/shared/utils"

	"strings"

	"fmt"
	"log"
	"net/http"

	// _ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielkov/gin-helmet/ginhelmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Advertencia: no se encontró .env, se usan variables de entorno del sistema")
	}

	if err := db.Connect(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := db.InitializeDatabase(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := config.InitRedis(context.Background()); err != nil {
		fmt.Println("Error initializing Redis:", err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered in cache goroutine: %v", r)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if sqlDB, err := db.GetSQLDB(); err == nil {
			if err := sqlDB.PingContext(ctx); err != nil {
				log.Printf("Advertencia en warm-up de DB: %v", err)
			}
		}
	}()

	// pprof debug server — solo en localhost para no exponer datos internos
	// Descomentar para profiling en desarrollo:
	// go func() {
	// 	log.Println("pprof disponible en http://localhost:6060/debug/pprof/")
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	ip := utils.GetOutboundIP()
	log.Println("Environment variables loaded successfully", ip)

	HOST_URL_DEV := os.Getenv("HOST_URL_DEV")
	HOST_URL_PROD := os.Getenv("HOST_URL_PROD")
	HOST_URL_PROD_WWW := os.Getenv("HOST_URL_PROD_WWW")
	HOST_API_PROD := os.Getenv("HOST_API_PROD")
	HOST_API_PROD_WWW := os.Getenv("HOST_API_PROD_WWW")
	HOST_PYTHON_API_URL := os.Getenv("HOST_PYTHON_API_URL")
	HOST_URL_DEV_V2 := os.Getenv("HOST_URL_DEV_V2")

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(gin.Logger(), middleware.SafeRecovery())
	r.HandleMethodNotAllowed = true
	r.NoRoute(middleware.NotFound)
	r.NoMethod(middleware.MethodNotAllowed)
	if strings.TrimSpace(os.Getenv("GO_ENV")) == "prod" {
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression,
		gzip.WithExcludedPaths([]string{"/health", "/api/test"}),
		gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".webp", ".zip", ".gz"}),
	))
	r.Use(ginhelmet.Default())
	r.Use(ginhelmet.Referrer())

	// Orígenes permitidos: sin vacíos (variables sin definir) y sin el origen de desarrollo en producción.
	isProd := strings.TrimSpace(os.Getenv("GO_ENV")) == "prod"
	candidates := []string{HOST_URL_PROD, HOST_URL_PROD_WWW, HOST_API_PROD, HOST_API_PROD_WWW, HOST_PYTHON_API_URL}
	if !isProd {
		candidates = append(candidates, HOST_URL_DEV, HOST_URL_DEV_V2, "http://192.168.100.54:5173")
	}

	allowOrigins := make([]string, 0, len(candidates))
	for _, o := range candidates {
		o = strings.TrimSpace(o)
		if o != "" && o != "*" {
			allowOrigins = append(allowOrigins, o)
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	middleware.StartCleanup()
	// 5 req/seg (300/min) con burst 50, compartido entre instancias vía Redis
	r.Use(middleware.RateLimiterRedis(config.Rdb, "global", rate.Limit(5), 50))

	// Solo application/json y multipart/form-data, con límites de tamaño y validación de JSON
	r.MaxMultipartMemory = 8 << 20
	r.Use(middleware.RequestGuard(middleware.DefaultGuardConfig))

	// Los errores de validación usan los nombres del tag json (p. ej. first_name)
	helper.RegisterJSONTagName()

	routes.Init()
	api := r.Group("/api/v1")
	{
		routes.OwnerRoutes(api)
		routes.UserRoutes(api)
		routes.GymRoutes(api)
		routes.PlanRoutes(api)
		routes.ManagerRoutes(api)
		routes.DayPassRoutes(api)
		routes.SubscriptionRoutes(api)
		routes.CheckInRoutes(api)
		routes.PaymentRoutes(api)
		routes.AnalyticsRoutes(api)
		routes.StorageRoutes(api)
	}

	auth := api.Group("/auth")
	{
		routes.AuthRoutes(auth)
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to the CV API",
		})
	})

	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "OK"})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := ":4109"
	if strings.TrimSpace(os.Getenv("GO_ENV")) == "prod" {
		addr = "127.0.0.1:4109"
	}

	log.Println("Server starting on :4109...")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	if config.Rdb != nil {
		if err := config.Rdb.Close(); err != nil {
			log.Println("Error closing Redis:", err)
		}
	}

	db.CloseDB()

	log.Println("Server exiting")
}
