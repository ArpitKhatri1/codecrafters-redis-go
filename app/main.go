package main

import (
	"fmt"
	"net"
	"os"
)

var _ = net.Listen
var _ = os.Exit

func handleConnection(c net.Conn) {
	defer c.Close()
	buff := make([]byte, 1024)

	for {
		_, err := c.Read(buff)
		if err != nil {
			fmt.Println("Connection err", err)
			return
		}

		c.Write([]byte("PONG"))
	}

}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379") //creates a Listener
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	//Creates a bidirectional channel
	for {

		c, err := l.Accept() // Three way handshake , creating a socket of type net.Conn

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(c)
	}
}
