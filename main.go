package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
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
	VolunteerID uint      `gorm:"not null"`
	CheckinTime time.Time `gorm:"autoCreateTime"`
}

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Email     string `gorm:"not null;unique"`
	Password  string `gorm:"not null"`
	CreatedAt time.Time
}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

var jwtKey = []byte("sua_chave_secreta")

func generateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func signUp(c *gin.Context) {
	var input User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}
	hashedPassword, err := hashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criptografar senha"})
		return
	}
	user := User{Name: input.Name, Email: input.Email, Password: hashedPassword}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar voluntário"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Voluntário criado com sucesso"})
}

func login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}
	var user User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Voluntário não encontrado"})
		return
	}
	if !checkPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Senha incorreta"})
		return
	}
	token, err := generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db = initDB()
	log.Println("Banco conectado com sucesso:", db)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	auth := r.Group("/")
	auth.Use(AuthMiddleware())

	auth.GET("/generate/:id", generateQRCode)
	auth.GET("/checkin/:id", checkIn)
	auth.GET("/checkins", listCheckins)
	auth.GET("/ranking", checkinRanking)
	auth.POST("/volunteers", createVolunteer)
	auth.GET("/volunteers", listVolunteers)
	auth.GET("/volunteers/:id", getVolunteerByID)
	r.POST("/signup", signUp)
	r.POST("/login", login)

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

	db.AutoMigrate(&User{}, &Volunteer{}, &VolunteerCheckin{})
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

	checkin := VolunteerCheckin{VolunteerID: volunteer.ID}
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
				<p>A paz, <strong>%s</strong>! Vamos servir com alegria.</p>
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
		ID            uint
		Name          string
		TotalCheckins int
	}

	var results []Result

	err := db.Table("volunteer_checkins").
		Select("volunteers.id, volunteers.name, COUNT(volunteer_checkins.id) as total_checkins").
		Joins("JOIN volunteers ON volunteers.id = volunteer_checkins.volunteer_id").
		Group("volunteers.id, volunteers.name").
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

func getVolunteerByID(c *gin.Context) {
	id := c.Param("id")

	var volunteer Volunteer
	if err := db.First(&volunteer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Voluntário não encontrado"})
		return
	}

	var checkins []VolunteerCheckin
	if err := db.Where("volunteer_id = ?", volunteer.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar check-ins"})
		return
	}

	var firstCheckin, lastCheckin *time.Time
	var checkinsThisMonth int

	if len(checkins) > 0 {
		first := checkins[len(checkins)-1].CheckinTime
		last := checkins[0].CheckinTime
		firstCheckin = &first
		lastCheckin = &last

		currentYear, currentMonth, _ := time.Now().Date()
		for _, ci := range checkins {
			y, m, _ := ci.CheckinTime.Date()
			if y == currentYear && m == currentMonth {
				checkinsThisMonth++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  volunteer.ID,
		"name":                volunteer.Name,
		"role":                volunteer.Role,
		"created_at":          volunteer.CreatedAt,
		"checkins":            checkins,
		"total_checkins":      len(checkins),
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"checkins_this_month": checkinsThisMonth,
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido ou mal formatado"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		c.Next()
	}
}
