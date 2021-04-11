package main

import (
	"encoding/json"
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
	User *User
	Conn *websocket.Conn
	Pool *Pool
	Channel string
}

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

		var messageBody MessageBody
		json.Unmarshal(p, &messageBody)

		var message Message
		if messageBody.Type == JoinType {
			fmt.Println(messageBody.Content)
			user := User{
				Id: messageBody.Content["id"].(float64),
				Name: messageBody.Content["name"].(string),
			}
			c.User = &user

			var users []User
			for k, _ := range c.Pool.Channels[c.Channel] {
				users = append(users, *k.User)
			}
			body, _ := json.Marshal(map[string][]User{"users": users})

			message = Message{Type: messageBody.Type, Body: string(body), Client: c}
		} else {
			messageBody.User = *c.User
			body, _ := json.Marshal(messageBody)
			message = Message{Type: messageBody.Type, Body: string(body), Client: c}
		}

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
			}
			break
		case client := <-pool.Unregister:
			channel := pool.Channels[client.Channel]
			if channel != nil {
				if _, ok := channel[client]; ok {
					delete(channel, client)

					var users []User
					for k, _ := range client.Pool.Channels[client.Channel] {
						users = append(users, *k.User)
					}
					body, _ := json.Marshal(map[string][]User{"users": users})

					message := Message{Type: DisconnectType, Body: string(body), Client: client}

					for client, _ := range channel {
						client.Conn.WriteJSON(message)
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
