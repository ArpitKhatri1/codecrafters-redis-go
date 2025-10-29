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

func ParseRESPInput(input string) []string {
	commandArray := strings.Split(input, " ")
	fmt.Println(commandArray[1:])
	return commandArray[1:]
}
