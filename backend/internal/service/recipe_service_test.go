package service

import (
	"testing"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
)

func TestRecipeCreateMissingName(t *testing.T) {
	svc := NewRecipeService(repository.NewBrewRecipeRepository(nil), newTestLogger())
	r := &model.BrewRecipe{WaterTemp: 92, Steps: `[{"step_number":1,"description":"闷蒸","duration_seconds":30}]`}
	if _, err := svc.Create(1, r); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestRecipeCreateInvalidWaterTemp(t *testing.T) {
	svc := NewRecipeService(repository.NewBrewRecipeRepository(nil), newTestLogger())
	r := &model.BrewRecipe{Name: "测试配方", WaterTemp: 60, Steps: `[{"step_number":1,"description":"闷蒸","duration_seconds":30}]`}
	if _, err := svc.Create(1, r); err == nil {
		t.Error("expected error for water temp out of range")
	}
}

func TestRecipeCreateEmptySteps(t *testing.T) {
	svc := NewRecipeService(repository.NewBrewRecipeRepository(nil), newTestLogger())
	r := &model.BrewRecipe{Name: "测试配方", WaterTemp: 92, Steps: "[]"}
	if _, err := svc.Create(1, r); err == nil {
		t.Error("expected error for empty steps")
	}
}

func TestRecipeCreateBrokenSteps(t *testing.T) {
	svc := NewRecipeService(repository.NewBrewRecipeRepository(nil), newTestLogger())
	r := &model.BrewRecipe{Name: "测试配方", WaterTemp: 92, Steps: `[{"step_number":1,"duration_seconds":30}]`}
	if _, err := svc.Create(1, r); err == nil {
		t.Error("expected error for step missing description")
	}
}
