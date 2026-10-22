// Example: use the session manager directly (no gRPC) to connect to a MUD
// and print its output for a few seconds.
//
//	go run ./backend/examples/connect mud.example.org 4000
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/tollofsen/bazooka/backend/internal/router"
	"github.com/tollofsen/bazooka/backend/internal/session"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: connect <host> <port>")
	}
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	r := router.New()
	m := session.NewManager(r)
	s, err := m.Connect(context.Background(), "example", os.Args[1], port)
	if err != nil {
		log.Fatal(err)
	}
	defer m.CloseAll()
	out, cancel := r.Subscribe(s.ID)
	defer cancel()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case o := <-out:
			fmt.Print(o.Text)
		case <-timeout:
			return
		}
	}
}
