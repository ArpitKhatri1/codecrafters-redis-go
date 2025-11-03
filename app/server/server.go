package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
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
	defer c.ConnectionId.Close()
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
		Config: config,
		Store:  make(map[string]types.KVV),
	}

	s := &Server{
		ServerState: serverState,
	}

	s.startCleanupRoutine()
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
	_, err = conn.Write([]byte(pingCmd))

	if err != nil {
		fmt.Println("Failed to send PING to master:", err)
		return
	}

	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to send PING to master:", err)
		return
	}

	replConfigPortCmd := "*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$" + strconv.Itoa(len(replicaPort)) + "\r\n" + replicaPort + "\r\n"

	_, err = conn.Write([]byte(replConfigPortCmd))
	if err != nil {
		fmt.Println("Failed to send REPL1 to master:", err)
		return
	}

	_, err = reader.ReadString('\n')

	if err != nil {
		fmt.Println("Failed to send PING to master:", err)
		return
	}

	replConfigPsync := "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"
	_, err = conn.Write([]byte(replConfigPsync))
	if err != nil {
		fmt.Println("Failed to send REPL2 to master:", err)
		return
	}

	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to send PING to master:", err)
		return
	}

	psyncCmd := "*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"

	_, err = conn.Write([]byte(psyncCmd))
	if err != nil {
		fmt.Println("Failed to send psync to master:", err)
		return
	}

	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to send PING to master:", err)
		return
	}

	// read the rdb reponse
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
