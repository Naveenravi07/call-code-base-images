package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type     string `json:"type"`
	FileName string `json:"filename"`
	Content  string `json:"content"`
}

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDirectory bool      `json:"isDirectory"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"modTime"`
	Extension   string    `json:"extension,omitempty"`
	IsHidden    bool      `json:"isHidden"`
	IsReadable  bool      `json:"isReadable"`
	IsWritable  bool      `json:"isWritable"`
}

type DirectoryListing struct {
	Path        string     `json:"path"`
	Files       []FileInfo `json:"files"`
	TotalFiles  int        `json:"totalFiles"`
	TotalDirs   int        `json:"totalDirs"`
	ParentPath  string     `json:"parentPath,omitempty"`
	IsRecursive bool       `json:"isRecursive"`
}

type ProjectFiles struct {
	Files       []FileInfo `json:"files"`
	TotalFiles  int        `json:"totalFiles"`
	TotalDirs   int        `json:"totalDirs"`
	ProjectPath string     `json:"projectPath"`
	LoadedAt    time.Time  `json:"loadedAt"`
}

type FileContent struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
	IsText   bool      `json:"isText"`
	MimeType string    `json:"mimeType"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func isTextFile(filename string) bool {
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".html": true, ".css": true,
		".py": true, ".go": true, ".java": true, ".cpp": true, ".c": true, ".h": true,
		".rb": true, ".php": true, ".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".sql": true, ".csv": true, ".log": true, ".conf": true, ".config": true, ".ini": true,
		".toml": true, ".dockerfile": true, ".gitignore": true, ".env": true, ".properties": true,
		".rs": true, ".swift": true, ".kt": true, ".scala": true, ".clj": true, ".hs": true,
		".lua": true, ".perl": true, ".r": true, ".dart": true, ".elm": true, ".ex": true,
		".vue": true, ".svelte": true, ".astro": true, ".prisma": true, ".graphql": true,
		".proto": true, ".thrift": true, ".avro": true, ".pug": true, ".jade": true,
		".less": true, ".scss": true, ".sass": true, ".styl": true, ".stylus": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return textExtensions[ext] || ext == ""
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".html":       "text/html",
		".css":        "text/css",
		".js":         "application/javascript",
		".json":       "application/json",
		".xml":        "application/xml",
		".txt":        "text/plain",
		".md":         "text/markdown",
		".py":         "text/x-python",
		".go":         "text/x-go",
		".java":       "text/x-java-source",
		".cpp":        "text/x-c++src",
		".c":          "text/x-csrc",
		".h":          "text/x-chdr",
		".rb":         "text/x-ruby",
		".php":        "text/x-php",
		".sh":         "text/x-shellscript",
		".sql":        "text/x-sql",
		".yaml":       "text/x-yaml",
		".yml":        "text/x-yaml",
		".toml":       "text/x-toml",
		".ini":        "text/x-ini",
		".conf":       "text/x-config",
		".log":        "text/x-log",
		".csv":        "text/csv",
		".ts":         "text/typescript",
		".tsx":        "text/typescript",
		".jsx":        "text/javascript",
		".vue":        "text/x-vue",
		".rs":         "text/x-rustsrc",
		".swift":      "text/x-swift",
		".kt":         "text/x-kotlin",
		".scala":      "text/x-scala",
		".dart":       "text/x-dart",
		".lua":        "text/x-lua",
		".r":          "text/x-r",
		".dockerfile": "text/x-dockerfile",
		".gitignore":  "text/x-gitignore",
		".env":        "text/x-env",
	}

	if mime, exists := mimeTypes[ext]; exists {
		return mime
	}
	return "application/octet-stream"
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func checkPermissions(path string) (readable, writable bool) {
	if file, err := os.Open(path); err == nil {
		file.Close()
		readable = true
	}

	if file, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		file.Close()
		writable = true
	} else if os.IsPermission(err) {
		writable = false
	} else {
		parentDir := filepath.Dir(path)
		if stat, err := os.Stat(parentDir); err == nil && stat.IsDir() {
			if file, err := os.Create(path + ".tmp"); err == nil {
				file.Close()
				os.Remove(path + ".tmp")
				writable = true
			}
		}
	}

	return readable, writable
}

func customFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		enableCors(&w)

		recursive := r.URL.Query().Get("recursive") == "true"

		requestPath := filepath.Clean(r.URL.Path)
		if requestPath == "." {
			requestPath = ""
		}

		fullPath := filepath.Join(dir, requestPath)

		// Security check - ensure path is within the allowed directory
		if !strings.HasPrefix(fullPath, filepath.Clean(dir)) {
			sendErrorResponse(w, "Access denied", http.StatusForbidden, "Path outside allowed directory")
			return
		}

		stat, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				sendErrorResponse(w, "File or directory not found", http.StatusNotFound, err.Error())
			} else {
				sendErrorResponse(w, "Unable to access path", http.StatusInternalServerError, err.Error())
			}
			return
		}

		if stat.IsDir() {
			if recursive {
				handleRecursiveFileListing(w, r, fullPath, requestPath)
			} else {
				handleDirectoryListing(w, r, fullPath, requestPath)
			}
		} else {
			handleFileContent(w, r, fullPath, requestPath)
		}
	}
}

func handleRecursiveFileListing(w http.ResponseWriter, r *http.Request, fullPath, requestPath string) {
	var allFiles []FileInfo
	totalFiles := 0
	totalDirs := 0

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
			return nil // Continue walking, don't stop on individual file errors
		}

		// Skip hidden files and directories (optional - you can remove this)
		if isHidden(info.Name()) && path != fullPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(fullPath, path)
		if err != nil {
			log.Printf("Error calculating relative path: %v", err)
			return nil
		}

		if relPath == "." {
			return nil
		}

		webPath := filepath.ToSlash(relPath)
		if requestPath != "" {
			webPath = filepath.ToSlash(filepath.Join(requestPath, relPath))
		}

		readable, writable := checkPermissions(path)

		fileInfo := FileInfo{
			Name:        info.Name(),
			Path:        "/" + webPath,
			IsDirectory: info.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			IsHidden:    isHidden(info.Name()),
			IsReadable:  readable,
			IsWritable:  writable,
		}

		if !info.IsDir() {
			fileInfo.Extension = filepath.Ext(info.Name())
			totalFiles++
		} else {
			totalDirs++
		}

		allFiles = append(allFiles, fileInfo)
		return nil
	})

	if err != nil {
		sendErrorResponse(w, "Unable to walk directory", http.StatusInternalServerError, err.Error())
		return
	}

	sort.Slice(allFiles, func(i, j int) bool {
		if allFiles[i].IsDirectory != allFiles[j].IsDirectory {
			return allFiles[i].IsDirectory
		}
		return strings.ToLower(allFiles[i].Path) < strings.ToLower(allFiles[j].Path)
	})

	response := ProjectFiles{
		Files:       allFiles,
		TotalFiles:  totalFiles,
		TotalDirs:   totalDirs,
		ProjectPath: "/" + filepath.ToSlash(requestPath),
		LoadedAt:    time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

func handleDirectoryListing(w http.ResponseWriter, r *http.Request, fullPath, requestPath string) {
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		sendErrorResponse(w, "Unable to read directory", http.StatusInternalServerError, err.Error())
		return
	}

	var fileInfos []FileInfo
	totalFiles := 0
	totalDirs := 0

	for _, file := range files {
		filePath := filepath.Join(fullPath, file.Name())
		relativePath := filepath.Join(requestPath, file.Name())
		if requestPath == "" {
			relativePath = file.Name()
		}

		readable, writable := checkPermissions(filePath)

		fileInfo := FileInfo{
			Name:        file.Name(),
			Path:        "/" + filepath.ToSlash(relativePath),
			IsDirectory: file.IsDir(),
			Size:        file.Size(),
			ModTime:     file.ModTime(),
			IsHidden:    isHidden(file.Name()),
			IsReadable:  readable,
			IsWritable:  writable,
		}

		if !file.IsDir() {
			fileInfo.Extension = filepath.Ext(file.Name())
			totalFiles++
		} else {
			totalDirs++
		}

		fileInfos = append(fileInfos, fileInfo)
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		if fileInfos[i].IsDirectory != fileInfos[j].IsDirectory {
			return fileInfos[i].IsDirectory
		}
		return strings.ToLower(fileInfos[i].Name) < strings.ToLower(fileInfos[j].Name)
	})

	parentPath := ""
	if requestPath != "" {
		parentPath = "/" + filepath.ToSlash(filepath.Dir(requestPath))
		if parentPath == "/." {
			parentPath = "/"
		}
	}

	response := DirectoryListing{
		Path:        "/" + filepath.ToSlash(requestPath),
		Files:       fileInfos,
		TotalFiles:  totalFiles,
		TotalDirs:   totalDirs,
		ParentPath:  parentPath,
		IsRecursive: false,
	}

	json.NewEncoder(w).Encode(response)
}

func handleFileContent(w http.ResponseWriter, r *http.Request, fullPath, requestPath string) {
	stat, _ := os.Stat(fullPath)
	isText := isTextFile(fullPath)

	if !isText {
		response := FileContent{
			Path:     "/" + filepath.ToSlash(requestPath),
			Content:  "",
			Size:     stat.Size(),
			ModTime:  stat.ModTime(),
			IsText:   false,
			MimeType: getMimeType(fullPath),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		sendErrorResponse(w, "Unable to read file", http.StatusInternalServerError, err.Error())
		return
	}

	response := FileContent{
		Path:     "/" + filepath.ToSlash(requestPath),
		Content:  string(content),
		Size:     stat.Size(),
		ModTime:  stat.ModTime(),
		IsText:   true,
		MimeType: getMimeType(fullPath),
	}

	json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, message string, code int, details string) {
	w.WriteHeader(code)
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Message: details,
	}
	json.NewEncoder(w).Encode(response)
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade Error:", err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		fmt.Println("Received raw message:", string(message))

		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("JSON unmarshal error:", err)
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Invalid message format"}`))
			continue
		}

		switch msg.Type {
		case "file_edit", "":
			handleFileEdit(conn, msg)
		case "file_create":
			handleFileCreate(conn, msg)
		case "file_delete":
			handleFileDelete(conn, msg)
		case "dir_create":
			handleDirCreate(conn, msg)
		default:
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Unknown message type"}`))
		}
	}
}

