package common

const (
	ELECTION_PREPARE_MESSAGE  string = "PREPARE"
	ELECTION_PROMISE_MESSAGE  string = "PROMISE"
	ELECTION_ACCEPT_MESSAGE   string = "ACCEPT"
	ELECTION_ACCEPTED_MESSAGE string = "ACCEPTED"
	COMMIT_MESSAGE            string = "COMMIT"

	HEARTBEAT_PING string = "PING"
	HEARTBEAT_PONG string = "PONG"

	TRANSACTION_MESSAGE     string = "Transaction"
	SHOW_LOG_MESSAGE        string = "Show Log"
	SHOW_BLOCKCHAIN_MESSAGE string = "Show Blockchain"
	SHOW_BALANCE            string = "Show Balance"
	EXIT_MESSAGE            string = "Exit"
	RECONCILE_REQ_MESSAGE   string = "Reconcile Request"
	RECONCILE_SEQ_NUMBERS   string = "Reconcile Seq Numbers"
	SERVER_TXN_COMPLETE     string = "Server Side Txn Complete"
	INSUFFICIENT_FUNDS      string = "Insufficient Funds"
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
	Type   string  `json:"message_type"`
	Ballot *Ballot `json:"ballot"`
}

type Message struct {
	FromId              int                  `json:"from_id"`
	Type                string               `json:"message_type"`
	TxnMessage          *TransferTxn         `json:"transaction_message,omitempty"`
	ElectionMsg         *ElectionMessage     `json:"election_message,omitempty"`
	ReconcileSeqMessage *ReconcileSeqMessage `json:"reconcile_sequence_numbers,omitempty"`
	AcceptedMessage     *AcceptedMessage     `json:"accepted_message,omitempty"`
	BlockMessage        *Block               `json:"block_message,omitempty"`
}

type Block struct {
	SeqNum int `json:"seq_num"`
	// each element in this list of lists is a transfer transaction
	Transactions []*TransferTxn `json:"transactions"`
}

type Response struct {
	MessageType string `json:"message_type"`
	Balance     int    `json:"balance,omitempty"`
	ClientId    int    `json:"client_id"`
	ToBePrinted string `json:"to_be_printed,omitempty"`
}
