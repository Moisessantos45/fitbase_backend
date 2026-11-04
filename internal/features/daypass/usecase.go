package daypass

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

func sendDayPassEmailAsync(ticket *DayPassTicketResponse) {
	if ticket == nil || strings.TrimSpace(ticket.Email) == "" {
		return
	}
	email := strings.TrimSpace(ticket.Email)
	go func() {
		renderer, err := templates.NewEmailRenderer()
		if err != nil {
			return
		}
		name := strings.TrimSpace(ticket.FullName)
		if name == "" {
			name = "Visitante"
		}
		data := templates.DayPassTicketData{
			Name:       name,
			GymName:    ticket.GymName,
			PlanName:   ticket.PlanName,
			PlanPrice:  ticket.PlanPrice,
			VisitDate:  ticket.VisitDate,
			TicketCode: ticket.Slug,
			IsPaid:     ticket.IsPaid,
		}
		html, err := renderer.RenderDayPassTicket(data)
		if err == nil {
			subject := fmt.Sprintf("Tu Pase de un Día en %s", ticket.GymName)
			if !ticket.IsPaid {
				subject = fmt.Sprintf("Ticket de Pase de un Día (Pago en Mostrador) — %s", ticket.GymName)
			}
			_ = utils.EnqueueEmail([]string{email}, subject, html)
		}
	}()
}

func validateVisitDateAndSchedule(ctx context.Context, repo Repository, gymID uint64, visitDate time.Time) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	visitDay := time.Date(visitDate.Year(), visitDate.Month(), visitDate.Day(), 0, 0, 0, 0, now.Location())

	if visitDay.Before(today) {
		return errorsx.BadRequest("No se pueden agendar visitas para fechas pasadas")
	}

	if visitDay.Equal(today) {
		schedules, err := repo.GetSchedulesByGymID(ctx, gymID)
		if err != nil {
			return err
		}

		dayOfWeek := int(now.Weekday())
		var todaySchedule *models.GymSchedule
		for i := range schedules {
			if schedules[i].DayOfWeek == dayOfWeek {
				todaySchedule = &schedules[i]
				break
			}
		}

		if todaySchedule == nil || todaySchedule.IsClosed {
			return errorsx.BadRequest("El gimnasio se encuentra cerrado hoy")
		}

		currentMinutes := now.Hour()*60 + now.Minute()
		if currentMinutes >= todaySchedule.CloseMinutes {
			return errorsx.BadRequest("El horario de atención del gimnasio ha concluido por hoy")
		}
	}

	return nil
}

func (uc *UseCase) PurchaseDayPass(ctx context.Context, authID uint64, req CreateDayPassRequest) (*DayPassTicketResponse, error) {
	if authID == 0 || req.GymID == 0 || req.PlanID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	userID, fullName, phone, email, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de usuario no encontrado")
	}

	visitDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.VisitDate))
	if err != nil {
		return nil, errorsx.BadRequest("Formato de fecha de visita inválido, use YYYY-MM-DD")
	}

	if err := validateVisitDateAndSchedule(ctx, uc.repo, req.GymID, visitDate); err != nil {
		return nil, err
	}

	planModel, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || planModel.GymID != req.GymID || !planModel.Active {
		return nil, errorsx.BadRequest("Plan no válido para esta sede")
	}

	if planModel.Duration > 1 && (planModel.MaxVisits == nil || *planModel.MaxVisits > 1) {
		return nil, errorsx.BadRequest("El plan seleccionado no es de un solo uso. Para planes de membresía, el socio debe crear una cuenta y suscribirse al plan")
	}

	dayPass, err := NewDayPass(req.GymID, req.PlanID, &userID, fullName, phone, email, visitDate)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, dayPass); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, dayPass.Slug)
	if err == nil && ticket != nil {
		sendDayPassEmailAsync(ticket)
	}
	return ticket, err
}

