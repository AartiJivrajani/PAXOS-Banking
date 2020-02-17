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
