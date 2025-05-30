package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicolaslucianob/checkinfp/models"
	"gorm.io/gorm"
)

func GetMe(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "*Voluntário não encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"name":      user.Name,
		"roles":     user.Roles,
		"email":     user.Email,
		"is_admin":  user.IsAdmin,
		"photo_url": user.PhotoURL,
	})
}

func UpdateProfile(c *gin.Context, db *gorm.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuário não autenticado"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var input struct {
		Name     *string   `json:"name"`
		Email    *string   `json:"email"`
		Password *string   `json:"password"`
		Roles    *[]string `json:"roles"`
		PhotoURL *string   `json:"photo_url"`
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
	if input.Roles != nil {
		user.Roles = *input.Roles
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
