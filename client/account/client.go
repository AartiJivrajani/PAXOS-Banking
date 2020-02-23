package account

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

type Client struct {
	Id   int `json:"id"`
	Port int `json:"port"`
}

var (
	ClientAccount  *Client
	showNextPrompt = make(chan bool)
)

func StartClient(id int) {
	ClientAccount = &Client{
		Id:   id,
		Port: common.ClientPortMap[id],
	}
	go ClientAccount.StartResponseListener()
	go ClientAccount.StartTransactions()
}

func (client *Client) handleIncomingConnections(conn net.Conn) {
	var (
		err  error
		resp *common.Response
	)
	d := json.NewDecoder(conn)
	for {
		err = d.Decode(&resp)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("error decoding the response message received from the server")
			continue
		}
		switch resp.MessageType {
		case common.SERVER_TXN_COMPLETE:
			showNextPrompt <- true
		case common.SHOW_LOG_MESSAGE:
			utils.PrettyPrint(resp.ToBePrinted)
		case common.SHOW_BLOCKCHAIN_MESSAGE:
			utils.PrettyPrint(resp.ToBePrinted)
		case common.SHOW_BALANCE:
			utils.PrettyPrint(fmt.Sprintf("Balance: %d", resp.Balance))
		}
	}
}

func (client *Client) StartResponseListener() {
	var (
		err error
	)
	PORT := ":" + strconv.Itoa(common.ServerPortMap[client.Id])
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Error("error establishing connection to the server port, shutting down... ")
		return
	}
	for {
		c, err := listener.Accept()
		if err != nil {
			// TODO: [Aarti] Have a back off and retry in this case. Cause lets say if some server went down
			// and came back up again, we need this client listener to be alive again.
			log.Error("error starting the server listener, shutting down...")
			return
		}
		go client.handleIncomingConnections(c)
	}
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
