package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nicolaslucianob/checkinfp/controllers"
	"github.com/nicolaslucianob/checkinfp/middlewares"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware())

	auth.GET("/generate/qr", func(c *gin.Context) {
		controllers.GenerateQRCode(c)
	})
	auth.GET("/me", func(c *gin.Context) {
		controllers.GetMe(c, db)
	})

	auth.POST("/checkin", func(c *gin.Context) {
		controllers.CheckIn(c, db)
	})
	auth.GET("/checkins", func(c *gin.Context) {
		controllers.ListCheckins(c, db)
	})
	auth.GET("/ranking", func(c *gin.Context) {
		controllers.CheckinRanking(c, db)
	})

	auth.POST("/volunteers", func(c *gin.Context) {
		controllers.CreateVolunteer(c, db)
	})
	auth.GET("/volunteers", func(c *gin.Context) {
		controllers.ListVolunteers(c, db)
	})
	auth.GET("/volunteers/:id", func(c *gin.Context) {
		controllers.GetVolunteerByID(c, db)
	})

	r.POST("/signup", func(c *gin.Context) {
		controllers.SignUp(c, db)
	})
	r.POST("/login", func(c *gin.Context) {
		controllers.Login(c, db)
	})
}
