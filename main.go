package main

import (
	"context"
	"log"
	"os"

	"bd_back_for_translate_app/handlers"

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
	
	handlers.NeuralClient, err = handlers.NewNeuralNetClient("./neuralnet/neuralnet_daemon.py")
	if err != nil {
		log.Fatalf("Ошибка запуска нейросетевого процесса: %v", err)
	}

	router := gin.Default()

	router.POST("/api/upload/data", handlers.UploadDataHandler)

	router.GET("/api/categories", handlers.GetCategoriesHandler)
	router.POST("/api/categories", handlers.CreateCategoryHandler)
	router.PUT("/api/categories/:id", handlers.UpdateCategoryHandler)
	router.DELETE("/api/categories/:id", handlers.DeleteCategoryHandler)

	router.GET("/api/words", handlers.GetWords)
	router.GET("/api/words/type/:type", handlers.GetWordsByType)
	router.POST("/api/words", handlers.CreateWord)
	router.PUT("/api/words/:id", handlers.UpdateWord)
	router.DELETE("/api/words/:id", handlers.DeleteWord)

	router.GET("/api/texts", handlers.GetTextsHandler("texts"))
	router.POST("/api/texts", handlers.CreateTextHandler("texts"))
	router.PUT("/api/texts/:id", handlers.UpdateTextHandler("texts"))
	router.DELETE("/api/texts/:id", handlers.DeleteTextHandler("texts"))

	
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
