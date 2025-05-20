package models

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Email     string `gorm:"not null;unique"`
	Role      string `gorm:"not null"`
	Password  string `gorm:"not null"`
	IsAdmin   bool   `json:"is_admin" gorm:"default:false"`
	PhotoURL  string `json:"photo_url"`
	CreatedAt time.Time
}

type VolunteerCheckin struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null"`
	CheckinTime time.Time `gorm:"autoCreateTime"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func InitDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar com o banco de dados:", err)
	}

	err = database.AutoMigrate(&User{}, &VolunteerCheckin{})
	if err != nil {
		log.Fatal("Erro ao fazer AutoMigrate:", err)
	}
	DB = database
}
