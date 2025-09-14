package handlers

import (
	"main/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type FolderHandler struct {
	cfg *config.Config
}

func NewFolderHandler(cfg *config.Config) *FolderHandler {
	return &FolderHandler{cfg: cfg}
}

func (h *FolderHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/folder/create", h.NewFolderCreate)
}

type newFolderRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (h *FolderHandler) NewFolderCreate(c *gin.Context) {
	var req newFolderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(req.Path, "/"))
	absPath := filepath.Join(h.cfg.CodeDir, cleanPath)

	if _, err := os.Stat(absPath); err == nil {
		c.JSON(409, gin.H{"error": "folder already exists"})
		return
	}

	if err := os.MkdirAll(absPath, os.ModePerm); err != nil {
		c.JSON(500, gin.H{"error": "failed to create folder"})
		return
	}

	c.JSON(201, gin.H{"message": "folder created successfully", "folderName": req.Name, "folderPath": cleanPath})
}

