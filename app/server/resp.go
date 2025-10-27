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

func ParseRESPInput(input string) {
	commandArray := strings.Split(input, "\r\n")
	fmt.Println(commandArray[0])
}
