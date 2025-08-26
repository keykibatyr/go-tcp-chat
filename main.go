package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Fatal(err)
	}

	log.Println("chat server listening on :3000")

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to connect")
			continue
		}

		go handleCon(conn)

	}
}

func handleCon(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		line := sc.Text()
		fmt.Fprintln(conn, "echo:", line)
	}
	err := sc.Err()
	if err != nil {
		log.Println("scanner error:", err)
	}
}
