package handlers

import (
	"main/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type NodeHandler struct {
	cfg *config.Config
}

func NewNodeHandler(cfg *config.Config) *NodeHandler {
	return &NodeHandler{cfg: cfg}
}

func (h *NodeHandler) RegisterRoutes(r *gin.Engine) {
	r.DELETE("/api/node", h.DeleteNode)
	r.POST("/api/node/rename", h.RenameNode)
	r.POST("/api/node/move", h.MoveNode)
}

func (h *NodeHandler) DeleteNode(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "path query parameter is required"})
		return
	}

	cleanPath := strings.TrimPrefix(path, "/")
	cleanPath = filepath.Clean(cleanPath)

	if cleanPath == "." || cleanPath == "" {
		c.JSON(400, gin.H{"error": "cannot delete root"})
		return
	}

	absPath := filepath.Join(h.cfg.CodeDir, cleanPath)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "path not found"})
		return
	}

	if err := os.RemoveAll(absPath); err != nil {
		c.JSON(500, gin.H{"error": "failed to delete path"})
		return
	}

	c.JSON(200, gin.H{"message": "deleted successfully", "path": cleanPath})
}

type renameRequest struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
}

func (h *NodeHandler) RenameNode(c *gin.Context) {
	var req renameRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	oldClean := filepath.Clean(strings.TrimPrefix(req.OldPath, "/"))
	newClean := filepath.Clean(strings.TrimPrefix(req.NewPath, "/"))

	oldAbs := filepath.Join(h.cfg.CodeDir, oldClean)
	newAbs := filepath.Join(h.cfg.CodeDir, newClean)

	if _, err := os.Stat(oldAbs); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "old path does not exist"})
		return
	}

	if err := os.MkdirAll(filepath.Dir(newAbs), os.ModePerm); err != nil {
		c.JSON(500, gin.H{"error": "failed to create destination directory"})
		return
	}

	if _, err := os.Stat(newAbs); err == nil {
		c.JSON(409, gin.H{"error": "a file/folder already exists at newPath"})
		return
	}

	if err := os.Rename(oldAbs, newAbs); err != nil {
		c.JSON(500, gin.H{"error": "failed to rename file/folder"})
		return
	}

	c.JSON(200, gin.H{"message": "renamed successfully", "oldPath": oldClean, "newPath": newClean})
}

type moveRequest struct {
	SourcePath string `json:"sourcePath"`
	TargetPath string `json:"targetPath"`
}

func (h *NodeHandler) MoveNode(c *gin.Context) {
	var req moveRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	src := filepath.Clean(strings.TrimPrefix(req.SourcePath, "/"))
	dstDir := filepath.Clean(strings.TrimPrefix(req.TargetPath, "/"))

	srcPath := filepath.Join(h.cfg.CodeDir, src)
	dstBase := filepath.Join(h.cfg.CodeDir, dstDir)

	// Restrict moving parent into child
	if rel, err := filepath.Rel(srcPath, dstBase); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		c.JSON(400, gin.H{"error": "Cannot move a folder into its own subfolder"})
		return
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Source path does not exist"})
		return
	}

	fileName := filepath.Base(srcPath)
	finalDst := filepath.Join(dstBase, fileName)

	if err := os.MkdirAll(dstBase, os.ModePerm); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create target directory"})
		return
	}

	if _, err := os.Stat(finalDst); err == nil {
		c.JSON(409, gin.H{"error": "Target already exists"})
		return
	}

	if err := os.Rename(srcPath, finalDst); err != nil {
		c.JSON(500, gin.H{"error": "Failed to move file/folder"})
		return
	}

	c.JSON(200, gin.H{"message": "Moved successfully", "from": req.SourcePath, "to": filepath.Join("/", dstDir, fileName)})
}

