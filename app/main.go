package main

import (
	"flag"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

func main() {
	port := flag.String("port", "6379", "Port to listen on")

	flag.Parse()

	server.RunServer(*port)
}
