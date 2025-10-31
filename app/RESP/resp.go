package resp

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	transactions "github.com/codecrafters-io/redis-starter-go/app/transactions"
)

type KVV struct {
	value    string
	expireAt time.Time
}

var (
	store = make(map[string]KVV)
	mu    sync.RWMutex
)

type RESPParser struct {
	commandArray []string
	command      string
}

func NewRESPParser(commandArray []string) *RESPParser {

	return &RESPParser{

		commandArray: commandArray,
		command:      commandArray[0],
	}
}

func (r *RESPParser) handlePING() string {
	return "+PONG\r\n"
}

func (r *RESPParser) handleECHO() string {
	temp := "$"
	for i := 1; i < len(r.commandArray); i++ {

		temp += strconv.Itoa(len(r.commandArray[i]))
		temp += "\r\n"
		temp += r.commandArray[i]
		temp += "\r\n"
	}
	return temp
}

func (r *RESPParser) handleGET() string {

	searchKey := r.commandArray[1]

	mu.Lock()
	value, ok := store[searchKey]
	mu.Unlock()

	// check extractied time

	if !ok {
		return returnSpecialBlukErrorStatus()
	} else {
		if time.Now().After(value.expireAt) && !value.expireAt.IsZero() {
			mu.Lock()
			delete(store, searchKey)
			mu.Unlock()

			return returnSpecialBlukErrorStatus()

		} else {

			return "$" + strconv.Itoa(len(value.value)) + "\r\n" + value.value + "\r\n"
		}

	}

}

func (r *RESPParser) handleSET() string {

	key := r.commandArray[1]
	keyValue := r.commandArray[2]
	var value KVV
	var expireAt time.Time
	// check for addition parameters
	if len(r.commandArray) >= 4 {
		// check which option
		option := r.commandArray[3]
		option = strings.ToLower(option)

		switch option {
		case "px":
			expiryTime := r.commandArray[4] // string value convert to interget
			formattedTime, err := time.ParseDuration(expiryTime + "ms")
			if err != nil {
				return returnRESPErrorString("ERR")
			}
			value = KVV{
				value:    keyValue,
				expireAt: time.Now().Add(formattedTime),
			}

		}
	} else {
		value = KVV{
			value:    keyValue,
			expireAt: expireAt,
		}
	}

	mu.Lock()
	store[key] = value
	mu.Unlock()

	return returnOKStatus()
}

func (r *RESPParser) handleINCR() string {
	key := r.commandArray[1]
	var increased int
	mu.Lock()
	defer mu.Unlock()
	value, exists := store[key]
	//check if value is integer
	if !exists {
		value.value = "1"
		store[key] = value
		return returnRESPInteger(1)
	}

	val, err := strconv.Atoi(value.value)
	if err != nil {
		return "-ERR value is not an integer or out of range\r\n"
	}
	val += 1
	increased = val
	value.value = strconv.Itoa(val)
	store[key] = value

	return returnRESPInteger(increased)
}

func (r *RESPParser) handleMULTI(c net.Conn) string {
	transactions.CreateTransaction(c)
	return returnOKStatus()

}

func (r *RESPParser) handleEXEC(c net.Conn) string {
	transactionsList := transactions.GetTransactionsForConnection(c) // return transactions pointer which is private so need a method to access queue

	if transactionsList == nil {
		return returnRESPErrorString("ERR EXEC without MULTI")
	}

	transactions.HandleDeleteConnection(c)
	queue := transactionsList.GetQueue()

	if len(queue) == 0 {
		return "*0\r\n"
	}

	ansString := "*" + strconv.Itoa(len(queue)) + "\r\n"

	//length + \r\n

	for _, queries := range queue {
		parser := NewRESPParser(queries)
		ansString += parser.handleCommandSelection()
	}

	return ansString
}

func (r *RESPParser) handleDISCARD(c net.Conn) string {
	transactionsList := transactions.GetTransactionsForConnection(c)

	if transactionsList == nil {
		return returnRESPErrorString("ERR DISCARD without MULTI")
	}
	transactions.HandleDeleteConnection(c)
	return returnOKStatus()
}

func (r *RESPParser) handleCommandSelection() string {
	switch r.command {
	case ECHO:
		return r.handleECHO()
	case PING:
		return r.handlePING()

	case SET: // set key value [options] [optional value]
		return r.handleSET()

	case GET:
		return r.handleGET()

	case INCR:
		return r.handleINCR()
	default:
		return "-ERR"

	}
}

// add a go routine which runs every second for active checks

func init() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			cleanupExpiredKeys()
		}
	}()
}

func cleanupExpiredKeys() {
	mu.Lock()
	defer mu.Unlock()
	for key, value := range store {
		if time.Now().After(value.expireAt) && !value.expireAt.IsZero() {

			delete(store, key)

		}
	}

}

func ParseRESPInput(reader *bufio.Reader, c net.Conn) (string, error) {

	line, err := reader.ReadString('\n') //store in buffer until it accquires \n which then stops and return in line

	if err != nil {
		return "", err
	}

	line = strings.TrimSuffix(line, "\r\n")

	switch line[0] {
	case '*':
		return parseArray(line, reader, c)

	default:
		return "", fmt.Errorf("unknow type")
	}

}

func parseArray(line string, reader *bufio.Reader, c net.Conn) (string, error) {
	commandLength, err := strconv.Atoi(line[1:])

	if err != nil {
		return "", err
	}
	commandArray := make([]string, commandLength)

	for i := 0; i < commandLength; i++ {
		_, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		data, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		data = strings.TrimSuffix(data, "\r\n")

		commandArray[i] = strings.ToLower(data)
	}

	parser := NewRESPParser(commandArray)

	//check if a transaction is already present
	transactionsList := transactions.GetTransactionsForConnection(c)

	if transactionsList != nil {
		switch parser.command {
		case EXEC:
			return parser.handleEXEC(c), nil
		case DISCARD:
			return parser.handleDISCARD(c), nil
		default:
			return transactions.AddCommandToQueue(c, commandArray), nil
		}
	}

	// dispatcher

	switch parser.command {
	case ECHO:
		return parser.handleECHO(), nil
	case PING:
		return parser.handlePING(), nil

	case SET: // set key value [options] [optional value]
		return parser.handleSET(), nil

	case GET:
		return parser.handleGET(), nil

	case INCR:
		return parser.handleINCR(), nil
	case MULTI:
		return parser.handleMULTI(c), nil
	case EXEC:
		return parser.handleEXEC(c), nil
	case DISCARD:
		return parser.handleDISCARD(c), nil
	default:
		return "-ERR", nil

	}

}
