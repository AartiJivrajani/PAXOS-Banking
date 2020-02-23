package utils

import (
	"PAXOS-Banking/common"
	"container/list"
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

var (
	stars  = "*********************************************************************************************"
	dashes = "============================================================================================="
)

func ConfigureLogger(level string) {
	log.SetOutput(os.Stderr)
	switch strings.ToLower(level) {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning", "warn":
		log.SetLevel(log.WarnLevel)
	}
}

func GetBlockchainPrint(blockchain *list.List) string {
	return ""
}

// GetLocalLogPrint returns a string ready to print.
// this string has the format - Sender:Recvr:Amt -> Sender:Recvr:Amt
func GetLocalLogPrint(log *list.List) string {
	var l string
	for block := log.Front(); block != nil; block = block.Next() {
		l = l + strconv.Itoa(block.Value.(*common.TransferTxn).Sender) + ":" + strconv.Itoa(block.Value.(*common.TransferTxn).Recvr) +
			":" + fmt.Sprintf("%g", block.Value.(*common.TransferTxn).Amount)
		if block.Next() != nil {
			l = l + "->"
		}
	}
	return l
}

func PrettyPrint(str string) {
	log.Info(stars)
	log.WithFields(log.Fields{
		"message": str,
	}).Info("local log")
	log.Info(stars)
}

// ListToArrays creates an array of arrays for all the blockchain transactions.
// Something like this - [[sender, recrv, amount], [sender, recvr, amount]]
func ListToArrays(l *list.List) [][]int {
	finalList := make([][]int, 0)
	for block := l.Front(); block != nil; block = block.Next() {
		txns := block.Value.(*list.List)
		innerList := make([]int, 0)
		for txn := txns.Front(); txn != nil; txn = txn.Next() {
			innerList = append(innerList,
				txn.Value.(*common.TransferTxn).Sender,
				txn.Value.(*common.TransferTxn).Recvr,
				txn.Value.(*common.TransferTxn).Amount)
		}
		finalList = append(finalList, innerList)
	}
	return finalList
}

func GetBlockChainFromArr(chain *common.BlockArrChain) *list.List {
	finalList := list.New()
	for _, block := range chain.Chain {
		innerList := list.New()
		var blockChainBlock *common.Block
		for _, txn := range block.Transactions {
			// add the sender, receiver, amount to the list of transactions
			txn := &common.TransferTxn{
				Sender: txn[0],
				Recvr:  txn[1],
				Amount: txn[2],
			}
			innerList.PushBack(txn)
		}
		blockChainBlock = &common.Block{
			SeqNum:       block.SeqNum,
			Transactions: innerList,
		}
		finalList.PushBack(blockChainBlock)
	}
	return finalList
}
