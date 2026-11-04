package checkin

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
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
		return errorsx.BadRequest("El gimnasio se encuentra cerrado hoy")
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	if currentMinutes < todaySchedule.OpenMinutes || currentMinutes >= todaySchedule.CloseMinutes {
		return errorsx.BadRequest("Fuera del horario de atención del gimnasio")
	}

	return nil
}

func (uc *UseCase) ValidateAccess(ctx context.Context, authID uint64, req ValidateAccessRequest) (*AccessResultResponse, error) {
	if authID == 0 || req.GymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	slugStr := strings.TrimSpace(req.Slug)
	emailStr := strings.ToLower(strings.TrimSpace(req.Email))

	if slugStr == "" && emailStr == "" {
		return nil, errorsx.BadRequest("Debes ingresar el código QR (slug) o el correo electrónico del socio")
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
		return nil, errorsx.Forbidden("No tienes autorización para operar los accesos en esta sede")
	}

	if err := validateGymSchedule(ctx, uc.repo, req.GymID); err != nil {
		return nil, err
	}

	var dayPass *models.DayPass
	var subscription *models.Subscription

	if slugStr != "" {
		slug, err := uuid.Parse(slugStr)
		if err != nil {
			return nil, errorsx.BadRequest("Código QR/slug inválido")
		}

		dp, err := uc.repo.GetDayPassBySlug(ctx, slug)
		if err == nil && dp != nil {
			dayPass = dp
		} else {
			sub, err := uc.repo.GetSubscriptionBySlug(ctx, slug)
			if err == nil && sub != nil {
				subscription = sub
			}
		}
	} else {
		sub, err := uc.repo.GetSubscriptionByEmail(ctx, emailStr, req.GymID)
		if err == nil && sub != nil {
			subscription = sub
		} else {
			dp, err := uc.repo.GetDayPassByEmail(ctx, emailStr, req.GymID)
			if err == nil && dp != nil {
				dayPass = dp
			}
		}
	}

	if dayPass != nil {
		if dayPass.GymID != req.GymID {
			return nil, errorsx.BadRequest("El pase no corresponde a esta sede")
		}

		now := time.Now()
		if dayPass.VisitDate.Format("2006-01-02") != now.Format("2006-01-02") {
			return nil, errorsx.BadRequest(fmt.Sprintf("El pase diario no es válido para la fecha de hoy (%s)", dayPass.VisitDate.Format("2006-01-02")))
		}

		if dayPass.IsUsed {
			return nil, errorsx.BadRequest("Pase ya utilizado")
		}

		isPaid, err := uc.repo.IsDayPassPaid(ctx, dayPass.ID)
		if err != nil || !isPaid {
			return nil, errorsx.BadRequest("El pase diario no ha sido pagado. Debe abonarse en recepción antes de ingresar")
		}

		planName := "Pase de un Día"
		planModel, err := uc.repo.GetPlanByID(ctx, dayPass.PlanID)
		if err == nil && planModel != nil {
			planName = planModel.Name
		}

		accessTypeID := models.AccessTypeDayPassID
		accessTypeName := "Día de Pase"
		if dayPass.UserID == nil {
			accessTypeID = models.AccessTypeGuestID
			accessTypeName = "Invitado"
		}

		var checkedInAt time.Time
		err = uc.repo.WithTransaction(func(txRepo Repository) error {
			if err := txRepo.MarkDayPassAsUsed(ctx, dayPass.ID); err != nil {
				return err
			}

			checkIn, err := NewCheckIn(req.GymID, accessTypeID, dayPass.UserID, &dayPass.ID, nil, &authID)
			if err != nil {
				return err
			}

			if err := txRepo.Register(ctx, checkIn); err != nil {
				return err
			}

			checkedInAt = checkIn.CheckedInAt
			return nil
		})

		if err != nil {
			return nil, err
		}

		return &AccessResultResponse{
			Allowed:     true,
			Message:     "Acceso concedido",
			AccessType:  accessTypeName,
			FullName:    dayPass.FullName,
			PlanName:    planName,
			CheckedInAt: checkedInAt,
		}, nil
	}

	if subscription != nil {
		planModel, err := uc.repo.GetPlanByID(ctx, subscription.PlanID)
		if err != nil || planModel.GymID != req.GymID {
			return nil, errorsx.BadRequest("La membresía no corresponde a esta sede")
		}

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startDate := time.Date(subscription.StartDate.Year(), subscription.StartDate.Month(), subscription.StartDate.Day(), 0, 0, 0, 0, now.Location())
		endDate := time.Date(subscription.EndDate.Year(), subscription.EndDate.Month(), subscription.EndDate.Day(), 0, 0, 0, 0, now.Location())

		if today.Before(startDate) || today.After(endDate) {
			return nil, errorsx.BadRequest(fmt.Sprintf("Membresía fuera de vigencia (vigente del %s al %s)",
				subscription.StartDate.Format("2006-01-02"),
				subscription.EndDate.Format("2006-01-02"),
			))
		}

		if subscription.StatusID == models.StatusSubscriptionExpiredID || subscription.StatusID == models.StatusSubscriptionPendingID {
			return nil, errorsx.BadRequest("La membresía no se encuentra disponible para acceso")
		}

		userName, err := uc.repo.GetUserDetailsByUserID(ctx, subscription.UserID)
		if err != nil {
			userName = "Socio"
		}

		checkIn, err := NewCheckIn(req.GymID, models.AccessTypeMembershipID, &subscription.UserID, nil, &subscription.ID, &authID)
		if err != nil {
			return nil, err
		}

		if err := uc.repo.Register(ctx, checkIn); err != nil {
			return nil, err
		}

		return &AccessResultResponse{
			Allowed:     true,
			Message:     "Acceso concedido",
			AccessType:  "Membresía",
			FullName:    userName,
			PlanName:    planModel.Name,
			CheckedInAt: checkIn.CheckedInAt,
		}, nil
	}

	if emailStr != "" {
		return nil, errorsx.NotFound("No se encontró ninguna membresía o pase para el correo especificado en esta sede")
	}

	return nil, errorsx.NotFound("Ticket o credencial no encontrada")
}

func (uc *UseCase) GetMyHistory(ctx context.Context, authID uint64, page, pageSize int) (*PaginatedCheckInSummary, error) {
	if authID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	userID, _, err := uc.repo.GetUserDetailsByAuthID(ctx, authID)
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

	return &PaginatedCheckInSummary{
		Data: items,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) GetAllByGym(ctx context.Context, authID, gymID uint64, fromStr, toStr string, page, pageSize int) (*PaginatedCheckInSummary, error) {
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
		return nil, errorsx.Forbidden("No tienes permiso para consultar los accesos de esta sede")
	}

	var fromDate, toDate time.Time
	trimmedFrom := strings.TrimSpace(fromStr)
	if trimmedFrom != "" {
		if parsed, err := time.Parse("2006-01-02", trimmedFrom); err == nil {
			fromDate = parsed
		}
	}

	trimmedTo := strings.TrimSpace(toStr)
	if trimmedTo != "" {
		if parsed, err := time.Parse("2006-01-02", trimmedTo); err == nil {
			toDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, parsed.Location())
		}
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	items, total, err := uc.repo.GetAllByGymID(ctx, gymID, fromDate, toDate, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedCheckInSummary{
		Data: items,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
