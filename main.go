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
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Channels   map[string]map[*Client]bool
	Broadcast  chan Message
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Channels:   make(map[string]map[*Client]bool),
		Broadcast:  make(chan Message),
	}
}

type Client struct {
	ID string
	Conn *websocket.Conn
	Pool *Pool
	Channel string
}

type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
	Client *Client `json:"-"`
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		message := Message{Type: MessageType, Body: string(p), Client: c}
		c.Pool.Broadcast <- message
		fmt.Printf("Message Received: %+v\n", message)
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			channel := pool.Channels[client.Channel]
			if channel == nil {
				channel = make(map[*Client]bool)
				pool.Channels[client.Channel] = channel
			}
			channel[client] = true
			for client, _ := range channel {
				fmt.Println(client)
				client.Conn.WriteJSON(Message{Type: JoinType, Body: "New User Joined..."})
			}
			break
		case client := <-pool.Unregister:
			channel := pool.Channels[client.Channel]
			if channel != nil {
				if _, ok := channel[client]; ok {
					delete(channel, client)
					for client, _ := range channel {
						client.Conn.WriteJSON(Message{Type: DisconnectType, Body: "User Disconnected..."})
					}
				}
			}
			break
		case message := <-pool.Broadcast:
			fmt.Println("Sending message to all clients in Pool")
			channel := pool.Channels[message.Client.Channel]
			if channel != nil {
				for client, _ := range channel {
					if err := client.Conn.WriteJSON(message); err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}
	}
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
