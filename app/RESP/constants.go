package resp

const (
	String  = '+'
	Bulk    = '$'
	Array   = '*'
	Integer = ':'
	Error   = '-'
)

const (
	GET      = "get"
	SET      = "set"
	INCR     = "incr"
	PING     = "ping"
	ECHO     = "echo"
	MULTI    = "multi"
	EXEC     = "exec"
	DISCARD  = "discard"
	INFO     = "info"
	REPLCONF = "replconf"
	PSYNC    = "psync"
)
