// Command bazooka-server runs the Bazooka backend: it owns MUD connections
// and exposes them over gRPC to frontends and automations.
//
// Usage:
//
//	bazooka-server [-listen 127.0.0.1:50051] [-connect host:port] [-session main]
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"google.golang.org/grpc"

	"github.com/tollofsen/bazooka/backend/internal/router"
	"github.com/tollofsen/bazooka/backend/internal/server"
	"github.com/tollofsen/bazooka/backend/internal/session"
	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:50051", "gRPC listen address (keep on loopback: the API is unauthenticated)")
	connect := flag.String("connect", "", "optionally connect to a MUD at startup, as host:port")
	sessionID := flag.String("session", "main", "session ID used with -connect")
	flag.Parse()

	r := router.New()
	mgr := session.NewManager(r)

	if *connect != "" {
		host, portStr, err := net.SplitHostPort(*connect)
		if err != nil {
			log.Fatalf("invalid -connect %q: %v", *connect, err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("invalid port in -connect %q: %v", *connect, err)
		}
		if _, err := mgr.Connect(context.Background(), *sessionID, host, port); err != nil {
			log.Fatalf("connect: %v", err)
		}
		log.Printf("session %q connected to %s", *sessionID, *connect)
	}

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterBazookaServer(grpcServer, server.New(mgr, r))

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Println("shutting down")
		grpcServer.Stop()
		mgr.CloseAll()
	}()

	log.Printf("bazooka backend listening on %s", *listen)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
