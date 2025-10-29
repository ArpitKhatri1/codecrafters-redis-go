package server

import (
	"strconv"
	"strings"
	"sync"
)

const (
	String = '+'
	Bulk   = '$'
	Array  = '*'
)

var (
	store = make(map[string]string)
	mu    sync.RWMutex
)

func ParseRESPInput(input string) string {
	ans := ""
	switch input[0] {
	case Array:
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
		case "set":
			key := commandArray[4]
			value := commandArray[6]

			mu.Lock()
			store[key] = value
			mu.Unlock()

			ans += "+OK\r\n"

		case "get":
			searchKey := commandArray[4]

			mu.Lock()
			value, ok := store[searchKey]
			mu.Unlock()

			if !ok {
				ans += "$-1\r\n"
			} else {
				ans += "$" + strconv.Itoa(len(value)) + "\r\n" + value + "\r\n"
			}
		}
	}
	return ans

}
