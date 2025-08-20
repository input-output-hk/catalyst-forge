package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// PromotionDeps contains dependencies for promotion routes
type PromotionDeps struct {
	H *handlers.PromotionHandler
}

// RegisterPromotions registers v1 promotion routes
func RegisterPromotions(r *gin.Engine, deps PromotionDeps) {
	v1 := r.Group("/api/v1")
	{
		promotions := v1.Group("/promotions")
		{
			promotions.POST("", deps.H.Create)
			promotions.GET("", deps.H.List)
			promotions.GET(":promotion_id", deps.H.GetByID)
			promotions.PATCH(":promotion_id", deps.H.Update)
			promotions.DELETE(":promotion_id", deps.H.Delete)
		}
	}
}
