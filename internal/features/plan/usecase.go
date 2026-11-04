package plan

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"strings"
)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) Service {
	return &UseCase{repo: repo}
}

func (uc *UseCase) validatePlanOwner(ctx context.Context, authID, planID uint64) error {
	if authID == 0 || planID == 0 {
		return errorsx.BadRequest("Parámetros inválidos")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return errorsx.NotFound("Perfil de dueño no encontrado")
	}

	planModel, err := uc.repo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, planModel.GymID)
	if err != nil {
		return err
	}

	if gymOwnerID != ownerID {
		return errorsx.Forbidden("No tienes permiso para modificar este plan")
	}

	return nil
}

func (uc *UseCase) Create(ctx context.Context, authID, gymID uint64, name, description, planType string, price, discount float64, duration int, maxVisits *int, items []PlanItemRequest) (*PlanDetailResponse, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
	if err != nil {
		return nil, errorsx.NotFound("Perfil de dueño no encontrado")
	}

	gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
	if err != nil {
		return nil, errorsx.NotFound("Gimnasio no encontrado")
	}

	if gymOwnerID != ownerID {
		return nil, errorsx.Forbidden("No tienes permiso para crear planes en este gimnasio")
	}

	planModel, err := New(gymID, name, description, planType, price, discount, duration, maxVisits)
	if err != nil {
		return nil, errorsx.BadRequest(err.Error())
	}

	planItems := make([]models.PlanItem, 0, len(items))
	for _, it := range items {
		planItems = append(planItems, models.PlanItem{
			Name:        it.Name,
			Description: it.Description,
		})
	}

	if err := uc.repo.CreateWithItems(ctx, planModel, planItems); err != nil {
		return nil, err
	}

	return uc.repo.GetDetailsByID(ctx, planModel.ID)
}

func (uc *UseCase) Update(ctx context.Context, authID, planID uint64, name, description, planType string, price, discount *float64, duration *int, maxVisits *int, active *bool) error {
	err := uc.validatePlanOwner(ctx, authID, planID)
	if err != nil {
		return err
	}

	updateMap := BuildPlanUpdateData(name, description, planType, price, discount, duration, maxVisits, active)
	if len(updateMap) == 0 {
		return nil
	}

	return uc.repo.Update(ctx, planID, updateMap)
}

func (uc *UseCase) AddItem(ctx context.Context, authID, planID uint64, req PlanItemRequest) (*PlanPublicItem, error) {
	err := uc.validatePlanOwner(ctx, authID, planID)
	if err != nil {
		return nil, err
	}

	trimmedName := strings.TrimSpace(req.Name)
	if trimmedName == "" {
		return nil, errorsx.BadRequest("El nombre del beneficio es requerido")
	}

	item := &models.PlanItem{
		PlanID:      planID,
		Name:        trimmedName,
		Description: strings.TrimSpace(req.Description),
	}

	if err := uc.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}

	return &PlanPublicItem{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
	}, nil
}

func (uc *UseCase) UpdateItem(ctx context.Context, authID, planID, itemID uint64, req UpdatePlanItemRequest) error {
	if itemID == 0 {
		return errorsx.BadRequest("ID de beneficio inválido")
	}

	err := uc.validatePlanOwner(ctx, authID, planID)
	if err != nil {
		return err
	}

	item, err := uc.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	if item.PlanID != planID {
		return errorsx.BadRequest("El beneficio no pertenece a este plan")
	}

	updateData := BuildPlanItemUpdateData(req)
	if len(updateData) == 0 {
		return nil
	}

	return uc.repo.UpdateItem(ctx, itemID, updateData)
}

func (uc *UseCase) DeleteItem(ctx context.Context, authID, planID, itemID uint64) error {
	if itemID == 0 {
		return errorsx.BadRequest("ID de beneficio inválido")
	}

	err := uc.validatePlanOwner(ctx, authID, planID)
	if err != nil {
		return err
	}

	item, err := uc.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	if item.PlanID != planID {
		return errorsx.BadRequest("El beneficio no pertenece a este plan")
	}

	return uc.repo.DeleteItem(ctx, itemID)
}

func (uc *UseCase) GetDetailsByID(ctx context.Context, id uint64) (*PlanDetailResponse, error) {
	if id == 0 {
		return nil, errorsx.BadRequest("ID de plan inválido")
	}

	return uc.repo.GetDetailsByID(ctx, id)
}

func (uc *UseCase) GetAllByGymID(ctx context.Context, authID, gymID uint64, activeOnly bool) ([]PlanDetailResponse, error) {
	if gymID == 0 {
		return nil, errorsx.BadRequest("ID de gimnasio inválido")
	}

	if !activeOnly {
		if authID == 0 {
			return nil, errorsx.ErrUnauthorized
		}

		isAuthorized := false
		ownerID, err := uc.repo.GetOwnerIDByAuthID(ctx, authID)
		if err == nil && ownerID > 0 {
			gymOwnerID, err := uc.repo.GetGymOwnerID(ctx, gymID)
			if err == nil && ownerID == gymOwnerID {
				isAuthorized = true
			}
		}

		if !isAuthorized {
			managerGymID, err := uc.repo.GetManagerGymID(ctx, authID)
			if err == nil && managerGymID == gymID {
				isAuthorized = true
			}
		}

		if !isAuthorized {
			return nil, errorsx.Forbidden("No tienes autorización para consultar los planes de esta sede")
		}
	}

	return uc.repo.GetAllByGymID(ctx, gymID, activeOnly)
}

func (uc *UseCase) GetEnrollmentPlans(ctx context.Context, authID, gymID uint64) (*EnrollmentResponse, error) {
	if authID == 0 || gymID == 0 {
		return nil, errorsx.BadRequest("Parámetros inválidos")
	}

	return uc.repo.GetEnrollmentPlans(ctx, authID, gymID)
}
