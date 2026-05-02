package main

import (
	"net/http"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/handlers"

	"github.com/gin-gonic/gin"
)

/* NewV1Router собирает только V1-контур backend.
 	 Старый main.go остаётся как legacy-reference.
	 V1 работает по offline-first модели:
	 - backend отдает content snapshot;
	 - клиент локально собирает session и считает progress;
	 - backend принимает sync только от авторизованных пользователей. */
func NewV1Router() *gin.Engine {
	router := gin.Default()

	router.GET("/health", healthHandler)

	v1 := router.Group("/v1")

	handlers.AuthEndpoints(v1.Group("/auth"))

	v1.POST("/admin/login", notImplemented("TODO: admin login endpoint"))

	admin := v1.Group("/admin")
	admin.Use(auth.Middleware("editor"))
	{
		admin.GET("/categories", handlers.GetCategories)
		admin.POST("/categories", handlers.CreateCategory)
		admin.PUT("/categories/:id", handlers.UpdateCategory)

		admin.GET("/words", handlers.GetWords)
		admin.POST("/words", handlers.CreateWord)
		admin.PUT("/words/:id", handlers.UpdateWord)

		admin.POST("/words/import", notImplemented("TODO: import words CSV"))
		admin.POST("/words/recalculate-meta", notImplemented("TODO: recalculate words_meta_base"))
	}

	content := v1.Group("/content")
	{
		content.GET("/snapshot", notImplemented("TODO: return content snapshot for mobile client"))
	}

	user := v1.Group("")
	user.Use(auth.Middleware(""))
	{
		profile := user.Group("/profile")
		{
			profile.GET("", notImplemented("TODO: get user profile"))
			profile.GET("/words/progress", notImplemented("TODO: get user word progress"))
		}

		sync := user.Group("/sync")
		{
			sync.POST("/push", notImplemented("TODO: push offline events"))
			sync.GET("/pull", notImplemented("TODO: pull remote updates"))
		}
	}

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"scope":  "v1",
	})
}

func notImplemented(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": message,
		})
	}
}