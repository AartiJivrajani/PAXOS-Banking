package common

import "container/list"

const (
	TRANSACTION_MESSAGE     string = "Transaction"
	PREPARE_MESSAGE         string = "Prepare"
	SHOW_LOG_MESSAGE        string = "Show Log"
	SHOW_BLOCKCHAIN_MESSAGE string = "Show Blockchain"
	SHOW_BALANCE            string = "Show Balance"
	EXIT_MESSAGE            string = "Exit"
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
	SeqNum       int        `json:"seq_num"`
	Transactions *list.List `json:"transactions"`
}

type Response struct {
	MessageType string `json:"message_type"`
	Balance     int    `json:"balance,omitempty"`
	ClientId    int    `json:"client_id"`
	ToBePrinted string `json:"to_be_printed,omitempty"`
}
