package resp

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
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

func (r *RESPParser) handleMULTI() string {
	return returnOKStatus()

}

func (r *RESPParser) handleEXEC() string {
	return returnRESPErrorString("ERR EXEC without MULTI")
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

func ParseRESPInput(reader *bufio.Reader) (string, error) {

	line, err := reader.ReadString('\n') //store in buffer until it accquires \n which then stops and return in line

	if err != nil {
		return "", err
	}

	line = strings.TrimSuffix(line, "\r\n")

	switch line[0] {
	case '*':
		return parseArray(line, reader)

	default:
		return "", fmt.Errorf("unknow type")
	}

}

func parseArray(line string, reader *bufio.Reader) (string, error) {
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

	// dispatcher

	parser := NewRESPParser(commandArray)

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
		return parser.handleMULTI(), nil
	case EXEC:
		return parser.handleEXEC(), nil
	default:
		return "-ERR", nil

	}

}
