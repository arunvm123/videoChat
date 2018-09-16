package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Data is used to store various websocket response data
type Data struct {
	Type      string      `json:"type"`
	Name      string      `json:"name,omitempty"`
	Candidate interface{} `json:"candidate,omitempty"`
	Offer     interface{} `json:"offer,omitempty"`
	Answer    interface{} `json:"answer,omitempty"`
}

var users map[string]*websocket.Conn

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {

	users = make(map[string]*websocket.Conn)

	r := mux.NewRouter()

	r.HandleFunc("/", frontPage)
	r.HandleFunc("/websocket", websocketHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static/"))))

	m := handlers.LoggingHandler(os.Stdout, r)

	err := http.ListenAndServe(":8080", m)
	if err != nil {
		panic(err)
	}
}

func frontPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/index.html")
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Client subscribed")
	var data Data
	for {
		_, msg, err := connection.ReadMessage()
		err = json.Unmarshal(msg, &data)
		if err != nil {
			panic(err)
		}

		switch data.Type {
		case "login":
			if _, check := users[data.Name]; check {
				jsonData, err := json.Marshal(struct {
					Type    string `json:"type"`
					Success string `json:"success"`
				}{
					"login",
					"false",
				})
				if err != nil {
					panic(err)
				}
				connection.WriteMessage(1, jsonData)
			} else {
				users[data.Name] = connection
				connection.Name = data.Name
				jsonData, err := json.Marshal(struct {
					Type    string `json:"type"`
					Success string `json:"success"`
				}{
					"login",
					"true",
				})
				if err != nil {
					panic(err)
				}
				connection.WriteMessage(1, jsonData)
			}

		case "offer":
			fmt.Println("Sending offer to " + data.Name)
			conn := users[data.Name]

			if conn != nil {
				connection.OtherName = data.Name

				jsonData, err := json.Marshal(struct {
					Type  string      `json:"type"`
					Offer interface{} `json:"offer"`
					Name  string      `json:"name"`
				}{
					"offer",
					data.Offer,
					connection.Name,
				})

				if err != nil {
					panic(err)
				}

				conn.WriteMessage(1, jsonData)
			}

		case "answer":
			fmt.Println("Sending answer to" + data.Name)

			conn := users[data.Name]

			if conn != nil {
				connection.OtherName = data.Name

				jsonData, err := json.Marshal(struct {
					Type   string      `json:"type"`
					Answer interface{} `json:"answer"`
				}{
					"answer",
					data.Answer,
				})
				if err != nil {
					panic(err)
				}

				conn.WriteMessage(1, jsonData)

			}

		case "candidate":
			fmt.Println("Sending candidate to" + data.Name)

			conn := users[data.Name]

			if conn != nil {
				jsonData, err := json.Marshal(struct {
					Type      string      `json:"type"`
					Candidate interface{} `json:"candidate"`
				}{
					"candidate",
					data.Candidate,
				})

				if err != nil {
					panic(err)
				}

				conn.WriteMessage(1, jsonData)
			}

		case "leave":
			fmt.Println("Disconnecting from ", data.Name)

			conn := users[data.Name]
			conn.OtherName = ""

			if conn != nil {
				jsonData, err := json.Marshal(struct {
					Type string `json:"type"`
				}{
					"leave",
				})

				if err != nil {
					panic(err)
				}

				conn.WriteMessage(1, jsonData)

			}
		}
	}
}
