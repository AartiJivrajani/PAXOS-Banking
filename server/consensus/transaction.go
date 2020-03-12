package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
)

// getBalance gets the balance from the transactions in the block chain
// this does not account for the uncommitted transactions in the local log
func (server *Server) getBalance() int {
	balance := 10
	for _, block := range server.Blockchain {
		for _, txn := range block.Transactions {
			if txn.Recvr == server.Id {
				balance += txn.Amount
			}
			if txn.Sender == server.Id {
				balance -= txn.Amount
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
	for _, txn := range server.Log {
		if txn.Recvr == server.Id {
			balance += txn.Amount
		}
		if txn.Sender == server.Id {
			balance -= txn.Amount
		}
	}
	return balance
}

// checkIfTxnPossible fetches the client's current balance from the server's local log AND
// block chain. If this balance is greater than the amount to be transacted, a PAXOS run
// is not required, else, it is
func (server *Server) checkIfTxnPossible(txn *common.TransferTxn) bool {
	blockchainPrint := utils.GetBlockchainPrint(server.Blockchain)
	balance := server.getLocalBalance()
	log.WithFields(log.Fields{
		"currentLocalLog": utils.GetLocalLogPrint(server.Log),
		"blockchain":      blockchainPrint,
		"newTxn":          txn.Amount,
		"balance":         balance,
	}).Info("checking transaction possibility.....")
	if balance < txn.Amount {
		log.Error("transaction not possible")
		return false
	} else {
		return true
	}
}

// execLocalTxn carries out the transaction locally and saves the record in the local blockchain
func (server *Server) execLocalTxn(txn *common.TransferTxn) {
	server.Log = append(server.Log, txn)
	localLog, _ := json.Marshal(server.Log)
	server.RedisConn.Set(fmt.Sprintf(common.REDIS_LOG_KEY, server.Id), localLog, 0)
}
