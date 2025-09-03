package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", ":3000")

	if err != nil {
		log.Fatal("Connection is Failed")
		return
	}

	defer conn.Close()

	go func() {
		sc := bufio.NewScanner(conn)

		for sc.Scan() {
			line := sc.Text()
			fmt.Println(line)
		}
	}()

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		fmt.Fprintln(conn, line)
	}
}
