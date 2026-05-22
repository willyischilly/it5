package main

import (
	"log"
	"os"

	"planner-backend/internal/app"
	"planner-backend/internal/config"
	"planner-backend/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	r, _ := app.NewRouter(cfg, db)
	log.Printf("server http://localhost:%s  health=/health  swagger=/swagger  openapi=/api/openapi.yaml", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
