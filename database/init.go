package database

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error:", err)
	}

	vars := fmt.Sprintf(`
		host=%s 
		port=%s 
		user=%s 
		password=%s 
		dbname=%s 
		sslmode=disable`,
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
	)

	var erro error
	DB, erro = gorm.Open(postgres.Open(vars), &gorm.Config{})
	if erro != nil {
		log.Fatal("❌ Erro ao conectar ao banco:", erro)
	}

	if err := DB.Use(otelgorm.NewPlugin(
		otelgorm.WithDBName("postgres"), // aparece como tag no Sentry
	)); err != nil {
		log.Fatalf("falha ao ativar otelgorm: %v", err)
	}

	fmt.Println("Database connected successfully 🎉")

	/* 	sentry.AddSentryCallbacks(DB) */

}
