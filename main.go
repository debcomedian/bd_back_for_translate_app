package main

import (
	"log"
	"os"

	"bd_back_for_translate_app/database"
	"bd_back_for_translate_app/handlers"
	"bd_back_for_translate_app/services"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	database.Init()

	if err := services.BootstrapContentIfNeeded(database.DB); err != nil {
		log.Fatalf("Content bootstrap failed: %v", err)
	}

	handlers.DB = database.DB

	router := NewV1Router()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
