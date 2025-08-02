package handlers

import (
	"main/config"
	"os"

	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

type FileHandler struct {
	cfg *config.Config
}

func NewFileHandler(cfg *config.Config) *FileHandler {
	return &FileHandler{cfg: cfg}
}

func (h *FileHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/files/*path", h.ListFiles)
}

func (h *FileHandler) ListFiles(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		path = h.cfg.CodeDir
	} else {
		path = h.cfg.CodeDir + "/" + path
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		fileType := 0 // Unknown
		if entry.IsDir() {
			fileType = 2 // Directory
		} else {
			fileType = 1 // File
		}

		files = append(files, FileInfo{
			Name: entry.Name(),
			Type: fileType,
		})
	}

	c.JSON(200, files)
}
