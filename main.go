package main

import (
	"log"
	"os"

	"bd_back_for_translate_app/database"
	"bd_back_for_translate_app/handlers"
	"bd_back_for_translate_app/lexicon"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения системы")
	}

	database.Init()

	if err := lexicon.BootstrapContentIfNeeded(database.DB); err != nil {
		log.Fatalf("Ошибка инициализации словарного контура: %v", err)
	}

	handlers.DB = database.DB

	router := NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Не удалось запустить HTTP-сервер: %v", err)
	}
}
