package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	resp "github.com/codecrafters-io/redis-starter-go/app/RESP"
	types "github.com/codecrafters-io/redis-starter-go/app/types"
)

var _ = net.Listen
var _ = os.Exit

// Server is a local struct in the 'server' package
// It embeds *types.ServerState to gain its fields and data.
type Server struct {
	*types.ServerState
}

// handleClient is now a local function in the 'server' package,
// not a method on types.ClientState.
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

		c.ConnectionId.Write([]byte(output))
	}
}

// NewServer now returns the local *Server type
func NewServer(config *types.ServerConfig) *Server {
	// Create the underlying ServerState from the 'types' package
	serverState := &types.ServerState{
		Config: config,
		Store:  make(map[string]types.KVV),
	}

	// Create the local Server wrapper
	s := &Server{
		ServerState: serverState,
	}

	// Call the method on the local *Server type
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

// This method is now correctly defined on the local *Server type
func (s *Server) cleanupExpiredKeys() {
	// Fields are accessed directly via embedding
	s.StoreMu.Lock()
	defer s.StoreMu.Unlock()

	now := time.Now()
	for key, value := range s.Store {
		if !value.ExpireAt.IsZero() && now.After(value.ExpireAt) {
			delete(s.Store, key)
		}
	}
}

// This method is now correctly defined on the local *Server type
func (s *Server) startCleanupRoutine() {
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			s.cleanupExpiredKeys()
		}
	}()
}

// This method is now correctly defined on the local *Server type
func (s *Server) Start() {
	l, err := net.Listen("tcp", "0.0.0.0"+s.Config.Port)
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
