package consensus

import (
	"PAXOS-Banking/common"
	"container/list"
)

// getBalance gets the balance from the transactions in the block chain
// this does not account for the uncommitted transactions in the local log
func (server *Server) getBalance() int {
	balance := 100
	for block := server.Blockchain.Front(); block != nil; block = block.Next() {
		txns := block.Value.(*list.List)
		// each block has multiple transactions
		for txn := txns.Front(); txn != nil; txn = txn.Next() {
			if txn.Value.(*common.TransferTxn).Recvr == server.AssociatedClient {
				balance += txn.Value.(*common.TransferTxn).Amount
			}
			if txn.Value.(*common.TransferTxn).Sender == server.AssociatedClient {
				balance -= txn.Value.(*common.TransferTxn).Amount
			}
		}
	}
	return balance
}

// getLocalBalance gets the balance from transactions present in both -
// 1. The server's local block chain
// 2. The server's local log
func (server *Server) getLocalBalance() int {
	balance := server.getBalance()

	for block := server.Log.Front(); block != nil; block = block.Next() {
		if block.Value.(*common.TransferTxn).Recvr == server.AssociatedClient {
			balance += block.Value.(*common.TransferTxn).Amount
		}
		if block.Value.(*common.TransferTxn).Sender == server.AssociatedClient {
			balance -= block.Value.(*common.TransferTxn).Amount
		}
	}
	return balance
}

// checkIfTxnPossible fetches the client's current balance from the server's local
// block chain. If this balance is greater than the amount to be transacted, a PAXOS run
// is not required, else, it is
func (server *Server) checkIfTxnPossible(txn *common.TransferTxn) bool {
	balance := server.getBalance()
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
