package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicolaslucianob/checkinfp/models"
	"gorm.io/gorm"
)

func GetVolunteerDashboardData(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Usuário não encontrado"})
		return
	}

	var checkins []models.VolunteerCheckin
	if err := db.Where("user_id = ?", user.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
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
		ID            uuid.UUID
		Name          string
		TotalCheckins int
	}

	var ranking []RankingEntry
	if err := db.Table("volunteer_checkins").
		Select("users.id, users.name, COUNT(volunteer_checkins.id) as total_checkins").
		Joins("JOIN users ON users.id = volunteer_checkins.user_id").
		Group("users.id, users.name").
		Order("total_checkins DESC").
		Scan(&ranking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar ranking"})
		return
	}

	var rankingPosition int
	for i, entry := range ranking {
		if entry.ID == user.ID {
			rankingPosition = i + 1
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  user.ID,
		"name":                user.Name,
		"roles":               user.Roles,
		"created_at":          user.CreatedAt,
		"total_checkins":      len(checkins),
		"checkins_this_month": checkinsThisMonth,
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"ranking_position":    rankingPosition,
	})
}
