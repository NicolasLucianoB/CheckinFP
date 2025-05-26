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
	"github.com/nicolaslucianob/checkinfp/models"
	"github.com/nicolaslucianob/checkinfp/utils"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

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
		errMsg := err.Error()
		if strings.Contains(errMsg, "duplicate key") ||
			strings.Contains(errMsg, "UNIQUE constraint failed") ||
			strings.Contains(errMsg, "duplicate entry") ||
			strings.Contains(errMsg, "Error 1062") {
			c.JSON(http.StatusBadRequest, gin.H{"message": "*E-mail já cadastrado, irmão(ã)"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criar conta"})
		}
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
			"id":        user.ID,
			"name":      user.Name,
			"role":      user.Role,
			"email":     user.Email,
			"is_admin":  user.IsAdmin,
			"photo_url": user.PhotoURL,
		},
	})
}

func GetMe(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uint)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "*Voluntário não encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"name":      user.Name,
		"role":      user.Role,
		"email":     user.Email,
		"is_admin":  user.IsAdmin,
		"photo_url": user.PhotoURL,
	})
}

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
	role := c.Query("role")

	query := db.Model(&models.User{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if role != "" {
		query = query.Where("role LIKE ?", "%"+role+"%")
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
	if err := db.Where("volunteer_id = ?", user.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
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
		"role":                user.Role,
		"created_at":          user.CreatedAt,
		"checkins":            checkins,
		"total_checkins":      len(checkins),
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"checkins_this_month": checkinsThisMonth,
	})
}

func GenerateQRCode(c *gin.Context) {
	client := utils.NewRedisClient()

	qrKey := "checkinfp:qr_code_current"

	// 1. Tenta pegar QR Code salvo
	existing, err := client.HGetAll(utils.Ctx, qrKey).Result()
	if err == nil && len(existing) > 0 {
		ttl, _ := client.TTL(utils.Ctx, qrKey).Result()
		if ttl > 0 && existing["url"] != "" && existing["token"] != "" {
			hours := int(ttl.Hours())
			minutes := int(ttl.Minutes()) % 60
			seconds := int(ttl.Seconds()) % 60

			c.JSON(http.StatusOK, gin.H{
				"url":        existing["url"],
				"token":      existing["token"],
				"expires_in": fmt.Sprintf("%02dh:%02dm:%02ds", hours, minutes, seconds),
				"expires_at": time.Now().Add(ttl).UnixMilli(),
			})
			return
		}
	}

	defer client.Close()

	token := utils.GenerateRandomToken()
	// Salva o token no Redis com expiração de 3 horas.
	redisTokenKey := fmt.Sprintf("checkinfp:token:%s", token)
	err = client.Set(utils.Ctx, redisTokenKey, "valid", 3*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao salvar token no cache"})
		return
	}

	host := os.Getenv("FRONT_HOST")
	scheme := "https"
	scanURL := fmt.Sprintf("%s://%s/checkin?token=%s", scheme, host, token)
	log.Printf("QR Code gerado com URL: %s", scanURL)

	filename := fmt.Sprintf("qr-%s.png", token)
	err = qrcode.WriteFile(scanURL, qrcode.Medium, 256, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar QR Code"})
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao abrir QR Code"})
		return
	}
	defer file.Close()

	url, err := utils.UploadToCloudinary(filename, strings.TrimSuffix(filename, ".png"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao enviar QR Code para o Cloudinary"})
		return
	}
	os.Remove(filename)

	_ = client.HSet(utils.Ctx, qrKey, map[string]interface{}{
		"url":   url,
		"token": token,
	}).Err()
	_ = client.Expire(utils.Ctx, qrKey, 3*time.Hour).Err()

	expiry := 3 * time.Hour
	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"token":      token,
		"expires_in": fmt.Sprintf("%02dh:%02dm:%02ds", int(expiry.Hours()), int(expiry.Minutes())%60, int(expiry.Seconds())%60),
		"expires_at": time.Now().Add(expiry).UnixMilli(),
	})
}

func RegenerateQRCode(c *gin.Context) {
	client := utils.NewRedisClient()
	defer client.Close()

	qrKey := "checkinfp:qr_code_current"
	err := client.Del(utils.Ctx, qrKey).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao deletar QR Code do cache"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "QR Code resetado com sucesso"})
}

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
	userID := userIDVal.(uint)

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
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
		return
	}
	userID := userIDVal.(uint)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Usuário não encontrado"})
		return
	}

	var checkins []models.VolunteerCheckin
	if err := db.Where("volunteer_id = ?", user.ID).Order("checkin_time DESC").Find(&checkins).Error; err != nil {
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
		Select("users.id, users.name, COUNT(volunteer_checkins.id) as total_checkins").
		Joins("JOIN users ON users.id = volunteer_checkins.volunteer_id").
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
		"role":                user.Role,
		"created_at":          user.CreatedAt,
		"total_checkins":      len(checkins),
		"checkins_this_month": checkinsThisMonth,
		"first_checkin":       firstCheckin,
		"last_checkin":        lastCheckin,
		"ranking_position":    rankingPosition,
	})
}

func UpdateProfile(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uint)

	var input struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Password *string `json:"password"`
		Role     *string `json:"role"`
		PhotoURL *string `json:"photo_url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Usuário não encontrado"})
		return
	}

	if input.Name != nil {
		if len(strings.Fields(*input.Name)) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "*Por favor, informe seu nome completo, varão(oa)"})
			return
		}
		user.Name = *input.Name
	}
	if input.Email != nil && *input.Email != user.Email {
		var existingUser models.User
		if err := db.Where("email = ?", *input.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "*E-mail já cadastrado, irmão(ã)"})
			return
		}
		user.Email = *input.Email
	}
	if input.Role != nil {
		user.Role = *input.Role
	}
	if input.PhotoURL != nil {
		user.PhotoURL = *input.PhotoURL
	}
	if input.Password != nil {
		hashedPassword, err := HashPassword(*input.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar senha"})
			return
		}
		user.Password = hashedPassword
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao atualizar perfil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Perfil atualizado com sucesso"})
}

func GetLastCheckin(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uint)

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
