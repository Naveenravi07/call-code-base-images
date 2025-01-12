package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

func customFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(dir, r.URL.Path)

		if stat, err := os.Stat(filePath); err == nil && stat.IsDir() {
			// Serve index.html if explicitly requested
			if strings.HasSuffix(r.URL.Path, "index.html") {
				http.ServeFile(w, r, filepath.Join(filePath, "index.html"))
				return
			}

			// Serve directory listing
			files, err := ioutil.ReadDir(filePath)
			if err != nil {
				http.Error(w, "Unable to read directory", http.StatusInternalServerError)
				return
			}

			// Generate a simple HTML listing
			fmt.Fprint(w, "<pre>")
			for _, file := range files {
				name := file.Name()
				if file.IsDir() {
					name += "/"
				}
				fmt.Fprintf(w, `<a href="%s">%s</a><br>`, filepath.Join(r.URL.Path, name), name)
			}
			fmt.Fprint(w, "</pre>")
			return
		}

		http.ServeFile(w, r, filePath)
	}
}

func main() {
	dir := "/code"
	http.Handle("/files/", http.StripPrefix("/files", customFileHandler(dir)))

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
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid message format"))
			continue
		}

		log.Printf("Parsed message: FileName=%s, Content=%s\n", msg.FileName, msg.Content)

		err = ioutil.WriteFile(msg.FileName, []byte(msg.Content), 0644)
		if err != nil {
			log.Println("Write file error:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Error writing file"))
			continue
		}

		conn.WriteMessage(websocket.TextMessage, []byte("File edited successfully"))
	}
}

