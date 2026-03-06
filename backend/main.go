package main

import (
	"log"
	"net/http"
	"os"

	"github.com/RomanWilkening/budget_projection/backend/database"
	"github.com/RomanWilkening/budget_projection/backend/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	database.Init()

	// Set up Gin router
	router := setupRouter()

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	// API routes
	api := router.Group("/api")
	{
		// Persons
		persons := api.Group("/persons")
		{
			persons.GET("", handlers.ListPersons)
			persons.GET("/:id", handlers.GetPerson)
			persons.POST("", handlers.CreatePerson)
			persons.PUT("/:id", handlers.UpdatePerson)
			persons.DELETE("/:id", handlers.DeletePerson)
		}

		// Accounts
		accounts := api.Group("/accounts")
		{
			accounts.GET("", handlers.ListAccounts)
			accounts.GET("/:id", handlers.GetAccount)
			accounts.POST("", handlers.CreateAccount)
			accounts.PUT("/:id", handlers.UpdateAccount)
			accounts.DELETE("/:id", handlers.DeleteAccount)
			accounts.POST("/:id/owners", handlers.AddAccountOwner)
			accounts.DELETE("/:id/owners/:personId", handlers.RemoveAccountOwner)
		}

		// Positions
		positions := api.Group("/positions")
		{
			positions.GET("", handlers.ListPositions)
			positions.GET("/:id", handlers.GetPosition)
			positions.POST("", handlers.CreatePosition)
			positions.PUT("/:id", handlers.UpdatePosition)
			positions.DELETE("/:id", handlers.DeletePosition)
		}

		// Projection
		api.GET("/projection", handlers.GetProjection)
	}

	// Serve frontend static files
	// In production, the built React app is placed in ./frontend/dist
	if _, err := os.Stat("frontend/dist"); err == nil {
		router.Static("/assets", "frontend/dist/assets")
		router.StaticFile("/vite.svg", "frontend/dist/vite.svg")

		// Serve index.html for all non-API, non-asset routes (SPA routing)
		router.NoRoute(func(c *gin.Context) {
			c.File("frontend/dist/index.html")
		})
	} else {
		router.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Budget Projection API is running. Frontend not built yet.",
			})
		})
	}

	return router
}
