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
	commandArray := strings.Split(input, "\r\n")
	ans := ""

	if strings.ToLower(commandArray[2]) == "echo" {
		for i := 3; i < len(commandArray)-1; i++ {

			ans += commandArray[i]
			ans += "\r\n"
		}
	}

	return ans
}
