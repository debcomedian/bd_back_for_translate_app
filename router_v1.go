package main

import (
	"net/http"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/handlers"

	"github.com/gin-gonic/gin"
)

func NewV1Router() *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())
	router.GET("/health", healthHandler)

	v1 := router.Group("/v1")

	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
	}

	v1.POST("/admin/login", handlers.AdminLogin)

	admin := v1.Group("/admin")
	admin.Use(auth.Middleware("admin"))
	{
		admin.GET("/categories", handlers.GetCategories)
		admin.POST("/categories", handlers.CreateCategory)
		admin.PUT("/categories/:id", handlers.UpdateCategory)

		admin.GET("/words", handlers.GetWords)
		admin.POST("/words", handlers.CreateWord)
		admin.PUT("/words/:id", handlers.UpdateWord)

		admin.POST("/words/import", handlers.ImportWords)
		admin.POST("/words/recalculate-meta", handlers.RecalculateWordMeta)
	}

	content := v1.Group("/content")
	{
		content.GET("/snapshot", handlers.GetContentSnapshot)
	}

	user := v1.Group("")
	user.Use(auth.Middleware(""))
	{
		profile := user.Group("/profile")
		{
			profile.GET("", handlers.GetProfile)
			profile.GET("/words/progress", handlers.GetUserWordProgress)
		}

		sync := user.Group("/sync")
		{
			sync.POST("/push", handlers.SyncPush)
			sync.GET("/pull", handlers.SyncPull)
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

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowedOrigins := map[string]bool{
			"http://localhost:5173":  true,
			"http://127.0.0.1:5173": true,
			"http://192.168.0.100:5173": true,
		}

		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}