package payment

import (
	"context"
	"encoding/json"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/templates"
	"fitbase/internal/shared/utils"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/checkout/session"
	"github.com/stripe/stripe-go/v80/webhook"
)

type UseCase struct {
	repo                Repository
	stripeSecretKey     string
	stripeWebhookSecret string
	frontendURL         string
}

func NewUseCase(repo Repository, stripeSecretKey, stripeWebhookSecret, frontendURL string) Service {
	return &UseCase{
		repo:                repo,
		stripeSecretKey:     stripeSecretKey,
		stripeWebhookSecret: stripeWebhookSecret,
		frontendURL:         frontendURL,
	}
}

func validateGymSchedule(ctx context.Context, repo Repository, gymID uint64) error {
	schedules, err := repo.GetSchedulesByGymID(ctx, gymID)
	if err != nil {
		return err
	}

	now := time.Now()
	dayOfWeek := int(now.Weekday())
	var todaySchedule *models.GymSchedule
	for i := range schedules {
		if schedules[i].DayOfWeek == dayOfWeek {
			todaySchedule = &schedules[i]
			break
		}
	}

	if todaySchedule == nil || todaySchedule.IsClosed {
		return errorsx.BadRequest("El gimnasio se encuentra cerrado hoy. No es posible registrar cobros en mostrador")
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	if currentMinutes < todaySchedule.OpenMinutes || currentMinutes >= todaySchedule.CloseMinutes {
		return errorsx.BadRequest("Fuera del horario de atención del gimnasio. No es posible registrar cobros en mostrador")
	}

	return nil
}

func (uc *UseCase) RegisterCashPayment(ctx context.Context, authID uint64, req RegisterPaymentRequest) (*PaymentReceiptResponse, error) {
	if authID == 0 || req.GymID == 0 || req.Amount <= 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	isAuthorized := false
	managerGymID, err := uc.repo.GetManagerGymID(ctx, authID)
	if err == nil && managerGymID == req.GymID {
		isAuthorized = true
	} else {
		ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
		if err == nil {
			gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, req.GymID)
			if err == nil && ownerID == gymOwnerID {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		return nil, errorsx.Forbidden("No tienes permiso para registrar pagos en esta sede")
	}

	if err := validateGymSchedule(ctx, uc.repo, req.GymID); err != nil {
		return nil, err
	}

	method, err := uc.repo.GetPaymentMethodByID(ctx, req.PaymentMethodID)
	if err != nil || !method.Active {
		return nil, errorsx.BadRequest("Método de pago no válido o inactivo")
	}

	var cashRegisterID *uint64
	if req.PaymentMethodID == models.PaymentMethodCashID {
		activeBox, err := uc.repo.GetActiveCashRegisterByAuthID(ctx, authID, req.GymID)
		if err != nil || activeBox == nil {
			return nil, errorsx.BadRequest("Debes abrir un turno de caja física antes de registrar cobros en efectivo")
		}
		cashRegisterID = &activeBox.ID
	}

	var dayPassDetails *DayPassPaymentDetails
	if req.DayPassID != nil && *req.DayPassID > 0 {
		dp, err := uc.repo.GetDayPassDetailsByID(ctx, *req.DayPassID)
		if err != nil || dp == nil {
			return nil, errorsx.NotFound("Pase de un solo uso no encontrado")
		}

		if dp.GymID != req.GymID {
			return nil, errorsx.BadRequest("El pase de un solo uso no corresponde a esta sede")
		}

		if dp.IsUsed {
			return nil, errorsx.BadRequest("El pase ya ha sido utilizado previamente")
		}

		paid, err := uc.repo.IsDayPassPaid(ctx, *req.DayPassID)
		if err == nil && paid {
			return nil, errorsx.BadRequest("El pase ya cuenta con un pago completado registrado")
		}

		nowDate := time.Now().Format("2006-01-02")
		if dp.VisitDate != "" && dp.VisitDate < nowDate {
			return nil, errorsx.BadRequest(fmt.Sprintf("El pase de un solo uso está vencido (fecha programada: %s). No se puede cobrar un pase expirado", dp.VisitDate))
		}

		dayPassDetails = dp
	}

	payment, err := NewPayment(
		req.GymID,
		req.PaymentMethodID,
		models.PaymentStatusPaidID,
		req.Amount,
		req.SubscriptionID,
		req.DayPassID,
		cashRegisterID,
	)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	receipt, err := uc.repo.GetReceiptByID(ctx, payment.ID)
	if err != nil {
		return nil, err
	}

	if dayPassDetails != nil && strings.TrimSpace(dayPassDetails.Email) != "" {
		dpDetails := dayPassDetails
		go func() {
			renderer, err := templates.NewEmailRenderer()
			if err != nil {
				return
			}
			name := strings.TrimSpace(dpDetails.FullName)
			if name == "" {
				name = "Visitante"
			}
			data := templates.DayPassPaymentConfirmedData{
				Name:              name,
				GymName:           dpDetails.GymName,
				PlanName:          dpDetails.PlanName,
				Amount:            receipt.Amount,
				PaymentMethodName: receipt.PaymentMethodName,
				TicketCode:        dpDetails.Slug,
				VisitDate:         dpDetails.VisitDate,
			}
			html, err := renderer.RenderDayPassPaymentConfirmed(data)
			if err == nil {
				_ = utils.EnqueueEmail([]string{dpDetails.Email}, fmt.Sprintf("Pago Confirmado: Tu Pase de un Día en %s", dpDetails.GymName), html)
			}
		}()
	}

	return receipt, nil
}

func (uc *UseCase) GetRevenueSummary(ctx context.Context, authID, gymID uint64) (*GymRevenueSummaryResponse, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.Forbidden("Solo el propietario de la sede puede consultar los reportes de ingresos")
	}

	gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
	if err != nil || ownerID != gymOwnerID {
		return nil, errorsx.Forbidden("Solo el propietario de la sede puede consultar los reportes de ingresos")
	}

	totalRev, totalTx, err := uc.repo.GetTotalRevenueByGymID(ctx, gymID)
	if err != nil {
		return nil, err
	}

	gymName, err := uc.repo.GetGymName(ctx, gymID)
	if err != nil {
		gymName = "Gimnasio"
	}

	return &GymRevenueSummaryResponse{
		GymID:             gymID,
		GymName:           gymName,
		TotalRevenue:      totalRev,
		TotalTransactions: totalTx,
	}, nil
}

func (uc *UseCase) GetAllByGym(ctx context.Context, authID, gymID uint64, page, pageSize int) (*PaginatedPaymentSummary, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.Forbidden("Solo el propietario de la sede puede consultar el listado de transacciones")
	}

	gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
	if err != nil || ownerID != gymOwnerID {
		return nil, errorsx.Forbidden("Solo el propietario de la sede puede consultar el listado de transacciones")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	items, total, err := uc.repo.GetAllByGymID(ctx, gymID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedPaymentSummary{
		Data: items,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) GetMyPayments(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedPaymentSummary, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	userID, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de usuario no encontrado")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	items, total, err := uc.repo.GetAllByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedPaymentSummary{
		Data: items,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) CreateCheckoutSession(ctx context.Context, authID uint64, req CreateCheckoutSessionRequest) (*CheckoutSessionResponse, error) {
	if authID == 0 || req.GymID == 0 || req.PlanID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	profile, err := uc.repo.GetUserProfileByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de usuario no encontrado")
	}

	gym, err := uc.repo.GetGymByID(ctx, req.GymID)
	if err != nil {
		return nil, errorsx.NotFound("Gimnasio no encontrado")
	}

	plan, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || !plan.Active || plan.GymID != req.GymID {
		return nil, errorsx.BadRequest("Plan no disponible o inactivo")
	}

	isDayPassPlan := strings.ToLower(plan.PlanType) == "visita" || strings.ToLower(plan.PlanType) == "daypass" || strings.ToLower(plan.PlanType) == "dia" || plan.Duration <= 1
	if req.Type == "daypass" && !isDayPassPlan {
		return nil, errorsx.BadRequest("El plan seleccionado no es de tipo pase de día")
	}
	if req.Type == "subscription" && isDayPassPlan {
		return nil, errorsx.BadRequest("El plan seleccionado es de tipo pase de día, debe usarse daypass")
	}

	payment, err := NewPayment(
		req.GymID,
		models.PaymentMethodCreditCardID,
		models.PaymentStatusPendingID,
		plan.Price,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	stripeKey := uc.stripeSecretKey
	if stripeKey == "" {
		stripeKey = "sk_test_mock_fitbase_development_key"
	}
	stripe.Key = stripeKey

	successURL := fmt.Sprintf("%s/payment/success?session_id={CHECKOUT_SESSION_ID}&payment_id=%d", uc.frontendURL, payment.ID)
	cancelURL := fmt.Sprintf("%s/payment/cancel?payment_id=%d", uc.frontendURL, payment.ID)

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("mxn"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name),
						Description: stripe.String(fmt.Sprintf("%s en %s", plan.Name, gym.Name)),
					},
					UnitAmount: stripe.Int64(int64(math.Round(plan.Price * 100))),
				},
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(profile.Email),
		Metadata: map[string]string{
			"payment_id": fmt.Sprintf("%d", payment.ID),
			"type":       req.Type,
			"gym_id":     fmt.Sprintf("%d", req.GymID),
			"user_id":    fmt.Sprintf("%d", profile.ID),
			"auth_id":    fmt.Sprintf("%d", authID),
			"full_name":  profile.FullName,
			"phone":      profile.Phone,
			"email":      profile.Email,
			"plan_id":    fmt.Sprintf("%d", req.PlanID),
			"visit_date": req.VisitDate,
			"auto_renew": fmt.Sprintf("%t", req.AutoRenew),
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, errorsx.Internal("Error al iniciar sesión en Stripe", err)
	}

	return &CheckoutSessionResponse{
		SessionID:   sess.ID,
		CheckoutURL: sess.URL,
		PaymentID:   payment.ID,
		Amount:      plan.Price,
	}, nil
}

func (uc *UseCase) CreateGuestCheckoutSession(ctx context.Context, req CreateGuestCheckoutSessionRequest) (*CheckoutSessionResponse, error) {
	if req.GymID == 0 || req.PlanID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	gym, err := uc.repo.GetGymByID(ctx, req.GymID)
	if err != nil {
		return nil, errorsx.NotFound("Gimnasio no encontrado")
	}

	plan, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || !plan.Active || plan.GymID != req.GymID {
		return nil, errorsx.BadRequest("Plan no disponible o inactivo")
	}

	isDayPassPlan := strings.ToLower(plan.PlanType) == "visita" || strings.ToLower(plan.PlanType) == "daypass" || strings.ToLower(plan.PlanType) == "dia" || plan.Duration <= 1
	if !isDayPassPlan {
		return nil, errorsx.BadRequest("El plan seleccionado no es de tipo pase de día")
	}

	payment, err := NewPayment(
		req.GymID,
		models.PaymentMethodCreditCardID,
		models.PaymentStatusPendingID,
		plan.Price,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	stripeKey := uc.stripeSecretKey
	if stripeKey == "" {
		stripeKey = "sk_test_mock_fitbase_development_key"
	}
	stripe.Key = stripeKey

	successURL := fmt.Sprintf("%s/payment/success?session_id={CHECKOUT_SESSION_ID}&payment_id=%d", uc.frontendURL, payment.ID)
	cancelURL := fmt.Sprintf("%s/payment/cancel?payment_id=%d", uc.frontendURL, payment.ID)

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("mxn"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name),
						Description: stripe.String(fmt.Sprintf("%s en %s (Pase de Invitado)", plan.Name, gym.Name)),
					},
					UnitAmount: stripe.Int64(int64(math.Round(plan.Price * 100))),
				},
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(req.Email),
		Metadata: map[string]string{
			"payment_id": fmt.Sprintf("%d", payment.ID),
			"type":       "daypass",
			"gym_id":     fmt.Sprintf("%d", req.GymID),
			"user_id":    "0",
			"auth_id":    "0",
			"full_name":  req.FullName,
			"phone":      req.Phone,
			"email":      req.Email,
			"plan_id":    fmt.Sprintf("%d", req.PlanID),
			"visit_date": req.VisitDate,
			"auto_renew": "false",
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, errorsx.Internal("Error al iniciar sesión en Stripe", err)
	}

	return &CheckoutSessionResponse{
		SessionID:   sess.ID,
		CheckoutURL: sess.URL,
		PaymentID:   payment.ID,
		Amount:      plan.Price,
	}, nil
}

