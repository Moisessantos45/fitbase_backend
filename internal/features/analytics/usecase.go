package analytics

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"math"
	"strings"
	"time"
)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) Service {
	return &UseCase{repo: repo}
}

func (uc *UseCase) validateOwnerAccess(ctx context.Context, authID, gymID uint64) error {
	if authID == 0 || gymID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return errorsx.Forbidden("Solo el propietario de la sede puede acceder a los reportes")
	}

	gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
	if err != nil || ownerID != gymOwnerID {
		return errorsx.Forbidden("Solo el propietario de la sede puede acceder a los reportes")
	}

	return nil
}

func parseDateRange(fromStr, toStr string) (time.Time, time.Time) {
	var from, to time.Time
	trimmedFrom := strings.TrimSpace(fromStr)
	if trimmedFrom != "" {
		if parsed, err := time.Parse("2006-01-02", trimmedFrom); err == nil {
			from = parsed
		}
	}

	trimmedTo := strings.TrimSpace(toStr)
	if trimmedTo != "" {
		if parsed, err := time.Parse("2006-01-02", trimmedTo); err == nil {
			to = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, parsed.Location())
		}
	}

	return from, to
}

func (uc *UseCase) GetManagerShifts(ctx context.Context, authID, gymID uint64, fromStr, toStr string, page, pageSize int) (*PaginatedManagerShifts, error) {
	if err := uc.validateOwnerAccess(ctx, authID, gymID); err != nil {
		return nil, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	from, to := parseDateRange(fromStr, toStr)
	offset := (page - 1) * pageSize

	shifts, total, err := uc.repo.GetManagerShiftsReport(ctx, gymID, from, to, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errorsx.BadRequest("Página fuera de rango")
	}

	return &PaginatedManagerShifts{
		Data: shifts,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *UseCase) GetAttendanceMetrics(ctx context.Context, authID, gymID uint64, fromStr, toStr string) (*GymAttendanceReport, error) {
	if err := uc.validateOwnerAccess(ctx, authID, gymID); err != nil {
		return nil, err
	}

	from, to := parseDateRange(fromStr, toStr)
	return uc.repo.GetGymAttendanceReport(ctx, gymID, from, to)
}

func (uc *UseCase) GetCrossedRevenue(ctx context.Context, authID, gymID uint64, fromStr, toStr string) (*RevenueCrossedReport, error) {
	if err := uc.validateOwnerAccess(ctx, authID, gymID); err != nil {
		return nil, err
	}

	from, to := parseDateRange(fromStr, toStr)
	return uc.repo.GetRevenueCrossedReport(ctx, gymID, from, to)
}

func (uc *UseCase) GetClientHistory(ctx context.Context, authID, gymID, targetUserID uint64) (*ClientMembershipHistoryReport, error) {
	if err := uc.validateOwnerAccess(ctx, authID, gymID); err != nil {
		return nil, err
	}

	if targetUserID == 0 {
		return nil, errorsx.BadRequest("ID de usuario inválido")
	}

	return uc.repo.GetClientHistoryReport(ctx, gymID, targetUserID)
}

func (uc *UseCase) GetDashboardSummary(ctx context.Context, authID, gymID uint64) (*DashboardSummaryReport, error) {
	if err := uc.validateOwnerAccess(ctx, authID, gymID); err != nil {
		return nil, err
	}

	return uc.repo.GetDashboardSummary(ctx, gymID)
}
