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
	case String:
		formattedString := input[1 : len(input)-1]
		if strings.ToLower(formattedString) == "ping" {
			ans = "+PONG\r\n"
		}
	case Array:
		commandArray := strings.Split(input, "\r\n")
		ans = ""

		if strings.ToLower(commandArray[2]) == "echo" {
			for i := 3; i < len(commandArray)-1; i++ {

				ans += commandArray[i]
				ans += "\r\n"
			}
		}

	}
	return ans

}
