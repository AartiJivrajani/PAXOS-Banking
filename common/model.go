package common

import "container/list"

const (
	TRANSACTION_MESSAGE     string = "Transaction"
	PREPARE_MESSAGE         string = "Prepare"
	SHOW_LOG_MESSAGE        string = "Show Log"
	SHOW_BLOCKCHAIN_MESSAGE string = "Show Blockchain"
	SHOW_BALANCE            string = "Show Balance"
	EXIT_MESSAGE            string = "Exit"
	RECONCILE_REQ_MESSAGE   string = "Reconcile Request"
	RECONCILE_SEQ_NUMBERS   string = "Reconcile Seq Numbers"
)

type TransferTxn struct {
	Sender int `json:"sender"`
	Recvr  int `json:"receiver"`
	Amount int `json:"amount"`
}

type ReconcileSeqMessage struct {
	Id                  int   `json:"id"`
	ReconcileSeqNumbers []int `json:"reconcile_seq_numbers"`
}

type PrepareMessage struct {
}

type Message struct {
	Type                string               `json:"message_type"`
	TxnMessage          *TransferTxn         `json:"transaction_message,omitempty"`
	PrepareMsg          *PrepareMessage      `json:"prepare_message,omitempty"`
	ReconcileSeqMessage *ReconcileSeqMessage `json:"reconcile_sequence_numbers,omitempty"`
}

type Block struct {
	SeqNum int `json:"seq_num"`
	// each element in this list of lists is a transfer transaction
	Transactions *list.List `json:"transactions"`
}

type Response struct {
	MessageType string `json:"message_type"`
	Balance     int    `json:"balance,omitempty"`
	ClientId    int    `json:"client_id"`
	ToBePrinted string `json:"to_be_printed,omitempty"`
}

// structures required to convert a list to array and vice-versa
type BlockArr struct {
	SeqNum       int     `json:"seq_num"`
	Transactions [][]int `json:"transactions"`
}

type BlockArrChain struct {
	Chain []*BlockArr
}
