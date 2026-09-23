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

// recipeStep mirrors one step entry inside the steps JSON payload.
type recipeStep struct {
	StepNumber      int    `json:"step_number"`
	Description     string `json:"description"`
	DurationSeconds int    `json:"duration_seconds"`
}

// validateRecipe checks required fields and the steps JSON structure.
func validateRecipe(rec *model.BrewRecipe) error {
	if rec.Name == "" {
		return util.NewAppError(422, constants.CodeValidationError, "BrewRecipe[name] invalid: name is required")
	}
	if rec.WaterTemp < 80 || rec.WaterTemp > 100 {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("BrewRecipe[water_temp=%d] invalid: must be between 80 and 100", rec.WaterTemp))
	}
	var steps []recipeStep
	if err := json.Unmarshal([]byte(rec.Steps), &steps); err != nil {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("BrewRecipe[steps] invalid: %s", err.Error()))
	}
	if len(steps) == 0 {
		return util.NewAppError(422, constants.CodeValidationError, "BrewRecipe[steps] invalid: at least one step is required")
	}
	for i, st := range steps {
		if st.StepNumber < 1 || st.Description == "" || st.DurationSeconds <= 0 {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("BrewRecipe[steps][%d] invalid: step_number, description and duration_seconds are required", i))
		}
	}
	return nil
}

// Create shares a recipe.
func (s *RecipeService) Create(userID uint, rec *model.BrewRecipe) (*model.BrewRecipe, error) {
	rec.UserID = userID
	if rec.Steps == "" {
		rec.Steps = "[]"
	}
	if err := validateRecipe(rec); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeCreateFailed, rec.Name), "error", err)
		return nil, err
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
	if err := validateRecipe(rec); err != nil {
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

// Copy duplicates a public recipe into the user's own independent version.
func (s *RecipeService) Copy(userID, id uint) (*model.BrewRecipe, error) {
	src, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", id))
		}
		return nil, fmt.Errorf("recipe copy find: %w", err)
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
		s.logger.Error(fmt.Sprintf(constants.LogRecipeCopyFailed, id), "error", err)
		return nil, fmt.Errorf("recipe copy: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeCopySuccess, id, dup.ID), "source_id", id, "id", dup.ID)
	return dup, nil
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
