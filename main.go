package main

import (
	"bufio"
	"fmt"
	"log"
	"context"
	"net"
)

type Client struct {
	name string
	conn net.Conn
	out  chan string
	room *Room
}

type Hub struct {
	registered   chan *Client
	deregistered chan *Client
	broadcast    chan Message
	createreq 	chan string
	rooms        map[string]*Room
	clients      map[*Client]bool
}

type Message struct {
	sender  *Client
	content string
}

type Room struct {
	name      string
	clients   map[*Client]bool
	broadcast chan Message
}

type Broadcaster interface {
	sendToAll(msg Message, sender *Client)
}

func newHub() *Hub {
	return &Hub{
		registered:   make(chan *Client),
		deregistered: make(chan *Client),
		broadcast:    make(chan Message),
		createreq: 	  make(chan string),
		rooms:        make(map[string]*Room),
		clients:      make(map[*Client]bool),
	}
}

func (h *Hub) run(ctx context.Context) {
	for {
		select {
		case room_name := <- h.createreq:
			ctx, cancel := context.WithCancel(context.Background())

			defer cancel()

			room := &Room{
				name:      room_name,
				clients:   make(map[*Client]bool),
				broadcast: make(chan Message),
			}
			
			h.rooms[room_name] = room
			go room.run(ctx)

		case <- ctx.Done():
			return
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
			h.sendToAll(msg.content, msg.sender) //WILL ADD IT LATER))

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

func (r *Room) sendToAll(msg string, sender *Client) {
	for c := range r.clients {
		if c == sender {
			continue
		}
		c.out <- msg
	}
}

func (r *Room) run(ctx context.Context) {
	for {
		select {
		case <- ctx.Done():
			return
		case msg := <-r.broadcast:
			r.sendToAll(msg.content, msg.sender)
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	
	log.Println("chat server listening on :3000")

	defer func () {
		cancel()

		listener.Close()
	}()

	hub := newHub()
	go hub.run(ctx)

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
		room: nil,
	}

	h.registered <- client

	ctx, cancel := context.WithCancel(context.Background())
	
	defer func () {

		h.deregistered <- client

		close(client.out)

		cancel()

		conn.Close()

	} ()

	go writer(ctx, client)

	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		line := sc.Text()

		switch {
		case line == "/quit":
			h.deregistered <- client
			delete(h.clients, client)
			conn.Close()
			return
		case line == "/users":
			client.out <- fmt.Sprintf("Online users: (%d)", len(h.clients))
			for user := range h.clients {
				client.out <- fmt.Sprintf("-%s", user.name)
			}
		case len(line) > 8 && line[:7] == "/create":
			// go room.run(ctx)  //add context 
			h.createreq <- fmt.Sprint(line[8:])
			log.Println("Sending to", client.name, ":", line)
			client.out <- fmt.Sprintf("You successfully created a new Room (%s)", line[8:])

		case len(line) > 5 && line[:5] == "/join":
			client.room = h.rooms[line[6:]]
			client.room.clients[client] = true
			client.out <- fmt.Sprintf("You joined to the %s", client.room.name)

		case len(line) > 6 && line[:5] == "/nick":
			old := client.name
			client.name = line[6:]
			client.out <- fmt.Sprintf("your nick is now %s", client.name)
			client.room.broadcast <- Message{content: fmt.Sprintf("* %s has changed their nick to --> %s <--", old, client.name), sender: client}


		default:
			if client.room != nil {
				client.room.broadcast <- Message{content: fmt.Sprintf("[the user %s]: %s", client.name, line), sender: client}
			} else {
				client.out <- "You are not in a room. Use /join <room> first." // fix this bulshit architecture, make a an option sending to main and rooms
			}
		}
	}

	err := sc.Err()
	if err != nil {
		log.Println("scanner error:", err)
	}

	

}

func writer(ctx context.Context, client *Client) {
	fmt.Println("Writer started for", client.name)
	for{
		select{
		case msg, err := <- client.out:
			if !err {
				return
			}
			fmt.Fprintln(client.conn, msg)
		case <- ctx.Done():
			return
		}
	}
}


//making the client logging more complex (relogging opportunity)
//not allowing overwrite the already existing room by creating the room with same name
//handle the double main() func trouble