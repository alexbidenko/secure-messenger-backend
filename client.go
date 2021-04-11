package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	User *User
	Conn *websocket.Conn
	Pool *Pool
	Channel string
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
