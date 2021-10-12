package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const (
	MessageType    = iota
	JoinType       = iota
	DisconnectType = iota
	OfferType      = iota
	AnswerType     = iota
)

type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
	Client *Client `json:"-"`
}

type MessageBody struct {
	Type int `json:"type"`
	Content map[string]interface{} `json:"content"`
	User User `json:"user"`
}

type User struct {
	Id float64 `json:"id"`
	Name string `json:"name"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return conn, nil
}

func serveWs(pool *Pool, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	fmt.Println("WebSocket Endpoint Hit")
	conn, err := Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &Client{
		Conn: conn,
		Pool: pool,
		Channel: vars["channel"],
	}

	pool.Register <- client
	client.Read()
}

func main() {
	pool := NewPool()
	go pool.Start()

	r := mux.NewRouter()
	r.HandleFunc("/chat/{channel}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})
	fmt.Println("Server started!")
	log.Fatal(http.ListenAndServe(":7777", r))
}
