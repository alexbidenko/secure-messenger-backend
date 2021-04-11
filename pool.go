package main

import (
	"encoding/json"
	"fmt"
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
