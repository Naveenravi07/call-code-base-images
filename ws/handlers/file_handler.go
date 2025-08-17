package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"main/config"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type FileNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Path     string      `json:"path"`
	Children []*FileNode `json:"children,omitempty"`
	Language string      `json:"language,omitempty"`
}

type FileHandler struct {
	cfg *config.Config
}

func NewFileHandler(cfg *config.Config) *FileHandler {
	return &FileHandler{cfg: cfg}
}

func (h *FileHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/files", h.ListFiles)
	r.GET("/api/files/content",h.GetFileContent)
	r.POST("/api/files/content", h.SaveFileContent)
}

func (h *FileHandler) ListFiles(c *gin.Context) {
	rootNode, err := buildFileTree(h.cfg.CodeDir, "")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, []*FileNode{rootNode})
}

func (h *FileHandler) GetFileContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "path query parameter is required"})
		return
	}

	content, err := getFileContent(path)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.String(200, content)
}

type saveFileRequest struct {
    Content string `json:"content"`
}

func (h *FileHandler) SaveFileContent(c *gin.Context) {
    path := c.Query("path")
    if path == "" {
        c.JSON(400, gin.H{"error": "path query parameter is required"})
        return
    }

    var req saveFileRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request body"})
        return
    }

    absPath := filepath.Join(h.cfg.CodeDir, path)
    if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.Status(200)
}


func buildFileTree(absPath string, relPath string) (*FileNode, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	node := &FileNode{
		ID:   genID(absPath),
		Name: info.Name(),
		Type: "folder",
		Path: "/" + relPath,
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			excluded := []string{"__pycache__", "node_modules", ".git", ".github", ".vscode", ".idea"}
			if slices.Contains(excluded, entry.Name()) {
				continue
			}
			child, err := buildFileTree(
				filepath.Join(absPath, entry.Name()),
				filepath.Join(relPath, entry.Name()),
			)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		} else {
			lang := detectLanguage(entry.Name())
			child := &FileNode{
				ID:       genID(filepath.Join(absPath, entry.Name())),
				Name:     entry.Name(),
				Type:     "file",
				Path:     "/" + filepath.Join(relPath, entry.Name()),
				Language: lang,
			}
			node.Children = append(node.Children, child)
		}
	}

	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.Type != b.Type {
			return a.Type == "folder" 
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	return node, nil
}


func genID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])[:8]
}

func getFileContent(path string) (string, error) {
	var absPath string = config.LoadConfig().CodeDir
	data, err := os.ReadFile(absPath + path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}


func detectLanguage(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".css":
		return "css"
	case ".html":
		return "html"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}

