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
	rooms		 map[string]*Room
	clients      map[*Client]bool
}

type Message struct {
	sender  *Client
	content string
}

type Room struct{
	name string
	clients map[*Client]bool
	broadcast chan Message
}

func newHub() *Hub {
	return &Hub{
		registered:   make(chan *Client),
		deregistered: make(chan *Client),
		broadcast:    make(chan Message),
		rooms:        make(map[string]*Room),
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
		// case msg := <-h.broadcast:
		// 	h.sendToAll(msg.content, msg.sender)
		
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

func (r *Room) run(){
	for {
		select{
		case msg := <- r.broadcast:
			for c := range r.clients{
				if c == msg.sender{
				continue
			}
				c.out <- msg.content
			}
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
			client.out <- fmt.Sprintf("Online users: (%d)", len(h.clients))
			for user := range h.clients {
				client.out <- fmt.Sprintf("-%s", user.name)
			}
		case len(line) > 8 && line[:7] == "/create":
			room := &Room{
				name: line[7:],
				clients: make(map[*Client]bool),
				broadcast: make(chan Message),
			}
			h.rooms[line[7:]] = room

			go room.run()

			client.out <- fmt.Sprintf("You successfully created a new Room (%s)", line[7:])

		case len(line) > 5 && line[:5] == "/join":
			room := h.rooms[line[5:]]
			room.clients[client] = true
			client.out <- fmt.Sprintf("You joined to the %s", room.name)

		default:
			h.broadcast <- Message{content: fmt.Sprintf("[the user %s]: %s", client.name, line), sender: client}
		}
	}

	err := sc.Err()
	if err != nil {
		log.Println("scanner error:", err)
	}

	h.deregistered <- client

}

func writer(client *Client) {
	for msg := range client.out {
		fmt.Fprintln(client.conn, msg)
	}
}
