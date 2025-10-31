package transactions

import (
	"net"
	"sync"
)

type transactions struct {
	queue [][]string
}

var (
	connTx = make(map[net.Conn]*transactions)
	connMu sync.Mutex
)

func CreateTransaction(c net.Conn) {
	connMu.Lock()
	defer connMu.Unlock()

	connTx[c] = &transactions{
		queue: make([][]string, 0),
	}
}

func HandleDeleteConnection(c net.Conn) {
	connMu.Lock()
	defer connMu.Unlock()

	delete(connTx, c)
}

func GetTransactionsForConnection(c net.Conn) *transactions {
	connMu.Lock()
	defer connMu.Unlock()

	return connTx[c]
}

func AddCommandToQueue(c net.Conn, arr []string) string {
	connMu.Lock()
	defer connMu.Unlock()

	//get the transaction

	tx := GetTransactionsForConnection(c)

	tx.queue = append(tx.queue, arr)

	return "+QUEUED\r\n"

}
func (t *transactions) GetQueue() [][]string {
	return t.queue
}
