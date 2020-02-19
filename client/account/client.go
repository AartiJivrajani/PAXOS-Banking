package account

import (
	"PAXOS-Banking/common"
	"encoding/json"
	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

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
	ClientAccount.StartTransactions()
}

// SendRequestToServer sends the request to server over UDP and also
// starts a timer. If the timer times out in `MAX_CLIENT_TIMEOUT`, sleep for `WAIT_SECONDS`
// and send the request again
// TODO: Implement timeout and retry logic as mentioned in the comment above
func (client *Client) SendRequestToServer(request *common.Message) {
	// send the request to server as is. The server can decode
	// the type of request and process it further
	PORT := ":" + strconv.Itoa(common.ServerPortMap[client.Id])
	conn, err := net.Dial("tcp", PORT)

	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"clientId": client.Id,
		}).Error("error connecting to the server")
		return
	}
	jReq, err := json.Marshal(request)

	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"clientId": client.Id,
			"request":  request,
		}).Error("error marshalling request message")
		return
	}
	_, err = conn.Write(jReq)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"clientId": client.Id,
		}).Error("error sending request to the server")
	}
	return
}
