package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// RecipeService handles brew recipes.
type RecipeService struct {
	repo   *repository.BrewRecipeRepository
	logger *slog.Logger
}

// NewRecipeService creates a RecipeService.
func NewRecipeService(repo *repository.BrewRecipeRepository, logger *slog.Logger) *RecipeService {
	return &RecipeService{repo: repo, logger: logger}
}

// Create shares a recipe.
func (s *RecipeService) Create(userID uint, rec *model.BrewRecipe) (*model.BrewRecipe, error) {
	rec.UserID = userID
	if rec.Steps == "" {
		rec.Steps = "[]"
	}
	if err := s.repo.Create(rec); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeCreateFailed, rec.Name), "error", err)
		return nil, fmt.Errorf("recipe create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeCreateSuccess, rec.Name), "id", rec.ID)
	return rec, nil
}

// Get returns a recipe by id.
func (s *RecipeService) Get(id uint) (*model.BrewRecipe, error) {
	rec, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", id))
		}
		return nil, fmt.Errorf("recipe get: %w", err)
	}
	return rec, nil
}

// Update edits a recipe owned by the user.
func (s *RecipeService) Update(userID, id uint, rec *model.BrewRecipe) (*model.BrewRecipe, error) {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", id))
		}
		return nil, fmt.Errorf("recipe update find: %w", err)
	}
	if exist.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("BrewRecipe[id=%d] update failed: user_id=%d not owner", id, userID))
	}
	if err := validateRecipeFields(rec); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeUpdateFailed, id), "error", err)
		return nil, err
	}
	exist.Name = rec.Name
	exist.Device = rec.Device
	exist.WaterTemp = rec.WaterTemp
	exist.GrindSize = rec.GrindSize
	exist.Ratio = rec.Ratio
	exist.Steps = rec.Steps
	if err := s.repo.Update(exist); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeUpdateFailed, id), "error", err)
		return nil, fmt.Errorf("recipe update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeUpdateSuccess, id), "id", id)
	return exist, nil
}

// Fork copies a public recipe into the caller's own version, keeping the source name.
func (s *RecipeService) Fork(userID, id uint) (*model.BrewRecipe, error) {
	src, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", id))
		}
		return nil, fmt.Errorf("recipe fork find: %w", err)
	}
	dup := &model.BrewRecipe{
		UserID:    userID,
		Name:      src.Name,
		Device:    src.Device,
		WaterTemp: src.WaterTemp,
		GrindSize: src.GrindSize,
		Ratio:     src.Ratio,
		Steps:     src.Steps,
	}
	if err := s.repo.Create(dup); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeForkFailed, id), "error", err)
		return nil, fmt.Errorf("recipe fork: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeForkSuccess, id, dup.ID), "source_id", id, "id", dup.ID)
	return dup, nil
}

// validateRecipeFields checks required recipe fields and the step JSON shape.
func validateRecipeFields(rec *model.BrewRecipe) error {
	if rec.Name == "" {
		return util.NewAppError(422, constants.CodeValidationError, "BrewRecipe[name] validate failed: name is required")
	}
	if rec.WaterTemp <= 0 || rec.WaterTemp > 100 {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("BrewRecipe[water_temp=%d] validate failed: water temp must be 1-100", rec.WaterTemp))
	}
	var steps []map[string]interface{}
	if err := json.Unmarshal([]byte(rec.Steps), &steps); err != nil || len(steps) == 0 {
		return util.NewAppError(422, constants.CodeValidationError,
			"BrewRecipe[steps] validate failed: steps must be a non-empty JSON array")
	}
	return nil
}

// List filters recipes.
func (s *RecipeService) List(device, keyword string, page, pageSize int) ([]model.BrewRecipe, int64, error) {
	items, total, err := s.repo.List(device, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("recipe list: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeListSuccess, device), "total", total)
	return items, total, nil
}
