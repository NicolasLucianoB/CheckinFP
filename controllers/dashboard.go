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
	sortBy := c.DefaultQuery("sort_by", "punctuality")

	type PunctualityEntry struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		PhotoURL   string    `json:"avatar_url"`
		Checkins   int       `json:"checkins"`
		Punctual   int       `json:"punctual"`
		Percentage float64   `json:"percentage"`
	}

	now := time.Now()
	var startDate time.Time
	switch period {
	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "last_event":
		var lastCheckin models.VolunteerCheckin
		if err := db.Order("checkin_time DESC").First(&lastCheckin).Error; err == nil {
			startDate = lastCheckin.CheckinTime.Truncate(24 * time.Hour)
		} else {
			startDate = now.AddDate(0, 0, -7)
		}
	case "total":
		startDate = time.Time{}
	default:
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	idealTimes := map[time.Weekday][]time.Time{
		time.Sunday:    {time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC)},
		time.Monday:    {time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)},
		time.Tuesday:   {time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)},
		time.Wednesday: {time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)},
		time.Thursday:  {time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)},
		time.Friday:    {time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)},
		time.Saturday:  {time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)},
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

	if len(checkins) == 0 {
		c.JSON(http.StatusOK, gin.H{"ranking": []PunctualityEntry{}})
		return
	}

	punctualityMap := make(map[uuid.UUID]*PunctualityEntry)

	for _, checkin := range checkins {
		userID := checkin.UserID
		if _, ok := punctualityMap[userID]; !ok {
			punctualityMap[userID] = &PunctualityEntry{
				ID:       userID,
				Name:     checkin.User.Name,
				PhotoURL: checkin.User.PhotoURL,
				Checkins: 0,
				Punctual: 0,
			}
		}
		entry := punctualityMap[userID]
		entry.Checkins++

		ideals, ok := idealTimes[checkin.CheckinTime.Weekday()]
		if ok {
			for _, ideal := range ideals {
				scheduled := time.Date(checkin.CheckinTime.Year(), checkin.CheckinTime.Month(), checkin.CheckinTime.Day(),
					ideal.Hour(), ideal.Minute(), 0, 0, checkin.CheckinTime.Location())

				diff := scheduled.Sub(checkin.CheckinTime)

				switch {
				case diff >= 45*time.Minute:
					entry.Punctual++ // Chegou antes de 45min do culto, é pontual
				case diff >= 35*time.Minute:
					// Levemente atrasado — pode ser tratado depois
				case diff >= 30*time.Minute:
					// Atrasado — pode ser tratado depois
				case diff < 30*time.Minute:
					// Muito atrasado — pode ser tratado depois
				}

				break // Considera apenas o primeiro horário ideal possível no dia
			}
		}
	}

	var ranking []PunctualityEntry
	for _, entry := range punctualityMap {
		if entry.Checkins > 0 {
			entry.Percentage = (float64(entry.Punctual) / float64(entry.Checkins)) * 100
			ranking = append(ranking, *entry)
		}
	}

	switch sortBy {
	case "punctuality":
		sort.Slice(ranking, func(i, j int) bool {
			return ranking[i].Percentage > ranking[j].Percentage
		})
	case "attendance":
		sort.Slice(ranking, func(i, j int) bool {
			return ranking[i].Checkins > ranking[j].Checkins
		})
	}

	c.JSON(http.StatusOK, gin.H{"ranking": ranking})
}
