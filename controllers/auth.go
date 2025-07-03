package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nicolaslucianob/checkinfp/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func GenerateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 3).Unix(),
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
	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: hashedPassword,
		Roles:    input.Roles,
		IsAdmin:  false,
	}
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
	var expiration time.Time
	if user.IsAdmin {
		expiration = time.Now().AddDate(0, 1, 0) // 1 mês
	} else {
		expiration = time.Now().Add(time.Hour * 3) // 3 horas
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"is_admin": user.IsAdmin,
		"exp":      expiration.Unix(),
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
			"roles":     user.Roles,
			"email":     user.Email,
			"is_admin":  user.IsAdmin,
			"photo_url": user.PhotoURL,
		},
	})
}

func generateResetToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func sendResetEmail(email string, token string) error {
	brevoAPIKey := os.Getenv("BREVO_API_KEY")
	brevoSenderName := os.Getenv("BREVO_NAME")
	brevoSenderEmail := os.Getenv("BREVO_SENDER_EMAIL")

	if brevoAPIKey == "" {
		return fmt.Errorf("BREVO_API_KEY não configurada")
	}

	body := map[string]interface{}{
		"sender": map[string]string{
			"name":  brevoSenderName,
			"email": brevoSenderEmail,
		},
		"to": []map[string]string{
			{"email": email},
		},
		"subject": "Recuperação de Senha - CheckinFP",
		"htmlContent": fmt.Sprintf(`
			<p>Olá, irmão!(ã)</p>
			<p>O atribulado esqueceu a senha e solicitou uma redefinição? Clique no botão abaixo:</p>
			<p><a href="https://checkin-fp.vercel.app/reset-password?token=%s">Redefinir senha</a></p>
			<p>Vigia, esse link expira em 15 minutos.</p>
		`, token),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", brevoAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("erro ao enviar e-mail: status %d, resposta: %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

func ForgotPassword(c *gin.Context, db *gorm.DB) {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil || input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email inválido, irmão(ã)"})
		return
	}

	var user models.User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Enviaremos um link para redefinir sua senha, irmão(ã)."})
		return
	}

	token, err := generateResetToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar token"})
		return
	}

	if err := sendResetEmail(user.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao enviar e-mail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Enviaremos um link para redefinir sua senha, irmão(ã)."})
}

func ResetPassword(c *gin.Context, db *gorm.DB) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil || input.Token == "" || input.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	token, err := jwt.Parse(input.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inválido")
		}
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token inválido ou expirado, irmão(ã)"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["user_id"] == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Oremos...Token inválido"})
		return
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID do voluntário inválido"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Erro ao processar ID do voluntário"})
		return
	}

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Voluntário não encontrado"})
		return
	}

	hashedPassword, err := HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar nova senha"})
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao salvar nova senha"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Senha redefinida com sucesso, irmão(ã)"})
}
