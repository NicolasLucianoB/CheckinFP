package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"crypto/rand"
	"encoding/hex"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var JwtKey = []byte(os.Getenv("JWT_SECRET"))

func GenerateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 3).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func InitCloudinary() (*cloudinary.Cloudinary, error) {
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return nil, err
	}
	return cld, nil
}

func UploadToCloudinary(filePath string, filename string) (string, error) {
	cld, err := InitCloudinary()
	if err != nil {
		return "", fmt.Errorf("erro ao inicializar Cloudinary: %v", err)
	}

	ctx := context.Background()
	uploadParams := uploader.UploadParams{
		PublicID: filename,
	}

	log.Println("Tentando enviar arquivo:", filePath)

	uploadResult, err := cld.Upload.Upload(ctx, filePath, uploadParams)
	if err != nil {
		// return "", fmt.Errorf("erro ao fazer upload para Cloudinary: %v", err)
		return "", fmt.Errorf("erro no upload para Cloudinary:\nArquivo: %s\nErro: %v", filePath, err)
	}

	log.Println("Upload concluído. URL:", uploadResult.SecureURL)
	return uploadResult.SecureURL, nil
}

var Ctx = context.Background()

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASS"),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})
}

func GenerateRandomToken() string {
	bytes := make([]byte, 16) // 128 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
