package resp

import (
	"strconv"
)

func returnOKStatus() string {
	return "+OK\r\n"
}

func returnSpecialBlukErrorStatus() string {
	return "$-1\r\n"
}

func returnRESPInteger(num int) string {
	return ":" + strconv.Itoa(num) + "\r\n"
}

func returnRESPBlukString(str string) string {
	return "$" + str + "\r\n"
}

func returnRESPErrorString(str string) string {
	return "-" + str + "\r\n"
}
