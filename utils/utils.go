package utils

import (
	"PAXOS-Banking/common"
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

func GetBlockchainPrint(blockchain []*common.Block) string {
	var l string
	for i, block := range blockchain {
		l += "["
		for j, txn := range block.Transactions {
			l = l + "(" + strconv.Itoa(txn.Sender) + ":" + strconv.Itoa(txn.Recvr) + ":" +
				strconv.Itoa(txn.Amount) + ")"
			if j < len(block.Transactions)-1 {
				l = l + "->"
			}
		}
		l += "]"
		if i < len(blockchain)-1 {
			l = l + "->"
		}
	}
	return l
}

// GetLocalLogPrint returns a string ready to print.
// this string has the format - Sender:Recvr:Amt -> Sender:Recvr:Amt
func GetLocalLogPrint(log []*common.TransferTxn) string {
	var l string
	for index, txn := range log {
		l = l + "(" + strconv.Itoa(txn.Sender) + ":" + strconv.Itoa(txn.Recvr) +
			":" + strconv.Itoa(txn.Amount) + ")"
		if index < len(log)-1 {
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

//func GetBlockChainFromArr(chain *common.BlockArrChain) *list.List {
//	finalList := list.New()
//	for _, block := range chain.Chain {
//		innerList := list.New()
//		var blockChainBlock *common.Block
//		for _, txn := range block.Transactions {
//			// add the sender, receiver, amount to the list of transactions
//			txn := &common.TransferTxn{
//				Sender: txn[0],
//				Recvr:  txn[1],
//				Amount: txn[2],
//			}
//			innerList.PushBack(txn)
//		}
//		blockChainBlock = &common.Block{
//			SeqNum:       block.SeqNum,
//			Transactions: innerList,
//		}
//		finalList.PushBack(blockChainBlock)
//	}
//	return finalList
//}

//func LogToArray(l *list.List) []*common.TransferTxn {
//	finalArr := make([]*common.TransferTxn, 0)
//	for block := l.Front(); block != nil; block = block.Next() {
//		finalArr = append(finalArr, block.Value.(*common.TransferTxn))
//	}
//	log.WithFields(log.Fields{
//		"list": GetLocalLogPrint(l),
//	}).Info("original list")
//	log.WithFields(log.Fields{
//		"array": finalArr,
//	}).Info("converted log to array")
//	return finalArr
//}
