package common

import "container/list"

type TransferTxn struct {
	Sender int `json:"sender"`
	Recvr  int `json:"receiver"`
	Amount int `json:"amount"`
}

type Block struct {
	SeqNum       int       `json:"seq_num"`
	Transactions list.List `json:"transactions"`
}

type Server struct {
	Id   int       `json:"id"`
	Log  list.List `json:"log"`
	Port int       `json:"port"`
}

type Client struct {
	Id   int `json:"id"`
	Port int `json:"port"`
}
