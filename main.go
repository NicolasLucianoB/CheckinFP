package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nicolaslucianob/checkinfp/models"
	"github.com/nicolaslucianob/checkinfp/routes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	if os.Getenv("GIN_MODE") != "release" {
		_ = godotenv.Load()
	}

	db = initDB()
	log.Println("Banco conectado com sucesso:", db)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://" + os.Getenv("APP_HOST") + ":3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Chamando a função de registro de rotas
	routes.RegisterRoutes(r, db)

	r.Run("0.0.0.0:8080") // rede local
}

func initDB() *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar com o banco de dados:", err)
	}

	db.AutoMigrate(&models.User{}, &models.Volunteer{}, &models.VolunteerCheckin{})
	return db
}
