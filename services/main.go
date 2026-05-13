package services

import (
	"encoding/json"
	"log"
	"os"

	"bd_back_for_translate_app/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	inputPath := os.Getenv("ACTIVE_BANK_PATH")
	if inputPath == "" {
		inputPath = "data/lexicon/active_bank.jsonl"
	}

	database.Init()

	report, err := ImportActiveBank(database.DB, inputPath)
	if err != nil {
		log.Fatalf("import active bank failed: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		log.Fatalf("encode report failed: %v", err)
	}
}
