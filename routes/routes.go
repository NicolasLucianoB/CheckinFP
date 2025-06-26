package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nicolaslucianob/checkinfp/controllers"
	"github.com/nicolaslucianob/checkinfp/middlewares"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	// Public Routes
	r.POST("/signup", func(c *gin.Context) { controllers.SignUp(c, db) })
	r.POST("/login", func(c *gin.Context) { controllers.Login(c, db) })

	// r.GET("/generate/qr", controllers.GenerateQRCode) // Public route for QR code generation

	// Protected Routes
	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware())

	// QR Code
	auth.GET("/generate/qr", controllers.GenerateQRCode)
	auth.POST("/generate/qr/reset", controllers.RegenerateQRCode)

	// Authenticated User Info
	auth.GET("/me", func(c *gin.Context) { controllers.GetMe(c, db) })
	auth.PUT("/me", func(c *gin.Context) { controllers.UpdateProfile(c, db) })

	// Check-in
	auth.POST("/checkin", func(c *gin.Context) { controllers.CheckIn(c, db) })
	auth.GET("/checkins", func(c *gin.Context) { controllers.ListCheckins(c, db) })
	auth.GET("/checkin/last", func(c *gin.Context) { controllers.GetLastCheckin(c, db) })
	auth.GET("/ranking", func(c *gin.Context) { controllers.CheckinRanking(c, db) })

	// Volunteers
	auth.POST("/volunteers", func(c *gin.Context) { controllers.CreateVolunteer(c, db) })
	auth.GET("/volunteers", func(c *gin.Context) { controllers.ListVolunteers(c, db) })
	auth.GET("/volunteers/:id", func(c *gin.Context) { controllers.GetVolunteerByID(c, db) })

	// Dashboard
	auth.GET("/dashboard", func(c *gin.Context) { controllers.GetVolunteerDashboardData(c, db) })
	auth.GET("/dashboard/punctuality-ranking", func(c *gin.Context) { controllers.GetPunctualityRanking(c, db) })
	auth.GET("/dashboard/roles-distribution", func(c *gin.Context) { controllers.GetRolesDistribution(c, db) })
	auth.GET("/dashboard/punctuality-meter", func(c *gin.Context) { controllers.GetPunctualityMeter(c, db) })
}
