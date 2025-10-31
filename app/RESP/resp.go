package resp

import (
	"fmt"
	"os"
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
	userInput    string
	commandArray []string
	command      string
}

func NewRESPParser(input string) *RESPParser {

	commandArray := strings.Split(input, "\r\n")
	command := strings.ToLower(commandArray[2])

	return &RESPParser{
		userInput:    input,
		commandArray: commandArray,
		command:      command,
	}
}

func (r *RESPParser) handlePING() string {
	return "+PONG\r\n"
}

func (r *RESPParser) handleECHO() string {
	temp := ""
	for i := 3; i < len(r.commandArray)-1; i++ {
		temp += r.commandArray[i]
		temp += "\r\n"
	}
	return temp
}

func (r *RESPParser) handleGET() string {

	searchKey := r.commandArray[4]

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

	key := r.commandArray[4]
	keyValue := r.commandArray[6]
	var value KVV
	var expireAt time.Time
	// check for addition parameters
	if len(r.commandArray) >= 9 {
		// check which option
		option := r.commandArray[8]
		option = strings.ToLower(option)

		switch option {
		case "px":
			expiryTime := r.commandArray[10] // string value convert to interget
			formattedTime, err := time.ParseDuration(expiryTime + "ms")
			if err != nil {
				fmt.Println("There was some error")
				os.Exit(1)
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

func (r *RESPParser) getKeywordAtPosition(position int) string { // one - based position
	return r.commandArray[position*2]
}

func (r *RESPParser) handleINCR() string {
	key := r.getKeywordAtPosition(2)
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

	for key, value := range store {
		if time.Now().After(value.expireAt) && !value.expireAt.IsZero() {

			mu.Lock()
			delete(store, key)
			mu.Unlock()

		}
	}

}

func ParseRESPInput(input string) string {
	parser := NewRESPParser(input)

	switch parser.command {
	case ECHO:
		return parser.handleECHO()
	case PING:
		return parser.handlePING()

	case SET: // set key value [options] [optional value]
		return parser.handleSET()

	case GET:
		return parser.handleGET()

	case INCR:
		return parser.handleINCR()
	default:
		return "-ERR"

	}

}
