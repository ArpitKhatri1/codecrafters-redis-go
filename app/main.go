package main

import (
	"flag"

	server "github.com/codecrafters-io/redis-starter-go/app/server"
	types "github.com/codecrafters-io/redis-starter-go/app/types"
)

func main() {
	port := flag.String("port", "6379", "Port to listen on")
	replicaof := flag.String("replicaof", "localhost 6379", "Replica of which master")

	flag.Parse()

	role := "master"
	if *replicaof == "" {
		role = "slave"
	}
	config := &types.ServerConfig{
		Port: *port,
		Role: role,
	}
	server := server.NewServer(config)
	server.Start()
}
