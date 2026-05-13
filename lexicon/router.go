package lexicon

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, db *gorm.DB, adminMiddleware gin.HandlerFunc, userMiddleware gin.HandlerFunc) {
	handler := NewHandler(db)

	router.GET("/lexicon/health", handler.Health)

	content := router.Group("/content")
	{
		content.GET("/snapshot", handler.GetSnapshot)
	}

	public := router.Group("")
	{
		public.GET("/directions", handler.GetDirections)
	}

	admin := router.Group("/admin")
	if adminMiddleware != nil {
		admin.Use(adminMiddleware)
	}
	{
		admin.GET("/categories", handler.GetCategories)
		admin.POST("/categories", handler.CreateCategory)
		admin.PUT("/categories/:id", handler.UpdateCategory)
		
		admin.GET("/concepts", handler.GetConcepts)
		admin.GET("/forms", handler.GetForms)
		admin.GET("/directions", handler.GetDirections)
		
		admin.GET("/content/bank-quality", handler.GetBankQualityReport)
		admin.POST("/content/import-active-bank", handler.ImportActiveBank)
		admin.POST("/content/rebuild-from-current", handler.RebuildFromCurrent)
		admin.POST("/content/recalculate-directions", handler.RecalculateDirections)
	}

	user := router.Group("")
	if userMiddleware != nil {
		user.Use(userMiddleware)
	}
	{
		user.GET("/profile/directions/progress", handler.GetProgress)
		user.POST("/sync/push", handler.SyncPush)
		user.GET("/sync/pull", handler.SyncPull)
	}
}
