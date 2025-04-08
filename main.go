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

type Volunteer struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"not null;unique"`
	Role      string
	CreatedAt time.Time
}

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

	db = initDB()
	log.Println("Banco conectado com sucesso:", db)

	r := gin.Default()
	r.GET("/generate/:id", generateQRCode)
	r.GET("/checkin/:id", checkIn)
	r.GET("/checkins", listCheckins)
	r.GET("/ranking", checkinRanking)
	r.POST("/volunteers", createVolunteer)
	r.GET("/volunteers", listVolunteers)
	r.POST("/seed/volunteers", seedVolunteers)
	r.DELETE("/seed/volunteers", deleteFakeVolunteers)

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
	db.AutoMigrate(&Volunteer{})
	return db
}

func generateQRCode(c *gin.Context) {
	id := c.Param("id")

	scanURL := fmt.Sprintf("http://%s:8080/checkin/%s", os.Getenv("APP_HOST"), id)
	log.Printf("QR Code gerado com URL: %s", scanURL)

	os.MkdirAll("qrcodes", os.ModePerm)
	filename := fmt.Sprintf("qrcodes/%s.png", id)

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
	id := c.Param("id")

	var volunteer Volunteer
	if err := db.First(&volunteer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Voluntário não encontrado"})
		return
	}

	checkin := VolunteerCheckin{Name: volunteer.Name}
	if err := db.Create(&checkin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao registrar o check-in"})
		return
	}

	log.Printf("Check-in realizado com sucesso: %s", volunteer.Name)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="pt-BR">
		<head>
			<meta charset="UTF-8">
			<title>Check-in realizado</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #f0f0f0;
					display: flex;
					align-items: center;
					justify-content: center;
					height: 100vh;
				}
				.message {
					background: white;
					padding: 40px;
					border-radius: 10px;
					box-shadow: 0 0 15px rgba(0,0,0,0.1);
					text-align: center;
				}
				.message h1 {
					color: #28a745;
				}
			</style>
		</head>
		<body>
			<div class="message">
				<h1>✅ Check-in realizado com sucesso!</h1>
				<p>Olá, <strong>%s</strong>! Seu horário de chegada foi registrado.</p>
			</div>
		</body>
		</html>
	`, volunteer.Name)))
}

func listCheckins(c *gin.Context) {
	var checkins []VolunteerCheckin
	if err := db.Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

func checkinRanking(c *gin.Context) {
	type Result struct {
		Name          string
		TotalCheckins int
	}

	var results []Result

	err := db.Model(&VolunteerCheckin{}).
		Select("name, COUNT(*) as total_checkins").
		Group("name").
		Order("total_checkins DESC").
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate ranking"})
		return
	}

	c.JSON(http.StatusOK, results)
}

func createVolunteer(c *gin.Context) {
	var volunteer Volunteer
	if err := c.ShouldBindJSON(&volunteer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	if err := db.Create(&volunteer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível cadastrar o voluntário"})
		return
	}

	c.JSON(http.StatusCreated, volunteer)
}

func listVolunteers(c *gin.Context) {
	var volunteers []Volunteer

	name := c.Query("name")
	role := c.Query("role")

	query := db.Model(&Volunteer{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if role != "" {
		query = query.Where("role LIKE ?", "%"+role+"%")
	}

	if err := query.Find(&volunteers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar voluntários"})
		return
	}

	c.JSON(http.StatusOK, volunteers)
}

func seedVolunteers(c *gin.Context) {
	fakeVolunteers := []Volunteer{
		{Name: "Teste_01", Role: "Câmera"},
		{Name: "Teste_02", Role: "Som"},
		{Name: "Teste_03", Role: "Direção"},
		{Name: "Teste_04", Role: "Projeção"},
		{Name: "Teste_05", Role: "Transmissão"},
	}

	for i := range fakeVolunteers {
		v := &fakeVolunteers[i]
		if err := db.Create(v).Error; err != nil {
			log.Printf("Erro ao criar voluntário %s: %v", v.Name, err)
			continue
		}

		scanURL := fmt.Sprintf("http://%s:8080/checkin/%d", os.Getenv("APP_HOST"), v.ID)
		log.Printf("QR Code gerado com URL: %s", scanURL)

		os.MkdirAll("qrcodes", os.ModePerm)
		filename := fmt.Sprintf("qrcodes/%d.png", v.ID)
		qrcode.WriteFile(scanURL, qrcode.Medium, 256, filename)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Voluntários de teste criados com sucesso e QR Codes gerados!"})
}

func deleteFakeVolunteers(c *gin.Context) {
	if err := db.Where("name LIKE ?", "Teste_%").Delete(&Volunteer{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao remover voluntários de teste"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Voluntários de teste removidos com sucesso!"})
}
