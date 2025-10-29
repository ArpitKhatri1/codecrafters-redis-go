package server

import (
	"fmt"
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
		formattedString := strings.TrimSuffix(input[1:], "\r\n")
		fmt.Println(formattedString)
		if strings.ToLower(formattedString) == "ping" {
			ans += "+PONG\r\n"
		}
	case Array:
		commandArray := strings.Split(input, "\r\n")
		if strings.ToLower(commandArray[2]) == "echo" {
			for i := 3; i < len(commandArray)-1; i++ {

				ans += commandArray[i]
				ans += "\r\n"
			}
		}

	}
	return ans

}
