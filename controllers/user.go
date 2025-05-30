package controllers

import (
	"fmt"
	"net/http"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/nicolaslucianob/checkinfp/models"
	"gorm.io/gorm"
)

func CreateVolunteer(c *gin.Context, db *gorm.DB) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Não foi possível cadastrar o usuário"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func ListVolunteers(c *gin.Context, db *gorm.DB) {
	var users []models.User

	name := c.Query("name")
	roles := c.Query("roles")

	query := db.Model(&models.User{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if roles != "" {
		query = query.Where("roles @> ?", fmt.Sprintf(`["%s"]`, roles))
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar usuários"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func GetVolunteerByID(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"id":                  user.ID,
		"name":                user.Name,
		"roles":               user.Roles,
		"created_at":          user.CreatedAt,
		"checkins":            checkins,
		"total_checkins":      len(checkins),
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"checkins_this_month": checkinsThisMonth,
	})
}
