package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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

func main() {
	fs := http.FileServer(http.Dir("/code"))

	http.HandleFunc("/", echo)
	http.HandleFunc("/ws", webSocketHandler)
	http.Handle("/files/", http.StripPrefix("/files", fs))

    fmt.Println("websocket server starting in port 8080")
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

