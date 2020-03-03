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
	SERVER_TXN_COMPLETE     string = "Server Side Txn Complete"
	ELECTION_ACK_MESSAGE    string = "Leader Election Ack Message"
	ACCEPT_MESSAGE          string = "Election Accept Message"
	ACCEPTED_MESSAGE        string = "PAXOS Accepted Message"
	COMMIT_MESSAGE          string = "New Block Message"
)

type TransferTxn struct {
	Sender int `json:"sender"`
	Recvr  int `json:"receiver"`
	Amount int `json:"amount"`
}

type AcceptedMessage struct {
	Txns   []*TransferTxn `json:"transactions"`
	SeqNum int            `json:"seq_num,omitempty"`
	Ballot *Ballot        `json:"ballot"`
}

type ReconcileSeqMessage struct {
	Id                  int   `json:"id"`
	ReconcileSeqNumbers []int `json:"reconcile_seq_numbers"`
}

type Ballot struct {
	BallotNum int `json:"ballot_number"`
	ProcessId int `json:"process_id"`
}

type ElectionMessage struct {
	FromId int     `json:"from_id"`
	Type   string  `json:"message_type"`
	Ballot *Ballot `json:"ballot"`
}

type BlockMessage struct {
	SeqNum int `json:"seq_num"`
	// TODO: fix me. Make this []*common.TransferTxns
	Txns []*TransferTxn `json:"txns"`
}

type Message struct {
	Type                string               `json:"message_type"`
	TxnMessage          *TransferTxn         `json:"transaction_message,omitempty"`
	ElectionMsg         *ElectionMessage     `json:"election_message,omitempty"`
	ReconcileSeqMessage *ReconcileSeqMessage `json:"reconcile_sequence_numbers,omitempty"`
	AcceptedMessage     *AcceptedMessage     `json:"accepted_message,omitempty"`
	BlockMessage        *BlockMessage        `json:"block_message,omitempty"`
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