func (uc *UseCase) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	var event stripe.Event
	var err error

	if uc.stripeWebhookSecret != "" {
		event, err = webhook.ConstructEvent(payload, signature, uc.stripeWebhookSecret)
		if err != nil {
			return errorsx.ErrUnauthorized
		}
	} else {
		if err := json.Unmarshal(payload, &event); err != nil {
			return errorsx.ErrInvalidData
		}
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return errorsx.ErrInvalidData
		}
		return uc.processCheckoutCompleted(ctx, &sess)
	case "checkout.session.expired":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return errorsx.ErrInvalidData
		}
		return uc.processCheckoutExpired(ctx, &sess)
	}

	return nil
}

func (uc *UseCase) processCheckoutCompleted(ctx context.Context, sess *stripe.CheckoutSession) error {
	paymentIDStr := sess.Metadata["payment_id"]
	if paymentIDStr == "" {
		return nil
	}

	var paymentID uint64
	fmt.Sscanf(paymentIDStr, "%d", &paymentID)
	if paymentID == 0 {
		return nil
	}

	paymentType := sess.Metadata["type"]
	var gymID, userID, planID uint64
	fmt.Sscanf(sess.Metadata["gym_id"], "%d", &gymID)
	fmt.Sscanf(sess.Metadata["user_id"], "%d", &userID)
	fmt.Sscanf(sess.Metadata["plan_id"], "%d", &planID)

	autoRenew := sess.Metadata["auto_renew"] == "true"
	visitDateStr := sess.Metadata["visit_date"]

	var createdDayPassID *uint64

	err := uc.repo.WithTransaction(func(txRepo Repository) error {
		payment, err := txRepo.GetPaymentByID(ctx, paymentID)
		if err != nil {
			return err
		}

		if payment.PaymentStatusID == models.PaymentStatusPaidID {
			return nil
		}

		plan, err := txRepo.GetPlanByID(ctx, planID)
		if err != nil {
			return err
		}

		var subID, dayPassID *uint64

		switch paymentType {
		case "subscription":
			durationDays := plan.Duration
			if durationDays <= 0 {
				durationDays = 30
			}
			now := time.Now()
			startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate := startDate.AddDate(0, 0, durationDays)

			sub := &models.Subscription{
				UserID:          userID,
				PlanID:          planID,
				StatusID:        models.StatusSubscriptionActiveID,
				Slug:            uuid.New(),
				StartDate:       startDate,
				EndDate:         endDate,
				PaymentMethodID: models.PaymentMethodCreditCardID,
				AutoRenew:       autoRenew,
			}
			if err := txRepo.CreateSubscription(ctx, sub); err != nil {
				return err
			}
			subID = &sub.ID

		case "daypass":
			var vDate time.Time
			if parsed, err := time.Parse("2006-01-02", visitDateStr); err == nil {
				vDate = parsed
			} else {
				now := time.Now()
				vDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			}

			fullName := "Cliente"
			phone := ""
			email := ""
			if fn := sess.Metadata["full_name"]; fn != "" {
				fullName = fn
			}
			if ph := sess.Metadata["phone"]; ph != "" {
				phone = ph
			}
			if em := sess.Metadata["email"]; em != "" {
				email = em
			}

			var authID uint64
			fmt.Sscanf(sess.Metadata["auth_id"], "%d", &authID)
			if authID > 0 && (fullName == "Cliente" || email == "") {
				if profile, err := txRepo.GetUserProfileByAuthID(ctx, authID); err == nil && profile != nil {
					fullName = profile.FullName
					phone = profile.Phone
					email = profile.Email
				}
			}

			var uIDPtr *uint64
			if userID > 0 {
				uIDPtr = &userID
			}
			dp := &models.DayPass{
				UserID:    uIDPtr,
				GymID:     gymID,
				PlanID:    planID,
				Slug:      uuid.New(),
				VisitDate: vDate,
				FullName:  fullName,
				Phone:     phone,
				Email:     email,
				IsUsed:    false,
			}
			if err := txRepo.CreateDayPass(ctx, dp); err != nil {
				return err
			}
			dayPassID = &dp.ID
			createdDayPassID = &dp.ID
		}

		updates := map[string]any{
			"payment_status_id": models.PaymentStatusPaidID,
			"payment_method_id": models.PaymentMethodCreditCardID,
		}
		if subID != nil {
			updates["subscription_id"] = *subID
		}
		if dayPassID != nil {
			updates["day_pass_id"] = *dayPassID
		}

		return txRepo.Update(ctx, paymentID, updates)
	})

	if err != nil {
		return err
	}

	if createdDayPassID != nil && *createdDayPassID > 0 {
		dpID := *createdDayPassID
		go func() {
			dpDetails, err := uc.repo.GetDayPassDetailsByID(context.Background(), dpID)
			if err != nil || dpDetails == nil || strings.TrimSpace(dpDetails.Email) == "" {
				return
			}
			renderer, err := templates.NewEmailRenderer()
			if err != nil {
				return
			}
			name := strings.TrimSpace(dpDetails.FullName)
			if name == "" {
				name = "Visitante"
			}
			amount := float64(sess.AmountTotal) / 100.0
			data := templates.DayPassPaymentConfirmedData{
				Name:              name,
				GymName:           dpDetails.GymName,
				PlanName:          dpDetails.PlanName,
				Amount:            amount,
				PaymentMethodName: "Tarjeta de Crédito / Débito (Stripe)",
				TicketCode:        dpDetails.Slug,
				VisitDate:         dpDetails.VisitDate,
			}
			html, err := renderer.RenderDayPassPaymentConfirmed(data)
			if err == nil {
				_ = utils.EnqueueEmail([]string{dpDetails.Email}, fmt.Sprintf("Pago Confirmado: Tu Pase de un Día en %s", dpDetails.GymName), html)
			}
		}()
	}

	return nil
}

