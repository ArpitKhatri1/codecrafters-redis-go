package main

import (
	"flag"
	"math/rand"
	"strings"

	server "github.com/codecrafters-io/redis-starter-go/app/server"
	types "github.com/codecrafters-io/redis-starter-go/app/types"
)

func generateReplid() string {
	str := []byte("abcdefghijklmnopqrstuvwxyz123456789")
	var id []byte
	for i := 0; i < 40; i++ {
		id = append(id, byte(str[rand.Int31n(34)+1]))
	}
	return string(id)

}

func main() {
	port := flag.String("port", "6379", "Port to listen on")
	replicaof := flag.String("replicaof", "", "Replica of which master")

	flag.Parse()

	var config *types.ServerConfig

	if *replicaof != "" {
		// masterHost := strings.Split(*replicaof, " ")[0]
		masterPort := strings.Split(*replicaof, " ")[1]
		config = &types.ServerConfig{
			Port:       *port,
			Role:       "slave",
			Replid:     "?",
			ReplOffset: -1,
			MasterHost: "127.0.0.1",
			MasterPort: masterPort,
		}
	} else {
		config = &types.ServerConfig{
			Port:       *port,
			Role:       "master",
			Replid:     generateReplid(),
			ReplOffset: 0,
		}
	}

	server := server.NewServer(config)

	if server.Config.Role == "slave" {
		go server.InitializeReplicantHandshake()

	}
	server.Start()
}
