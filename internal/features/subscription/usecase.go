package subscription

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/templates"
	"fitbase/internal/shared/utils"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) Service {
	return &UseCase{repo: repo}
}

func sendSubscriptionEmailAsync(ticket *SubscriptionTicketResponse, email string, isRenewal bool) {
	targetEmail := strings.TrimSpace(email)
	if targetEmail == "" && ticket != nil {
		targetEmail = strings.TrimSpace(ticket.Email)
	}
	if ticket == nil || targetEmail == "" {
		return
	}
	go func() {
		renderer, err := templates.NewEmailRenderer()
		if err != nil {
			return
		}
		name := strings.TrimSpace(ticket.FullName)
		if name == "" {
			name = "Socio"
		}
		qrURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=%s", ticket.Slug)
		data := templates.SubscriptionCreatedData{
			Name:       name,
			GymName:    ticket.GymName,
			PlanName:   ticket.PlanName,
			PlanPrice:  ticket.PlanPrice,
			StartDate:  ticket.StartDate,
			EndDate:    ticket.EndDate,
			TicketCode: ticket.Slug,
			QRCodeURL:  qrURL,
			AutoRenew:  ticket.AutoRenew,
			IsRenewal:  isRenewal,
		}
		html, err := renderer.RenderSubscriptionCreated(data)
		if err == nil {
			subject := fmt.Sprintf("Tu Membresía y Credencial Digital — %s", ticket.GymName)
			if isRenewal {
				subject = fmt.Sprintf("¡Membresía Renovada Exitosamente! — %s", ticket.GymName)
			}
			_ = utils.EnqueueEmail([]string{targetEmail}, subject, html)
		}
	}()
}

func (uc *UseCase) PurchaseSubscription(ctx context.Context, authID uint64, req PurchaseSubscriptionRequest) (*SubscriptionTicketResponse, error) {
	if authID == 0 || req.PlanID == 0 || req.PaymentMethodID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	userID, _, _, email, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de usuario no encontrado")
	}

	planModel, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || !planModel.Active {
		return nil, errorsx.BadRequest("El plan seleccionado no existe o no está activo")
	}

	if planModel.Duration <= 0 {
		return nil, errorsx.BadRequest("El plan no tiene una duración válida")
	}

	activeSub, err := uc.repo.GetActiveByUserID(ctx, userID, planModel.GymID)
	if err == nil && activeSub != nil {
		return nil, errorsx.Conflict(fmt.Sprintf("Ya cuentas con una suscripción activa en esta sede hasta el %s", activeSub.EndDate.Format("2006-01-02")))
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var startDate time.Time
	trimmedStartDate := strings.TrimSpace(req.StartDate)
	if trimmedStartDate == "" {
		startDate = today
	} else {
		parsedDate, err := time.Parse("2006-01-02", trimmedStartDate)
		if err != nil {
			return nil, errorsx.BadRequest("Formato de fecha de inicio inválido, use YYYY-MM-DD")
		}
		if parsedDate.Before(today) {
			return nil, errorsx.BadRequest("La fecha de inicio no puede ser en el pasado")
		}
		startDate = parsedDate
	}

	endDate := startDate.AddDate(0, 0, planModel.Duration)

	subscription, err := NewSubscription(
		userID,
		planModel.ID,
		models.StatusSubscriptionActiveID,
		req.PaymentMethodID,
		startDate,
		endDate,
		req.AutoRenew,
	)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, subscription); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, subscription.Slug)
	if err != nil {
		return nil, err
	}

	ticket.QRCodeURL = fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=%s", ticket.Slug)
	sendSubscriptionEmailAsync(ticket, email, false)

	return ticket, nil
}

