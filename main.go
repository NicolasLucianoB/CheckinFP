package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "checkin.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	createTable()

	r := gin.Default()
	r.GET("/generate/:name", generateQRCode)
	r.GET("/checkin/:name", checkIn)

	// r.Run(":8080") localhost
	r.Run("0.0.0.0:8080") // rede local
}

func createTable() {
	query := `CREATE TABLE IF NOT EXISTS checkins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		arrival_time DATETIME NOT NULL
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func generateQRCode(c *gin.Context) {
	name := c.Param("name")

	// serverIP := "" // Troque pelo seu IP local
	// scanURL := fmt.Sprintf("http://%s:8080/checkin/%s", serverIP, name)
	scanURL := fmt.Sprintf("http://localhost:8080/checkin/%s", name) //localhost

	os.MkdirAll("qrcodes", os.ModePerm)
	filename := fmt.Sprintf("qrcodes/%s.png", name)

	err := qrcode.WriteFile(scanURL, qrcode.Medium, 256, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR Code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "QR Code generated",
		"file":    filename,
		"url":     scanURL,
	})
}

func checkIn(c *gin.Context) {
	name := c.Param("name")
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	_, err := db.Exec("INSERT INTO checkins (name, arrival_time) VALUES (?, ?)", name, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check in"})
		return
	}

	log.Printf("Check-in realizado com sucesso: %s às %s", name, timestamp)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Check-in successful",
		"name":         name,
		"arrival_time": timestamp,
	})
}
