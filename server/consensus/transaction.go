package consensus

import "PAXOS-Banking/common"

// checkIfTxnPossible fetches the client's current balance from the server's local
// block chain. If this balance is greater than the amount to be transacted, a PAXOS run
// is not required, else, it is
func (server *Server) checkIfTxnPossible(txn *common.TransferTxn) bool {
	// the initial balance of each client is $100
	balance := 100
	for block := server.Blockchain.Front(); block != nil; block = block.Next() {
		if block.Value.(*common.TransferTxn).Recvr == server.AssociatedClient {
			balance += block.Value.(*common.TransferTxn).Amount
		}
		if block.Value.(*common.TransferTxn).Sender == server.AssociatedClient {
			balance -= block.Value.(*common.TransferTxn).Amount
		}
	}
	if balance > txn.Amount {
		return false
	} else {
		return true
	}
}

// execLocalTxn carries out the transaction locally and saves the record in the local blockchain
func (server *Server) execLocalTxn(txn *common.TransferTxn) {

}

// execPaxosRun initiates a PAXOS run and then adds the transaction to the local blockchain
func (server *Server) execPaxosRun(txn *common.TransferTxn) {
	server.getElected()
}
