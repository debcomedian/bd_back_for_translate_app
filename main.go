package main

import (
	"context"
	"log"
	"os"

	"github.com/debcomedian/bd_back_for_translate_app/handlers"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var dbPool *pgxpool.Pool

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error
	dbPool, err = pgxpool.Connect(context.Background(), dbURL)

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	handlers.Dbpool = dbPool

	router := gin.Default()

	// Маршруты для работы со словами
	router.GET("/api/words", handlers.GetWords)
	router.GET("/api/words/type/:type", handlers.GetWordsByType)
	router.POST("/api/words", handlers.CreateWord)
	router.PUT("/api/words/:id", handlers.UpdateWord)
	router.DELETE("/api/words/:id", handlers.DeleteWord)

	// Маршруты для работы с текстами
	router.GET("/api/texts/small", handlers.GetTextsHandler("texts_small"))
	router.POST("/api/texts/small", handlers.CreateTextHandler("texts_small"))
	router.PUT("/api/texts/small/:id", handlers.UpdateTextHandler("texts_small"))
	router.DELETE("/api/texts/small/:id", handlers.DeleteTextHandler("texts_small"))

	router.GET("/api/texts/medium", handlers.GetTextsHandler("texts_medium"))
	router.POST("/api/texts/medium", handlers.CreateTextHandler("texts_medium"))
	router.PUT("/api/texts/medium/:id", handlers.UpdateTextHandler("texts_medium"))
	router.DELETE("/api/texts/medium/:id", handlers.DeleteTextHandler("texts_medium"))

	router.GET("/api/texts/large", handlers.GetTextsHandler("texts_large"))
	router.POST("/api/texts/large", handlers.CreateTextHandler("texts_large"))
	router.PUT("/api/texts/large/:id", handlers.UpdateTextHandler("texts_large"))
	router.DELETE("/api/texts/large/:id", handlers.DeleteTextHandler("texts_large"))

	// Маршруты для работы с грамматиками
	router.GET("/api/grammars", handlers.GetGrammarHandler("grammars"))
	router.POST("/api/grammars", handlers.CreateGrammarHandler("grammars"))
	router.PUT("/api/grammars/:id", handlers.UpdateGrammarHandler("grammars"))
	router.DELETE("/api/grammars/:id", handlers.DeleteGrammarHandler("grammars"))

	router.GET("/api/grammar/rules", handlers.GetGrammarRulesHandler("grammar_rules"))
	router.POST("/api/grammar/rules", handlers.CreateGrammarRulesHandler("grammar_rules"))
	router.PUT("/api/grammar/rules/:id", handlers.UpdateGrammarRulesHandler("grammar_rules"))
	router.DELETE("/api/grammar/rules/:id", handlers.DeleteGrammarRulesHandler("grammar_rules"))

	router.GET("/api/grammar/examples", handlers.GetGrammarExamplesHandler("grammar_examples"))
	router.POST("/api/grammar/examples", handlers.CreateGrammarExamplesHandler("grammar_examples"))
	router.PUT("/api/grammar/examples/:id", handlers.UpdateGrammarExamplesHandler("grammar_examples"))
	router.DELETE("/api/grammar/examples/:id", handlers.DeleteGrammarExamplesHandler("grammar_examples"))

	router.GET("/api/grammar/exceptions", handlers.GetGrammarExceptionsHandler("grammar_exceptions"))
	router.POST("/api/grammar/exceptions", handlers.CreateGrammarExceptionsHandler("grammar_exceptions"))
	router.PUT("/api/grammar/exceptions/:id", handlers.UpdateGrammarExceptionsHandler("grammar_exceptions"))
	router.DELETE("/api/grammar/exceptions/:id", handlers.DeleteGrammarExceptionsHandler("grammar_exceptions"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