func handleFileEdit(conn *websocket.Conn, msg Message) {
	log.Printf("File edit: FileName=%s, Content length=%d\n", msg.FileName, len(msg.Content))

	dir := filepath.Dir(msg.FileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("Create directory error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error creating directory"}`))
		return
	}

	err := ioutil.WriteFile(msg.FileName, []byte(msg.Content), 0644)
	if err != nil {
		log.Println("Write file error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error writing file"}`))
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"message":  "File edited successfully",
		"filename": msg.FileName,
		"size":     len(msg.Content),
	}
	conn.WriteJSON(response)
}

func handleFileCreate(conn *websocket.Conn, msg Message) {
	if _, err := os.Stat(msg.FileName); err == nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "File already exists"}`))
		return
	}

	dir := filepath.Dir(msg.FileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("Create directory error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error creating directory"}`))
		return
	}

	err := ioutil.WriteFile(msg.FileName, []byte(msg.Content), 0644)
	if err != nil {
		log.Println("Create file error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error creating file"}`))
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"message":  "File created successfully",
		"filename": msg.FileName,
	}
	conn.WriteJSON(response)
}

func handleFileDelete(conn *websocket.Conn, msg Message) {
	err := os.Remove(msg.FileName)
	if err != nil {
		log.Println("Delete file error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error deleting file"}`))
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"message":  "File deleted successfully",
		"filename": msg.FileName,
	}
	conn.WriteJSON(response)
}

func handleDirCreate(conn *websocket.Conn, msg Message) {
	err := os.MkdirAll(msg.FileName, 0755)
	if err != nil {
		log.Println("Create directory error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error creating directory"}`))
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Directory created successfully",
		"dirname": msg.FileName,
	}
	conn.WriteJSON(response)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
}

func main() {
	dir := "/code"

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create code directory: %v", err)
	}

	http.Handle("/files/", http.StripPrefix("/files", customFileHandler(dir)))
	http.HandleFunc("/", echo)
	http.HandleFunc("/ws", webSocketHandler)

	fmt.Println("Server starting on port 8080")
	fmt.Printf("Serving files from: %s\n", dir)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /files/?recursive=true - Get all files recursively")
	fmt.Println("  GET /project - Get all project files with contents")
	fmt.Println("  GET /files/path - Get specific file/directory")
	fmt.Println("  WS /ws - WebSocket for file operations")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func echo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New Req")
	w.Write([]byte("Hello"))
}
