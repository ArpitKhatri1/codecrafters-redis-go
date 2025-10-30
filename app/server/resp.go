package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	String = '+'
	Bulk   = '$'
	Array  = '*'
)

type KVV struct {
	value    string
	expireAt time.Time
}

var (
	store = make(map[string]KVV)
	mu    sync.RWMutex
)

func ParseRESPInput(input string) string {
	ans := ""
	commandArray := strings.Split(input, "\r\n")
	command := strings.ToLower(commandArray[2])
	switch command {
	case "echo":
		for i := 3; i < len(commandArray)-1; i++ {
			ans += commandArray[i]
			ans += "\r\n"
		}
	case "ping":
		ans += "+PONG\r\n"
	case "set": // set key value [options] [optional value]
		key := commandArray[4]
		keyValue := commandArray[6]
		var value KVV
		var expireAt time.Time
		// check for addition parameters
		if len(commandArray) > 7 {
			// check which option
			option := commandArray[8]
			option = strings.ToLower(option)

			switch option {
			case "px":
				expiryTime := commandArray[10] // string value convert to interget
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

		ans += "+OK\r\n"

	case "get":
		searchKey := commandArray[4]

		mu.Lock()
		value, ok := store[searchKey]
		mu.Unlock()

		// check extractied time

		if !ok {
			ans += "$-1\r\n"
		} else {
			if time.Now().After(value.expireAt) && !value.expireAt.IsZero() {
				mu.Lock()
				delete(store, searchKey)
				mu.Unlock()

				ans += "$-1\r\n"

			} else {

				ans += "$" + strconv.Itoa(len(value.value)) + "\r\n" + value.value + "\r\n"
			}

		}
	}

	return ans

}
