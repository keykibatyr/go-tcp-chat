package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Client struct {
	name string
	conn net.Conn
	out  chan string
}

type Hub struct {
	registered   chan *Client
	deregistered chan *Client
	broadcast    chan Message
	clients      map[*Client]bool
}

type Message struct {
	sender  *Client
	content string
}

func newHub() *Hub {
	return &Hub{
		registered:   make(chan *Client),
		deregistered: make(chan *Client),
		broadcast:    make(chan Message),
		clients:      make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.registered:
			h.clients[c] = true
			h.sendToAll(fmt.Sprintf("* %s has joined the chat", c.name), c)
		case c := <-h.deregistered:
			_, ok := h.clients[c]
			if ok {
				h.clients[c] = false
				h.sendToAll(fmt.Sprintf("* %s has left the chat", c.name), c)
			}
		case msg := <-h.broadcast:
			h.sendToAll(msg.content, msg.sender)
		}
	}
}

func (h *Hub) sendToAll(msg string, sender *Client) {
	for c := range h.clients {
		if c == sender {
			continue
		}
		select {
		case c.out <- msg:
		default:
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Fatal(err)
	}

	log.Println("chat server listening on :3000")

	defer listener.Close()

	hub := newHub()
	go hub.run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to connect")
			continue
		}

		go handleCon(hub, conn)

	}
}

func handleCon(h *Hub, conn net.Conn) {
	client := &Client{
		name: conn.RemoteAddr().String(),
		conn: conn,
		out:  make(chan string),
	}
	h.registered <- client
	go writer(client)

	defer conn.Close()
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		line := sc.Text()

		if len(line) > 6 && line[:5] == "/nick" {
			old := client.name
			client.name = line[6:]
			client.out <- fmt.Sprintf("your nick is now %s", client.name)
			h.broadcast <- Message{content: fmt.Sprintf("* %s has changed their nick to --> %s <--", old, client.name), sender: client}

		}

		switch {
		case line == "/quit":
			h.deregistered <- client
			conn.Close()
			return
		case line == "/users":
			for user := range h.clients {
				client.out <- fmt.Sprintf("the user: %s", user.name)
			}
		default:
			h.broadcast <- Message{content: fmt.Sprintf("[the user %s]: %s", client.name, line), sender: client}
		}
	}

	err := sc.Err()
	if err != nil {
		log.Println("scanner error:", err)
	}

	h.deregistered <- client
	defer conn.Close()

}

func writer(client *Client) {
	for msg := range client.out {
		fmt.Fprintln(client.conn, msg)
	}
}
