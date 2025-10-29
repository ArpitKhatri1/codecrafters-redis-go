package server

import (
	"strings"
)

const (
	String = '+'
	Bulk   = '$'
	Array  = '*'
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

		}

	}
	return ans

}
