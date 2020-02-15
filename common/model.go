package common

import "container/list"

const (
	TRANSACTION_MESSAGE string = "Transaction"
	PREPARE_MESSAGE     string = "Prepare"
)

type TransferTxn struct {
	Sender int `json:"sender"`
	Recvr  int `json:"receiver"`
	Amount int `json:"amount"`
}

type PrepareMessage struct {
}

type Message struct {
	Type       string          `json:"message_type"`
	TxnMessage *TransferTxn    `json:"transaction_message,omitempty"`
	PrepareMsg *PrepareMessage `json:"prepare_message,omitempty"`
}

type Block struct {
	SeqNum       int       `json:"seq_num"`
	Transactions list.List `json:"transactions"`
}
