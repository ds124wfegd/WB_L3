package transport

import (
	"github.com/ds124wfegd/WB_L3/5/internal/transport/middleware"
	"github.com/gin-gonic/gin"
)

func InitRoutes(eventHandler *EventHandler, bookingHandler *BookingHandler, userHandler *UserHandler) *gin.Engine {

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.Static("/static", "/app/internal/web/templates")
	router.LoadHTMLGlob("/app/internal/web/templates/*.html")

	router.GET("/", func(c *gin.Context) {
		c.File("/app/internal/web/templates/index.html")
	})

	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())
	router.Use(middleware.Timeout(30))

	// API routes
	api := router.Group("/api/v1")
	{
		// Event routes
		events := api.Group("/events")
		{
			events.POST("", eventHandler.CreateEvent)
			events.GET("", eventHandler.GetAllEvents)
			events.GET("/:id", eventHandler.GetEvent)
		}

		// Booking routes
		bookings := api.Group("/bookings")
		{
			bookings.POST("/events/:id/book", bookingHandler.BookSeats)
			bookings.POST("/events/:id/confirm", bookingHandler.ConfirmBooking)
			bookings.GET("/users/:user_id", bookingHandler.GetUserBookings)
		}

		// User routes
		users := api.Group("/users")
		{
			users.POST("/register", userHandler.RegisterUser)
			users.GET("/:id", userHandler.GetUser)
			users.POST("/:id/telegram", userHandler.LinkTelegram)
		}

		// Admin routes
		admin := api.Group("/admin")
		{
			admin.GET("/bookings", bookingHandler.GetAllBookings)
			admin.GET("/events/:id/bookings", bookingHandler.GetEventBookings)
			admin.DELETE("/bookings/:id", bookingHandler.CancelBooking)
		}
	}

	// Web interface routes
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "user.html", nil)
	})

	router.GET("/admin", func(c *gin.Context) {
		c.HTML(200, "admin.html", nil)
	})

	router.GET("/event/:id", func(c *gin.Context) {
		c.HTML(200, "event.html", gin.H{
			"eventID": c.Param("id"),
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": gin.H{"time": "server is running"},
		})
	})

	return router
}
