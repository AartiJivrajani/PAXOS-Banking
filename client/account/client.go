package account

import "PAXOS-Banking/common"

type Client struct {
	Id   int `json:"id"`
	Port int `json:"port"`
}

var ClientAccount *Client

func StartClient(id int) {
	ClientAccount = &Client{
		Id:   id,
		Port: common.ClientPortMap[id],
	}
}

// SendRequestToServer sends the request to server over HTTP and also
// starts a timer. If the timer times out in `MAX_CLIENT_TIMEOUT`, sleep for `WAIT_SECONDS`
// and send the request again
func SendRequestToServer(request *common.Message) {
	// send the request to server as is. The server can decode
	// the type of request and process it further
}
