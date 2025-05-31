package controllers

import (
	"net/http"
	"sort"
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

func GetPunctualityRanking(c *gin.Context, db *gorm.DB) {
	period := c.DefaultQuery("period", "monthly")
	scope := c.DefaultQuery("scope", "team")

	type PunctualityEntry struct {
		ID         uuid.UUID
		Name       string
		Checkins   int
		Punctual   int
		Percentage float64
	}

	// Define time filters
	now := time.Now()
	var startDate time.Time
	switch period {
	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "last_event":
		// Para simplificar, considera último domingo como último evento
		offset := int(now.Weekday())
		startDate = now.AddDate(0, 0, -offset)
	case "total":
		startDate = time.Time{} // sem filtro
	default:
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	// Define os horários ideais por dia da semana
	idealTimes := map[time.Weekday][]time.Time{
		time.Sunday: {
			time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
			time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
		},
		time.Monday: {
			time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC),
		},
		time.Tuesday: {
			time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC),
		},
		time.Wednesday: {
			time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC),
		},
		time.Thursday: {
			time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC),
		},
		time.Friday: {
			time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC),
		},
		time.Saturday: {
			time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC),
		},
	}

	var checkins []models.VolunteerCheckin
	query := db.Preload("User").Where("checkin_time >= ?", startDate)
	if scope == "individual" {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
			return
		}
		userID := userIDVal.(uuid.UUID)
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar check-ins"})
		return
	}

	punctualityMap := map[uuid.UUID]*PunctualityEntry{}

	for _, checkin := range checkins {
		userID := checkin.UserID
		if _, ok := punctualityMap[userID]; !ok {
			punctualityMap[userID] = &PunctualityEntry{
				ID:       userID,
				Name:     checkin.User.Name,
				Checkins: 0,
				Punctual: 0,
			}
		}

		p := punctualityMap[userID]
		p.Checkins++

		// Verifica pontualidade
		ideals, ok := idealTimes[checkin.CheckinTime.Weekday()]
		if ok {
			for _, ideal := range ideals {
				scheduled := time.Date(checkin.CheckinTime.Year(), checkin.CheckinTime.Month(), checkin.CheckinTime.Day(),
					ideal.Hour(), ideal.Minute(), 0, 0, checkin.CheckinTime.Location())

				if checkin.CheckinTime.Before(scheduled) || checkin.CheckinTime.Equal(scheduled) {
					p.Punctual++
					break
				}
			}
		}
	}

	var ranking []PunctualityEntry
	for _, p := range punctualityMap {
		if p.Checkins > 0 {
			p.Percentage = (float64(p.Punctual) / float64(p.Checkins)) * 100
		}
		ranking = append(ranking, *p)
	}

	// Ordena por pontualidade
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Percentage > ranking[j].Percentage
	})

	c.JSON(http.StatusOK, gin.H{"ranking": ranking})
}
