package controllers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicolaslucianob/checkinfp/models"
	"github.com/nicolaslucianob/checkinfp/utils"
	"gorm.io/gorm"
)

func CheckIn(c *gin.Context, db *gorm.DB) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token ausente na requisição"})
		return
	}

	client := utils.NewRedisClient()
	defer client.Close()

	redisKey := fmt.Sprintf("checkinfp:token:%s", token)
	val, err := client.Get(utils.Ctx, redisKey).Result()
	if err != nil || val != "valid" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "QR Code inválido ou expirado"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	// Impede múltiplos check-ins por usuário no mesmo período
	checkUserKey := fmt.Sprintf("checkinfp:user_checkin:%d", userID)
	alreadyChecked, _ := client.Exists(utils.Ctx, checkUserKey).Result()
	if alreadyChecked > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Você já fez o check-in para este culto! 🙌🏽"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		return
	}

	checkKey := fmt.Sprintf("checkinfp:checkin:%d:%s", userID, token)
	success, err := client.SetNX(utils.Ctx, checkKey, "done", 3*time.Hour).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao acessar o cache"})
		return
	}
	if !success {
		c.JSON(http.StatusConflict, gin.H{
			"message": "Você já fez o check-in para este culto! 🙌🏽",
		})
		return
	}

	checkin := models.VolunteerCheckin{
		UserID:      user.ID,
		CheckinTime: time.Now(),
	}
	if err := db.Create(&checkin).Error; err != nil {
		_ = client.Del(utils.Ctx, checkKey).Err()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao registrar check-in"})
		return
	}

	_ = client.Set(utils.Ctx, checkUserKey, "done", 3*time.Hour).Err()

	log.Printf("Check-in realizado com sucesso: %s", user.Name)
	c.JSON(http.StatusOK, gin.H{
		"message": "✅ Check-in realizado com sucesso\nHora de servir com alegria!",
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

func GetLastCheckin(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var lastCheckin models.VolunteerCheckin
	err := db.Where("user_id = ?", userID).
		Order("checkin_time DESC").
		First(&lastCheckin).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"last_checkin": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar último check-in"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"last_checkin": lastCheckin.CheckinTime,
	})
}

func CheckinRanking(c *gin.Context, db *gorm.DB) {
	type Result struct {
		ID            uuid.UUID
		Name          string
		TotalCheckins int
	}

	var results []Result

	err := db.Table("volunteer_checkins").
		Select("users.id, users.name, COUNT(volunteer_checkins.id) as total_checkins").
		Joins("JOIN users ON users.id = volunteer_checkins.user_id").
		Group("users.id, users.name").
		Order("total_checkins DESC").
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate ranking"})
		return
	}

	c.JSON(http.StatusOK, results)
}
