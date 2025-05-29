package models

import (
	"fmt"
	"log"
	"os"
	"time"

	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// RolesArray é um tipo customizado para armazenar arrays de roles como JSON no banco de dados.
type RolesArray []string

func (r *RolesArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("falha ao converter valor para []byte")
	}
	return json.Unmarshal(bytes, r)
}

func (r RolesArray) Value() (driver.Value, error) {
	return json.Marshal(r)
}

type User struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name      string     `gorm:"not null"`
	Email     string     `gorm:"not null;unique"`
	Roles     RolesArray `gorm:"type:json"`
	Password  string     `gorm:"not null"`
	IsAdmin   bool       `json:"is_admin" gorm:"default:false"`
	PhotoURL  string     `json:"photo_url"`
	CreatedAt time.Time
}

type VolunteerCheckin struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uuid.UUID `gorm:"not null"`
	CheckinTime time.Time `gorm:"autoCreateTime"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func InitDB() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Sao_Paulo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar com o banco de dados:", err)
	}

	err = database.AutoMigrate(&User{}, &VolunteerCheckin{})
	if err != nil {
		log.Fatal("Erro ao fazer AutoMigrate:", err)
	}
	DB = database
}