func (uc *UseCase) RegisterGuest(ctx context.Context, authID uint64, req RegisterGuestRequest) (*DayPassTicketResponse, error) {
	if authID == 0 || req.GymID == 0 || req.PlanID == 0 {
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
		return nil, errorsx.Forbidden("No tienes autorización para registrar visitas en esta sede")
	}

	var visitDate time.Time
	trimmedDate := strings.TrimSpace(req.VisitDate)
	if trimmedDate == "" {
		now := time.Now()
		visitDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		parsed, err := time.Parse("2006-01-02", trimmedDate)
		if err != nil {
			return nil, errorsx.BadRequest("Formato de fecha inválido, use YYYY-MM-DD")
		}
		visitDate = parsed
	}

	if err := validateVisitDateAndSchedule(ctx, uc.repo, req.GymID, visitDate); err != nil {
		return nil, err
	}

	planModel, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || planModel.GymID != req.GymID || !planModel.Active {
		return nil, errorsx.BadRequest("Plan no válido para esta sede")
	}

	if planModel.Duration > 1 && (planModel.MaxVisits == nil || *planModel.MaxVisits > 1) {
		return nil, errorsx.BadRequest("El plan seleccionado no es de un solo uso. Para planes de membresía, el socio debe crear una cuenta y suscribirse al plan")
	}

	dayPass, err := NewDayPass(req.GymID, req.PlanID, nil, req.FullName, req.Phone, req.Email, visitDate)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, dayPass); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, dayPass.Slug)
	if err == nil && ticket != nil {
		sendDayPassEmailAsync(ticket)
	}
	return ticket, err
}

func (uc *UseCase) CreatePublicDayPass(ctx context.Context, req RegisterGuestRequest) (*DayPassTicketResponse, error) {
	if req.GymID == 0 || req.PlanID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	var visitDate time.Time
	trimmedDate := strings.TrimSpace(req.VisitDate)
	if trimmedDate == "" {
		now := time.Now()
		visitDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		parsed, err := time.Parse("2006-01-02", trimmedDate)
		if err != nil {
			return nil, errorsx.BadRequest("Formato de fecha inválido, use YYYY-MM-DD")
		}
		visitDate = parsed
	}

	if err := validateVisitDateAndSchedule(ctx, uc.repo, req.GymID, visitDate); err != nil {
		return nil, err
	}

	planModel, err := uc.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || planModel.GymID != req.GymID || !planModel.Active {
		return nil, errorsx.BadRequest("Plan no válido para esta sede")
	}

	if planModel.Duration > 1 && (planModel.MaxVisits == nil || *planModel.MaxVisits > 1) {
		return nil, errorsx.BadRequest("El plan seleccionado no es de un solo uso. Para planes de membresía, el socio debe crear una cuenta y suscribirse al plan")
	}

	dayPass, err := NewDayPass(req.GymID, req.PlanID, nil, req.FullName, req.Phone, req.Email, visitDate)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	if err := uc.repo.Create(ctx, dayPass); err != nil {
		return nil, err
	}

	ticket, err := uc.repo.GetTicketBySlug(ctx, dayPass.Slug)
	if err == nil && ticket != nil {
		sendDayPassEmailAsync(ticket)
	}
	return ticket, err
}

func (uc *UseCase) GetTicketBySlug(ctx context.Context, slugStr string) (*DayPassTicketResponse, error) {
	slug, err := uuid.Parse(strings.TrimSpace(slugStr))
	if err != nil {
		return nil, errorsx.BadRequest("Código de ticket inválido")
	}

	return uc.repo.GetTicketBySlug(ctx, slug)
}

func (uc *UseCase) GetAllByGym(ctx context.Context, authID, gymID uint64, dateStr, search string, page, pageSize int) (*PaginatedDayPassSummary, error) {
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
		return nil, errorsx.Forbidden("No tienes permiso para consultar los pases de esta sede")
	}

	var filterDate time.Time
	trimmedDate := strings.TrimSpace(dateStr)
	if trimmedDate != "" {
		parsed, err := time.Parse("2006-01-02", trimmedDate)
		if err == nil {
			filterDate = parsed
		}
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	passes, total, err := uc.repo.GetAllByGymID(ctx, gymID, filterDate, search, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedDayPassSummary{
		Data: passes,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
