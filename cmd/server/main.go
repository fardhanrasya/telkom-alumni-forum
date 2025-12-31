package main

import (
	"log"
	"os"

	"anoa.com/telkomalumiforum/internal/bootstrap"
	"anoa.com/telkomalumiforum/internal/server"
	"anoa.com/telkomalumiforum/pkg/database"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Connect Database
	db := database.Connect()

	// Initialize Redis
	redisClient, err := database.ConnectRedis()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis, like feature will not work: %v", err)
	}

	// Bootstrap Database (Migrate and Seed)
	if err := bootstrap.Migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	if err := bootstrap.SeedRoles(db); err != nil {
		log.Fatalf("failed to seed roles: %v", err)
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "development" {
		if err := bootstrap.SeedAdminUser(db); err != nil {
			log.Fatalf("failed to seed admin user: %v", err)
		}
		if err := bootstrap.SeedBotUser(db); err != nil {
			log.Fatalf("failed to seed bot user: %v", err)
		}
	}

	// Initialize and Run Server
	srv := server.NewServer(db, redisClient)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
