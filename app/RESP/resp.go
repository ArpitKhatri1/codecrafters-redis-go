package resp

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	types "github.com/codecrafters-io/redis-starter-go/app/types"
)

type RESPParser struct {
	client       *types.ClientState
	commandArray []string
	command      string
}

func NewRESPParser(commandArray []string, client *types.ClientState) *RESPParser {
	return &RESPParser{
		client:       client,
		commandArray: commandArray,
		command:      commandArray[0],
	}
}

func (r *RESPParser) handlePING() string {
	return "+PONG\r\n"
}

func (r *RESPParser) handleECHO() string {
	arg := r.commandArray[1]
	return "$" + strconv.Itoa(len(arg)) + "\r\n" + arg + "\r\n"
}

func (r *RESPParser) handleGET() string {
	r.client.Server.StoreMu.RLock()
	defer r.client.Server.StoreMu.RUnlock()
	return r.handleGETUnlocked()
}

func (r *RESPParser) handleGETUnlocked() string {
	searchKey := r.commandArray[1]
	value, ok := r.client.Server.Store[searchKey]

	if !ok {
		return returnSpecialBlukErrorStatus()
	} else {
		if !value.ExpireAt.IsZero() && time.Now().After(value.ExpireAt) {
			return returnSpecialBlukErrorStatus()
		} else {
			return "$" + strconv.Itoa(len(value.Value)) + "\r\n" + value.Value + "\r\n"
		}
	}
}

func (r *RESPParser) handleSET() string {
	r.client.Server.StoreMu.Lock()
	defer r.client.Server.StoreMu.Unlock()
	return r.handleSETUnlocked()
}

func (r *RESPParser) handleSETUnlocked() string {
	key := r.commandArray[1]
	keyValue := r.commandArray[2]
	var value types.KVV
	var expireAt time.Time

	if len(r.commandArray) >= 4 {
		option := r.commandArray[3]
		switch option {
		case "px":
			if len(r.commandArray) < 5 {
				return returnRESPErrorString("syntax error")
			}
			expiryTime := r.commandArray[4]
			formattedTime, err := time.ParseDuration(expiryTime + "ms")
			if err != nil {
				return returnRESPErrorString("value is not an integer or out of range")
			}
			expireAt = time.Now().Add(formattedTime)
		}
	}

	value = types.KVV{
		Value:    keyValue,
		ExpireAt: expireAt,
	}

	r.client.Server.Store[key] = value

	if r.client.Server.Config.Role == "master" {
		r.client.Server.PropagationChan <- r.commandArray
	}

	return returnOKStatus()
}

func (r *RESPParser) handleINCR() string {
	r.client.Server.StoreMu.Lock()
	defer r.client.Server.StoreMu.Unlock()
	return r.handleINCRUnlocked()
}

func (r *RESPParser) handleINCRUnlocked() string {
	key := r.commandArray[1]
	var increased int

	value, exists := r.client.Server.Store[key]

	if !exists {
		value.Value = "1"
		r.client.Server.Store[key] = value
		return returnRESPInteger(1)
	}

	val, err := strconv.Atoi(value.Value)
	if err != nil {
		return returnRESPErrorString("value is not an integer or out of range")
	}
	val += 1
	increased = val
	value.Value = strconv.Itoa(val)
	r.client.Server.Store[key] = value

	if r.client.Server.Config.Role == "master" {
		fmt.Println("added")
		r.client.Server.PropagationChan <- r.commandArray
	}

	return returnRESPInteger(increased)
}

func (r *RESPParser) handleMULTI() string {
	if r.client.InTransaction {
		return returnRESPErrorString("MULTI calls can not be nested")
	}
	r.client.InTransaction = true
	r.client.TransactionQueue = make([][]string, 0)
	return returnOKStatus()
}

func (r *RESPParser) handleEXEC() string {
	if !r.client.InTransaction {
		return returnRESPErrorString("EXEC without MULTI")
	}

	queue := r.client.TransactionQueue
	r.client.InTransaction = false
	r.client.TransactionQueue = nil

	if len(queue) == 0 {
		return "*0\r\n"
	}

	ansString := "*" + strconv.Itoa(len(queue)) + "\r\n"

	r.client.Server.StoreMu.Lock()
	defer r.client.Server.StoreMu.Unlock()

	for _, queries := range queue {
		parser := NewRESPParser(queries, r.client)
		ansString += parser.handleCommandSelection()
	}

	return ansString
}

