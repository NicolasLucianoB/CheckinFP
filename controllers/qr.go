package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nicolaslucianob/checkinfp/utils"
	"github.com/skip2/go-qrcode"
)

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
