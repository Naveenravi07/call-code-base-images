package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	FileName string `json:"filename"`
	Content  string `json:"content"`
}

// Response format for JSON responses
type DirectoryListing struct {
	Path  string   `json:"path"`
	Files []string `json:"files"`
}

func customFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(dir, r.URL.Path)

		// Check if requested path is a directory
		if stat, err := os.Stat(filePath); err == nil && stat.IsDir() {
			// Serve JSON directory listing
			files, err := ioutil.ReadDir(filePath)
			if err != nil {
				http.Error(w, `{"error": "Unable to read directory"}`, http.StatusInternalServerError)
				return
			}

			var fileList []string
			for _, file := range files {
				if file.IsDir() {
					fileList = append(fileList, file.Name()+"/")
				} else {
					fileList = append(fileList, file.Name())
				}
			}

			response := DirectoryListing{
				Path:  r.URL.Path,
				Files: fileList,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// If it's a file, return the content as JSON
		if _, err := os.Stat(filePath); err == nil {
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				http.Error(w, `{"error": "Unable to read file"}`, http.StatusInternalServerError)
				return
			}

			// Return the file content in JSON format
			response := map[string]interface{}{
				"file":     r.URL.Path,
				"content":  string(content),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// File or directory not found
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "File or directory not found"}`, http.StatusNotFound)
	}
}

func main() {
	dir := "/code"
	http.Handle("/files/", http.StripPrefix("/files", customFileHandler(dir)))

	// WebSocket and other handlers
	http.HandleFunc("/", echo)
	http.HandleFunc("/ws", webSocketHandler)

	fmt.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func echo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New Req")
	w.Write([]byte("Hello"))
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

		log.Printf("Parsed message: FileName=%s, Content=%s\n", msg.FileName, msg.Content)

		err = ioutil.WriteFile(msg.FileName, []byte(msg.Content), 0644)
		if err != nil {
			log.Println("Write file error:", err)
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Error writing file"}`))
			continue
		}

		conn.WriteMessage(websocket.TextMessage, []byte(`{"status": "File edited successfully"}`))
	}
}

