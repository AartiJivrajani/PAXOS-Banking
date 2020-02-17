package common

const (
	FaultsAllowed        int     = 1
	LEADER               string  = "leader"
	FOLLOWER             string  = "follower"
	MaxReconnectAttempts float64 = 3
	// max number of seconds the client should wait for response from the server
	MAX_CLIENT_TIMEOUT int = 5
	// number of seconds the client should sleep before initiating a server request once again
	WAIT_SECONDS int = 10
)

var ClientPortMap = map[int]int{
	1: 7001,
	2: 7002,
	3: 7003,
}
var ServerPortMap = map[int]int{
	1: 8001,
	2: 8002,
	3: 8003,
}
