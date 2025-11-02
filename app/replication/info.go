package replication

import "sync"

type ServerStatus struct {
	role string
}

var ServerMapping = make(map[*string]ServerStatus) //maps port to server status
var portMu sync.RWMutex

func AddServerToMapping(port *string, role string) {
	portMu.Lock()
	defer portMu.Unlock()
	ServerMapping[port] = ServerStatus{
		role,
	}
}
