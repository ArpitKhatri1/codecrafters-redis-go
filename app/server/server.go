package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	resp "github.com/codecrafters-io/redis-starter-go/app/RESP"
	types "github.com/codecrafters-io/redis-starter-go/app/types"
)

var _ = net.Listen
var _ = os.Exit

type Server struct {
	*types.ServerState
}

func handleClient(c *types.ClientState) {
	// if the client disconnects remove from replica list
	defer c.ConnectionId.Close()
	fmt.Println("client connected")
	reader := bufio.NewReader(c.ConnectionId)

	for {
		output, err := resp.ParseRESPInput(reader, c)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Client %d disconnected\n", c.Id)
				return
			}
			fmt.Printf("Error handling client %d: %v\n", c.Id, err)
			return
		}
		fmt.Println(output)
		c.ConnectionId.Write([]byte(output))
	}
}

func NewServer(config *types.ServerConfig) *Server {
	serverState := &types.ServerState{
		Config:          config,
		Store:           make(map[string]types.KVV),
		Replicas:        make([]*types.ClientState, 0), //not some number because it will create nil array causing the functionss to panic and throw an error
		PropagationChan: make(chan []string, 100),
	}

	s := &Server{
		ServerState: serverState,
	}

	s.startCleanupRoutine()
	// start immediately before waiting for channel to open
	go s.handlePropagation()
	return s
}

// NewClient still takes the *types.ServerState
func NewClient(server *types.ServerState, conn net.Conn, id int) *types.ClientState {
	client := &types.ClientState{
		Server:           server,
		ConnectionId:     conn,
		Id:               id,
		InTransaction:    false,
		TransactionQueue: nil,
	}
	return client
}

func (s *Server) InitializeReplicantHandshake() {

	conn, err := net.Dial("tcp", s.Config.MasterHost+":"+s.Config.MasterPort)
	replicaPort := s.Config.Port
	reader := bufio.NewReader(conn)
	if err != nil {
		fmt.Println("Err connnecting to master")
		return
	}

	pingCmd := "*1\r\n$4\r\nPING\r\n"
	fmt.Println(pingCmd)
	conn.Write([]byte(pingCmd))
	reader.ReadString('\n')

	replConfigPortCmd := "*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$" + strconv.Itoa(len(replicaPort)) + "\r\n" + replicaPort + "\r\n"
	conn.Write([]byte(replConfigPortCmd))
	reader.ReadString('\n')

	replConfigPsync := "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"
	conn.Write([]byte(replConfigPsync))
	reader.ReadString('\n')

	psyncCmd := "*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"
	conn.Write([]byte(psyncCmd))
	reader.ReadString('\n')

	// read the rdb reponse

	fileSize, err := reader.ReadString('\n')
	fileSize = strings.TrimPrefix(fileSize[1:], "\r\n")
	if err != nil {
		panic(err)
	}

	IfileSize, _ := strconv.Atoi(fileSize)
	byteData := make([]byte, IfileSize)

	io.ReadFull(reader, byteData)
	fmt.Println("File read of size", IfileSize)

	//handle commands from master here , create a special master-replica connection
	// each replica will create a connection to master as a client

	masterClientState := NewClient(s.ServerState, conn, -1)

	fmt.Println("Handshake complete. Listening for propagated commands from master...")

	for {

		output, err := resp.ParseRESPInput(reader, masterClientState)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Master disconnected.")
			} else {
				fmt.Printf("Error reading from master: %v\n", err)
			}
			return // Stop the loop
		}

		_ = output
	}

}

func (s *Server) handlePropagation() {
	for commandArray := range s.PropagationChan {
		s.ReplicaMu.Lock()
		fmt.Println(commandArray, len(s.Replicas))

		if len(s.Replicas) == 0 {
			s.ReplicaMu.Unlock()
			continue
		}

		respString := resp.SerializeToRESPOutput(commandArray)
		fmt.Println(respString)
		for _, client := range s.Replicas {
			// fire and forget pattern
			go func(client *types.ClientState) {
				_, err := client.ConnectionId.Write([]byte(respString))
				if err != nil {
					fmt.Println("Error in propagation")
				}
			}(client)
		}
		s.ReplicaMu.Unlock()
	}
}

func (s *Server) cleanupExpiredKeys() {

	s.StoreMu.Lock()
	defer s.StoreMu.Unlock()

	now := time.Now()
	for key, value := range s.Store {
		if !value.ExpireAt.IsZero() && now.After(value.ExpireAt) {
			delete(s.Store, key)
		}
	}
}

func (s *Server) startCleanupRoutine() {
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			s.cleanupExpiredKeys()
		}
	}()
}

func (s *Server) Start() {
	l, err := net.Listen("tcp", "0.0.0.0:"+s.Config.Port)
	if err != nil {
		fmt.Println("Failed to bind on port " + s.Config.Port)
		return
	}
	fmt.Printf("Server listening on port %s\n", s.Config.Port)

	for i := 0; ; i++ {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting response")
			continue
		}

		fmt.Printf("Accepted new client: %d\n", i)

		// We pass the embedded *types.ServerState to NewClient
		client := NewClient(s.ServerState, conn, i)

		// Call the local handleClient function
		go handleClient(client)
	}
}
