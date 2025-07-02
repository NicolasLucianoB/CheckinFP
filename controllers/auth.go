package controllers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
