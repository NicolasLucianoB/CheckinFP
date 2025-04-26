package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/nicolaslucianob/checkinfp/models"
)

var jwtKey = []byte("sua_chave_secreta")

func GenerateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func SignUp(c *gin.Context, db *gorm.DB) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}
	if len(strings.Fields(input.Name)) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "*Por favor, informe seu nome completo, varão(oa)"})
		return
	}
	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar senha"})
		return
	}
	user := models.User{Name: input.Name, Email: input.Email, Password: hashedPassword, IsAdmin: false}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criar voluntário"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Voluntário criado com sucesso"})
}

func Login(c *gin.Context, db *gorm.DB) {
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}
	var user models.User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "*Email não encontrado, irmão(ã)"})
		return
	}
	if !CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "*Senha errada, irmão(ã)"})
		return
	}
	// Gerar token incluindo is_admin no payload
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenObj.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"name":     user.Name,
			"role":     user.Role,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func GetMe(c *gin.Context, db *gorm.DB) {
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "*Voluntário não encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"role":     user.Role,
		"email":    user.Email,
		"is_admin": user.IsAdmin,
	})
}

func CreateVolunteer(c *gin.Context, db *gorm.DB) {
	var volunteer models.Volunteer
	if err := c.ShouldBindJSON(&volunteer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	if err := db.Create(&volunteer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Não foi possível cadastrar o voluntário"})
		return
	}

	c.JSON(http.StatusCreated, volunteer)
}

func ListVolunteers(c *gin.Context, db *gorm.DB) {
	var volunteers []models.Volunteer

	name := c.Query("name")
	role := c.Query("role")

	query := db.Model(&models.Volunteer{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if role != "" {
		query = query.Where("role LIKE ?", "%"+role+"%")
	}

	if err := query.Find(&volunteers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar voluntários"})
		return
	}

	c.JSON(http.StatusOK, volunteers)
}

func GetVolunteerByID(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var volunteer models.Volunteer
	if err := db.First(&volunteer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Voluntário não encontrado"})
		return
	}

	var checkins []models.VolunteerCheckin
	if err := db.Where("volunteer_id = ?", volunteer.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar check-ins"})
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

func GenerateQRCode(c *gin.Context) {
	scheme := "https"
	host := os.Getenv("APP_HOST")
	if os.Getenv("ENV") == "development" {
		scheme = "http"
		host = "localhost:8080"
	}
	scanURL := fmt.Sprintf("%s://%s/checkin", scheme, host)
	log.Printf("QR Code gerado com URL: %s", scanURL)

	// Gerar o QR code em memória
	png, err := qrcode.Encode(scanURL, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate QR Code"})
		return
	}

	// Definir headers e retornar a imagem PNG
	c.Header("Content-Type", "image/png")
	c.Writer.Write(png)
}

func CheckIn(c *gin.Context, db *gorm.DB) {
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		return
	}

	checkin := models.VolunteerCheckin{VolunteerID: user.ID}
	if err := db.Create(&checkin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao registrar o check-in"})
		return
	}

	log.Printf("Check-in realizado com sucesso: %s", user.Name)
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("✅ Check-in realizado com sucesso! A paz, %s!", user.Name),
	})
}

func ListCheckins(c *gin.Context, db *gorm.DB) {
	var checkins []models.VolunteerCheckin
	if err := db.Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

func CheckinRanking(c *gin.Context, db *gorm.DB) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate ranking"})
		return
	}

	c.JSON(http.StatusOK, results)
}

func GetVolunteerDashboardData(c *gin.Context, db *gorm.DB) {
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	var volunteer models.Volunteer
	if err := db.First(&volunteer, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Voluntário não encontrado"})
		return
	}

	var checkins []models.VolunteerCheckin
	if err := db.Where("volunteer_id = ?", volunteer.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar check-ins"})
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

	type RankingEntry struct {
		ID            uint
		Name          string
		TotalCheckins int
	}

	var ranking []RankingEntry
	if err := db.Table("volunteer_checkins").
		Select("volunteers.id, volunteers.name, COUNT(volunteer_checkins.id) as total_checkins").
		Joins("JOIN volunteers ON volunteers.id = volunteer_checkins.volunteer_id").
		Group("volunteers.id, volunteers.name").
		Order("total_checkins DESC").
		Scan(&ranking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar ranking"})
		return
	}

	var rankingPosition int
	for i, entry := range ranking {
		if entry.ID == volunteer.ID {
			rankingPosition = i + 1
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  volunteer.ID,
		"name":                volunteer.Name,
		"role":                volunteer.Role,
		"created_at":          volunteer.CreatedAt,
		"total_checkins":      len(checkins),
		"checkins_this_month": checkinsThisMonth,
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"ranking_position":    rankingPosition,
	})
}
