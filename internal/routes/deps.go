package routes

import (
	"fitbase/config"
	"fitbase/config/db"
	"fitbase/internal/features/analytics"
	"fitbase/internal/features/auth"
	"fitbase/internal/features/checkin"
	"fitbase/internal/features/daypass"
	"fitbase/internal/features/gym"
	"fitbase/internal/features/manager"
	"fitbase/internal/features/owner"
	"fitbase/internal/features/payment"
	"fitbase/internal/features/plan"
	"fitbase/internal/features/storage"
	"fitbase/internal/features/subscription"
	"fitbase/internal/features/user"
	"fitbase/internal/shared/middleware"
	"fitbase/internal/shared/utils"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	AppCotizaERP uint64 = 1
	AppTask      uint64 = 2
	AppAdmin     uint64 = 3
)

var (
	maker          *utils.PasetoMaker
	authUc         auth.Service
	userUc         user.Service
	ownerUc        owner.Service
	gymUc          gym.Service
	planUc         plan.Service
	managerUc      manager.Service
	dayPassUc      daypass.Service
	subscriptionUc subscription.Service
	checkInUc      checkin.Service
	paymentUc      payment.Service
	analyticsUc    analytics.Service
	storageUc      storage.Service
)

func Init() {
	rd := config.Rdb
	maker = utils.NewPasetoMaker()

	frontendURL := os.Getenv("HOST_URL_PROD")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	aRp := auth.NewPostgresRepository(db.DB)
	authUc = auth.NewUseCase(aRp, rd, maker, frontendURL)

	uRp := user.NewPostgresRepository(db.DB)
	userUc = user.NewUseCase(uRp, authUc, rd)

	oRp := owner.NewPostgresRepository(db.DB)
	ownerUc = owner.NewUseCase(oRp, rd)

	gRp := gym.NewPostgresRepository(db.DB)
	gymUc = gym.NewUseCase(gRp, rd)

	pRp := plan.NewPostgresRepository(db.DB)
	planUc = plan.NewUseCase(pRp)

	mRp := manager.NewPostgresRepository(db.DB)
	managerUc = manager.NewUseCase(mRp, rd, maker, frontendURL)

	dpRp := daypass.NewPostgresRepository(db.DB)
	dayPassUc = daypass.NewUseCase(dpRp)

	subRp := subscription.NewPostgresRepository(db.DB)
	subscriptionUc = subscription.NewUseCase(subRp)

	ciRp := checkin.NewPostgresRepository(db.DB)
	checkInUc = checkin.NewUseCase(ciRp)

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	payRp := payment.NewPostgresRepository(db.DB)
	paymentUc = payment.NewUseCase(payRp, stripeSecretKey, stripeWebhookSecret, frontendURL)

	anRp := analytics.NewPostgresRepository(db.DB)
	analyticsUc = analytics.NewUseCase(anRp)

	stRp := storage.NewPostgresRepository(db.DB)
	storageUc = storage.NewUseCase(stRp)
}

func authMiddleware() gin.HandlerFunc {
	return middleware.AuthMiddleware(maker, config.Rdb)
}

func preAuthMiddleware() gin.HandlerFunc {
	return middleware.PreAuthMiddleware(config.Rdb)
}

func idempotencyMiddleware() gin.HandlerFunc {
	return middleware.IdempotencyMiddleware(config.Rdb)
}
