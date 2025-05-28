package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nicolaslucianob/checkinfp/models"
	"github.com/nicolaslucianob/checkinfp/routes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	if os.Getenv("GIN_MODE") != "release" {
		_ = godotenv.Load()
	}

	requiredEnv := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("Variável de ambiente %s está faltando ou vazia", env)
		}
	}

	db = initDB()
	log.Println("✅ Banco conectado com sucesso!")

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			frontLocal := "http://localhost:3000"
			frontRede := "http://" + strings.TrimSpace(os.Getenv("APP_HOST")) + ":3000"
			frontProd := "https://" + os.Getenv("FRONT_HOST")

			return origin == frontLocal || origin == frontRede || origin == frontProd
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	routes.RegisterRoutes(r, db)

	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func initDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password='%s' dbname=%s port=%s sslmode=disable TimeZone=America/Sao_Paulo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar com o banco de dados:", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.VolunteerCheckin{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}