func (uc *UseCase) processCheckoutExpired(ctx context.Context, sess *stripe.CheckoutSession) error {
	paymentIDStr := sess.Metadata["payment_id"]
	if paymentIDStr == "" {
		return nil
	}

	var paymentID uint64
	fmt.Sscanf(paymentIDStr, "%d", &paymentID)
	if paymentID == 0 {
		return nil
	}

	return uc.repo.Update(ctx, paymentID, map[string]any{
		"payment_status_id": models.PaymentStatusCanceledID,
		"payment_method_id": models.PaymentMethodCreditCardID,
	})
}

func (uc *UseCase) GetSessionStatus(ctx context.Context, authID uint64, sessionID string) (*StripeSessionStatusResponse, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	stripeKey := uc.stripeSecretKey
	if stripeKey == "" {
		stripeKey = "sk_test_mock_fitbase_development_key"
	}
	stripe.Key = stripeKey

	sess, err := session.Get(sessionID, nil)
	if err != nil {
		return nil, errorsx.Internal("Error al consultar sesión de Stripe", err)
	}

	var paymentID uint64
	if pidStr := sess.Metadata["payment_id"]; pidStr != "" {
		fmt.Sscanf(pidStr, "%d", &paymentID)
	}

	return &StripeSessionStatusResponse{
		SessionID:     sess.ID,
		PaymentStatus: string(sess.PaymentStatus),
		Status:        string(sess.Status),
		PaymentID:     paymentID,
		AmountTotal:   float64(sess.AmountTotal) / 100.0,
		Currency:      string(sess.Currency),
	}, nil
}
