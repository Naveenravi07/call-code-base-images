package main

import (
	"fmt"
	"log"
	"net/http"

	"main/config"
	"main/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	r := gin.Default()
	r.Use(CORSMiddleware())
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

	folderHandler := handlers.NewFolderHandler(cfg)
	folderHandler.RegisterRoutes(r)

	nodeHandler := handlers.NewNodeHandler(cfg)
	nodeHandler.RegisterRoutes(r)

	terminalHandler := handlers.SetupTerminalHandler(cfg)
	terminalHandler.RegisterRoutes(r)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
