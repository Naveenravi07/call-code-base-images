package main

import (
	"fmt"
	"log"

	"main/config"
	"main/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	r := gin.Default()
	setupRoutes(r, cfg)

	fmt.Printf("Server starting on port %s\n", cfg.Port)
	fmt.Printf("Serving files from: %s\n", cfg.CodeDir)
	log.Fatal(r.Run(":" + cfg.Port))
}

func setupRoutes(r *gin.Engine, cfg *config.Config) {
	r.GET("/", func(c *gin.Context) {
		c.String(200, "File Explorer API")
	})

	fileHandler := handlers.NewFileHandler(cfg)
	fileHandler.RegisterRoutes(r)
}
