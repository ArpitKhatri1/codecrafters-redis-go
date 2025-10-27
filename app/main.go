package main

import (
	"fmt"
	"net"
	"os"
)

var _ = net.Listen
var _ = os.Exit

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379") //creates a Listener
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	//Creates a bidirectional channel

	c, err := l.Accept() // Three way handshake , creating a socket of type net.Conn

	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	for {
		buff := make([]byte, 1024)

		c.Read(buff)

		c.Write([]byte("+PONG\r\n"))
	}

}