func (uc *UseCase) GetMySubscriptions(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedUserSubscriptionSummary, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	userID, _, _, _, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
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

	subs, total, err := uc.repo.GetAllByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := range subs {
		if parsedEnd, err := time.Parse("2006-01-02", subs[i].EndDate); err == nil {
			targetEnd := time.Date(parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(), 0, 0, 0, 0, now.Location())
			if !targetEnd.Before(today) {
				subs[i].RemainingDays = int(targetEnd.Sub(today).Hours()/24) + 1
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedUserSubscriptionSummary{
		Data: subs,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) GetTicketBySlug(ctx context.Context, slugStr string) (*SubscriptionTicketResponse, error) {
	slug, err := uuid.Parse(strings.TrimSpace(slugStr))
	if err != nil {
		return nil, errorsx.BadRequest("Código de ticket inválido")
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if parsedEndDate, err := time.Parse("2006-01-02", ticket.EndDate); err == nil {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		targetEnd := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 0, 0, 0, 0, now.Location())
		if !targetEnd.Before(today) {
			ticket.RemainingDays = int(targetEnd.Sub(today).Hours()/24) + 1
		}
	}

	ticket.QRCodeURL = fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=%s", ticket.Slug)

	return ticket, nil
}

func (uc *UseCase) GetAllByGym(ctx context.Context, authID, gymID uint64, statusID uint64, page, pageSize int) (*PaginatedSubscriptionSummary, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	isAuthorized := false

	managerGymID, err := uc.repo.GetManagerGymID(ctx, authID)
	if err == nil && managerGymID == gymID {
		isAuthorized = true
	} else {
		ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
		if err == nil {
			gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
			if err == nil && ownerID == gymOwnerID {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		return nil, errorsx.Forbidden("No tienes permiso para consultar las suscripciones de esta sede")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	subs, total, err := uc.repo.GetAllByGymID(ctx, gymID, statusID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedSubscriptionSummary{
		Data: subs,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) CancelSubscription(ctx context.Context, authID, subscriptionID uint64) (*SubscriptionTicketResponse, error) {
	if authID == 0 || subscriptionID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	sub, err := uc.repo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, errorsx.NotFound("Suscripción no encontrada")
	}

	if sub.StatusID != models.StatusSubscriptionActiveID {
		return nil, errorsx.BadRequest("Solo se pueden cancelar suscripciones activas")
	}

	planModel, err := uc.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, errorsx.NotFound("Plan asociado no encontrado")
	}

	isAuthorized := false

	userID, _, _, _, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
	if err == nil && userID == sub.UserID {
		isAuthorized = true
	}

	if !isAuthorized {
		ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
		if err == nil && ownerID > 0 {
			gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, planModel.GymID)
			if err == nil && ownerID == gymOwnerID {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		managerGymID, err := uc.repo.GetManagerGymID(ctx, authID)
		if err == nil && managerGymID == planModel.GymID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return nil, errorsx.Forbidden("No tienes permiso para cancelar esta suscripción")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := time.Date(sub.EndDate.Year(), sub.EndDate.Month(), sub.EndDate.Day(), 0, 0, 0, 0, now.Location())

	remainingDays := 0
	if !endDate.Before(today) {
		remainingDays = int(endDate.Sub(today).Hours()/24) + 1
	}

	if err := uc.repo.Cancel(ctx, subscriptionID); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, sub.Slug)
	if err != nil {
		return nil, err
	}
	ticket.RemainingDays = remainingDays
	ticket.QRCodeURL = fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=%s", ticket.Slug)

	return ticket, nil
}

func (uc *UseCase) RenewSubscription(ctx context.Context, authID, subscriptionID uint64, req RenewSubscriptionRequest) (*SubscriptionTicketResponse, error) {
	if authID == 0 || subscriptionID == 0 || req.PaymentMethodID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	sub, err := uc.repo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, errorsx.NotFound("Suscripción no encontrada")
	}

	planModel, err := uc.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || !planModel.Active {
		return nil, errorsx.BadRequest("El plan asociado no se encuentra disponible o activo")
	}

	isAuthorized := false
	userID, _, _, _, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
	if err == nil && userID == sub.UserID {
		isAuthorized = true
	}

	if !isAuthorized {
		ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
		if err == nil && ownerID > 0 {
			gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, planModel.GymID)
			if err == nil && ownerID == gymOwnerID {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		managerGymID, err := uc.repo.GetManagerGymID(ctx, authID)
		if err == nil && managerGymID == planModel.GymID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return nil, errorsx.Forbidden("No tienes autorización para renovar esta suscripción")
	}

	durationDays := planModel.Duration
	if durationDays <= 0 {
		durationDays = 30
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	subEndDate := time.Date(sub.EndDate.Year(), sub.EndDate.Month(), sub.EndDate.Day(), 0, 0, 0, 0, now.Location())

	// 1. Verificar si la suscripción que se intenta renovar aún se encuentra activa
	if sub.StatusID == models.StatusSubscriptionActiveID && !subEndDate.Before(today) {
		return nil, errorsx.Conflict(fmt.Sprintf("No se puede renovar la suscripción porque aún cuentas con una membresía activa hasta el %s", sub.EndDate.Format("2006-01-02")))
	}

	// 2. Verificar si el usuario cuenta con otra suscripción activa en esta misma sede
	activeSub, err := uc.repo.GetActiveByUserID(ctx, sub.UserID, planModel.GymID)
	if err == nil && activeSub != nil && activeSub.ID != sub.ID {
		return nil, errorsx.Conflict(fmt.Sprintf("No se puede renovar porque ya cuentas con una suscripción activa en esta sede hasta el %s", activeSub.EndDate.Format("2006-01-02")))
	}

	newStartDate := today
	newEndDate := today.AddDate(0, 0, durationDays)

	if err := uc.repo.Renew(ctx, subscriptionID, newStartDate, newEndDate, req.PaymentMethodID); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if parsedEndDate, err := time.Parse("2006-01-02", ticket.EndDate); err == nil {
		targetEnd := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 0, 0, 0, 0, now.Location())
		if !targetEnd.Before(today) {
			ticket.RemainingDays = int(targetEnd.Sub(today).Hours()/24) + 1
		}
	}

	ticket.QRCodeURL = fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=%s", ticket.Slug)

	sendSubscriptionEmailAsync(ticket, ticket.Email, true)

	return ticket, nil
}