func (r *RESPParser) handleDISCARD() string {
	if !r.client.InTransaction {
		return returnRESPErrorString("DISCARD without MULTI")
	}
	r.client.InTransaction = false
	r.client.TransactionQueue = nil
	return returnOKStatus()
}

func (r *RESPParser) handleINFO() string {

	role := r.client.Server.Config.Role
	replid := r.client.Server.Config.Replid
	offset := strconv.Itoa(r.client.Server.Config.ReplOffset)

	infoLines := []string{
		"# Replication",
		"role:" + role,
		"master_replid:" + replid,
		"master_repl_offset:" + offset,
	}

	infoString := strings.Join(infoLines, "\r\n") + "\r\n"

	return "$" + strconv.Itoa(len(infoString)) + "\r\n" + infoString + "\r\n"
}

func (r *RESPParser) handleREPLCONF() string {
	return returnOKStatus()
}

func (r *RESPParser) handlePSYNC() string {
	masterReplId := r.client.Server.Config.Replid

	// Open empty.rdb only as a placeholder for now
	data, err := os.ReadFile("empty.rdb")
	if err != nil {
		panic(err)
	}

	fileSize := len(data)

	// r.client.Server.ReplicaMu.Lock()
	// defer r.client.Server.ReplicaMu.Unlock()
	// r.client.Server.Replicas = append(r.client.Server.Replicas, r.client)

	// FULLRESYNC + bulk string header + raw bytes
	return "+FULLRESYNC " + masterReplId + " 0\r\n" +
		"$" + strconv.Itoa(fileSize) + "\r\n" +
		string(data)
}

func (r *RESPParser) handleCommandSelection() string {
	switch r.command {
	case ECHO:
		return r.handleECHO()
	case PING:
		return r.handlePING()
	case SET:
		return r.handleSETUnlocked()
	case GET:
		return r.handleGETUnlocked()
	case INCR:
		return r.handleINCRUnlocked()
	default:
		return returnRESPErrorString("unknown command '" + r.command + "'")
	}
}

func ParseRESPInput(reader *bufio.Reader, c *types.ClientState) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSuffix(line, "\r\n")
	if len(line) == 0 {
		return "", fmt.Errorf("empty input")
	}

	fmt.Println(line)

	switch line[0] {
	case '*':
		return parseArray(line, reader, c)
	default:
		return "", fmt.Errorf("unknown input type: %s", line)
	}
}

func SerializeToRESPOutput(commandArray []string) string {
	result := ""
	result += "*" + strconv.Itoa(len(commandArray)) + "\r\n"

	for _, command := range commandArray {
		result += "$" + strconv.Itoa(len(command)) + "\r\n"
		result += command + "\r\n"
	}

	return result
}

func parseArray(line string, reader *bufio.Reader, c *types.ClientState) (string, error) {
	commandLength, err := strconv.Atoi(line[1:])
	if err != nil {
		return "", err
	}
	commandArray := make([]string, commandLength)

	for i := 0; i < commandLength; i++ {
		_, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		data, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		data = strings.TrimSuffix(data, "\r\n")

		if i == 0 {
			commandArray[i] = strings.ToLower(data)
		} else {
			commandArray[i] = data
		}
	}

	parser := NewRESPParser(commandArray, c)

	if c.InTransaction {
		switch parser.command {
		case EXEC:
			return parser.handleEXEC(), nil
		case DISCARD:
			return parser.handleDISCARD(), nil
		case MULTI:
			return returnRESPErrorString("MULTI calls can not be nested"), nil
		default:
			c.TransactionQueue = append(c.TransactionQueue, commandArray)
			return "+QUEUED\r\n", nil
		}
	}

	switch parser.command {
	case ECHO:
		return parser.handleECHO(), nil
	case PING:
		return parser.handlePING(), nil
	case SET:
		return parser.handleSET(), nil
	case GET:
		return parser.handleGET(), nil
	case INCR:
		return parser.handleINCR(), nil
	case MULTI:
		return parser.handleMULTI(), nil
	case EXEC:
		return parser.handleEXEC(), nil
	case DISCARD:
		return parser.handleDISCARD(), nil
	case INFO:
		return parser.handleINFO(), nil
	case REPLCONF:
		return parser.handleREPLCONF(), nil
	case PSYNC:
		return parser.handlePSYNC(), nil
	default:
		return returnRESPErrorString("unknown command '" + parser.command + "'"), nil
	}
}
