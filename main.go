package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

type VolunteerCheckin struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	CheckinTime time.Time `gorm:"autoCreateTime"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := initDB()
	log.Println("Banco conectado com sucesso:", db)

	r := gin.Default()
	r.GET("/generate/:name", generateQRCode)
	r.GET("/checkin/:name", checkIn)

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

	db.AutoMigrate(&VolunteerCheckin{})
	return db
}

func generateQRCode(c *gin.Context) {
	name := c.Param("name")

	scanURL := fmt.Sprintf("http://127.0.0.1:8080/checkin/%s", name)

	os.MkdirAll("qrcodes", os.ModePerm)
	filename := fmt.Sprintf("qrcodes/%s.png", name)

	err := qrcode.WriteFile(scanURL, qrcode.Medium, 256, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR Code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "QR Code generated",
		"file":    filename,
		"url":     scanURL,
	})
}

func checkIn(c *gin.Context) {
	name := c.Param("name")
	checkin := VolunteerCheckin{Name: name}
	if err := db.Create(&checkin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check in"})
		return
	}

	log.Printf("Check-in realizado com sucesso: %s", name)
	c.JSON(http.StatusOK, gin.H{
		"message": "Check-in successful",
		"name":    name,
	})
}
