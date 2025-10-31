package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	resp "github.com/codecrafters-io/redis-starter-go/app/RESP"
)

var _ = net.Listen
var _ = os.Exit

func handleConnection(c net.Conn) {
	defer c.Close()
	//even tho this is blocking , go handles the thread to another go routine.
	reader := bufio.NewReader(c) //aviod partial read if we use a byte channel , if full line ending with \n give that line solving buffer splitting

	for {

		output, err := resp.ParseRESPInput(reader)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Client disconnected")
				return
			}
			fmt.Println("Client removed")
			return
		}

		c.Write([]byte(output))

		// c.Write([]byte("+PONG\r\n"))
	}

}

func RunServer() {

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
