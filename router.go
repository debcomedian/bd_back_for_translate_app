package main

import (
	"net/http"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/database"
	"bd_back_for_translate_app/handlers"
	"bd_back_for_translate_app/lexicon"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", healthHandler)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
	}

	router.POST("/admin/login", handlers.AdminLogin)

	lexicon.RegisterRoutes(router, database.DB, auth.Middleware("admin"), auth.Middleware(""))

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "scope": "lexicon"})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:5173":     true,
			"http://127.0.0.1:5173":     true,
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
