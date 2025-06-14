package main

import (
	// "fmt"
	"log"
	"os"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/database"
	"bd_back_for_translate_app/handlers"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	// "golang.org/x/crypto/bcrypt"
)

var dbPool *pgxpool.Pool

func main() {

	var err error

	if err = godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	database.Init()
	handlers.DB = database.DB

	// h,_ := bcrypt.GenerateFromPassword([]byte("StrongP@ss"), bcrypt.DefaultCost)
    // fmt.Println(string(h))

	//handlers.SttClient, err = handlers.NewSTTClient("./stt/stt_daemon.py")
	// if err != nil {
	// 	log.Fatalf("Ошибка запуска нейросетевого процесса: %v", err)
	// }

	// handlers.TtsClient, err = handlers.NewTTSClient("./tts/tts_daemon.py")
	// if err != nil {
	// 	log.Fatalf("TTS daemon start failed: %v", err)
	// }

	// if err := handlers.GenerateMissingWordAudio(); err != nil {
	// 	log.Printf("TTS batch error: %v", err)
	// }

	router := gin.Default()

	router.POST("/api/upload/data", handlers.UploadDataHandler)

	router.GET("/api/categories", handlers.GetCategories)
	router.POST("/api/categories", handlers.CreateCategory)
	router.PUT("/api/categories/:id", handlers.UpdateCategory)
	router.DELETE("/api/categories/:id", handlers.DeleteCategory)

	router.GET("/api/words", handlers.GetWords)
	router.POST("/api/words", handlers.CreateWord)
	router.PUT("/api/words/:id", handlers.UpdateWord)
	router.DELETE("/api/words/:id", handlers.DeleteWord)

	router.GET("/api/texts", handlers.GetTexts)
	router.POST("/api/texts", handlers.CreateText)
	router.PUT("/api/texts/:id", handlers.UpdateText)
	router.DELETE("/api/texts/:id", handlers.DeleteText)

	router.GET("/api/grammars", handlers.GetGrammars)
	router.POST("/api/grammars", handlers.CreateGrammars)
	router.PUT("/api/grammars/:id", handlers.UpdateGrammars)
	router.DELETE("/api/grammars/:id", handlers.DeleteGrammars)

	router.GET("/api/grammar/rules", handlers.GetGrammarRules)
	router.POST("/api/grammar/rules", handlers.CreateGrammarRules)
	router.PUT("/api/grammar/rules/:id", handlers.UpdateGrammarRules)
	router.DELETE("/api/grammar/rules/:id", handlers.DeleteGrammarRules)

	router.GET("/api/grammar/examples", handlers.GetGrammarExamples)
	router.POST("/api/grammar/examples", handlers.CreateGrammarExamples)
	router.PUT("/api/grammar/examples/:id", handlers.UpdateGrammarExamples)
	router.DELETE("/api/grammar/examples/:id", handlers.DeleteGrammarExamples)

	router.GET("/api/grammar/exceptions", handlers.GetGrammarExceptions)
	router.POST("/api/grammar/exceptions", handlers.CreateGrammarExceptions)
	router.PUT("/api/grammar/exceptions/:id", handlers.UpdateGrammarExceptions)
	router.DELETE("/api/grammar/exceptions/:id", handlers.DeleteGrammarExceptions)

	router.GET("/api/users", handlers.GetUsers)
	router.POST("/api/users", handlers.CreateUser)
	router.PUT("/api/users/:id", handlers.UpdateUser)
	router.DELETE("/api/users/:id", handlers.DeleteUser)

	router.GET("/api/users/:id/words/progress", handlers.GetUserWordProgress)
	router.POST("/api/users/:id/words/:wordId/progress", handlers.CreateUserWordProgress)
	router.PUT("/api/users/:id/words/:wordId/progress", handlers.UpdateUserWordProgress)
	router.DELETE("/api/users/:id/words/:wordId/progress", handlers.DeleteUserWordProgress)

	router.GET("/api/text_tests", handlers.GetTextTests)
	router.POST("/api/text_tests", handlers.CreateTextTest)
	router.PUT("/api/text_tests/:id", handlers.UpdateTextTest)
	router.DELETE("/api/text_tests/:id", handlers.DeleteTextTest)

	router.GET("/api/text_questions", handlers.GetAllTextQuestions)
	router.GET("/api/text_tests/:id/questions", handlers.GetTextQuestions)
	router.POST("/api/text_tests/:id/questions", handlers.CreateTextQuestion)
	router.PUT("/api/text_questions/:qid", handlers.UpdateTextQuestion)
	router.DELETE("/api/text_questions/:qid", handlers.DeleteTextQuestion)

	router.GET("/api/users/:id/text_tests/progress", handlers.GetUserTextTestProgress)
	router.POST("/api/users/:id/text_tests/:testId/progress", handlers.UpsertUserTextTestProgress)
	router.PUT("/api/users/:id/text_tests/:testId/progress", handlers.UpsertUserTextTestProgress)
	router.DELETE("/api/users/:id/text_tests/:testId/progress", handlers.DeleteUserTextTestProgress)

	router.GET("/api/users/:id/grammar/progress", handlers.GetUserGrammarProgress)
	router.POST("/api/users/:id/grammar/:grammarId/progress", handlers.UpsertUserGrammarProgress)
	router.DELETE("/api/users/:id/grammar/:grammarId/progress", handlers.DeleteUserGrammarProgress)

	router.GET("/api/grammar_tests", handlers.GetGrammarTests)
	router.GET("/api/grammar_tests/:id/questions", handlers.GetGrammarQuestions)
	router.POST  ("/api/grammar_tests",        handlers.CreateGrammarTest)  // ← NEW
	router.PUT   ("/api/grammar_tests/:id",    handlers.UpdateGrammarTest)  // ← NEW
	router.DELETE("/api/grammar_tests/:id",    handlers.DeleteGrammarTest)  // ← NEW

	router.POST("/api/grammar_tests/:id/questions", handlers.CreateGrammarQuestion)
	router.GET("/api/grammar_questions", handlers.GetAllGrammarQuestions)
	router.PUT("/api/grammar_questions/:qid", handlers.UpdateGrammarQuestion)
	router.DELETE("/api/grammar_questions/:qid", handlers.DeleteGrammarQuestion)

	router.GET ("/api/grammar_answers",                 handlers.GetAllGrammarAnswers)
	router.GET ("/api/grammar_questions/:id/answers",   handlers.GetGrammarAnswers)
	router.POST("/api/grammar_questions/:id/answers",   handlers.CreateGrammarAnswer)
	router.PUT ("/api/grammar_answers/:aid",            handlers.UpdateGrammarAnswer)
	router.DELETE("/api/grammar_answers/:aid",          handlers.DeleteGrammarAnswer)

	router.POST("/api/users/:id/grammar_tests/:testId/progress",handlers.UpsertUserGrammarTestProgress)

	handlers.AuthEndpoints(router.Group("/auth"))

	adm := router.Group("/admin")
    adm.Use(auth.Middleware("editor"))
    handlers.AdminAPI(adm, database.DB)

	router.StaticFile("/admin/panel", "./static/admin_panel.html")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
