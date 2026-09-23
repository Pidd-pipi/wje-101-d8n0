package router

import (
	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/handler"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
)

func registerRecipeRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.RecipeHandler, limiter *middleware.RateLimiter) {
	recipes := v1.Group("/recipes")
	recipes.GET("", h.List)
	recipes.GET("/:id", h.Get)
	auth := recipes.Group("", middleware.AuthRequired(cfg))
	auth.POST("", limiter.Limit(), h.Create)
	auth.PUT("/:id", h.Update)
	auth.POST("/:id/copy", limiter.Limit(), h.Copy)
}
