package utils

import (
	"PAXOS-Banking/common"
	"fmt"
	//"Replicated-Blockchain/common"
	"container/list"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

var (
	Stars  = "*********************************************************************************************"
	Dashes = "============================================================================================="
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
	log.Info(Stars)
	log.WithFields(log.Fields{
		"localLog": str,
	}).Info("local log")
	log.Info(Stars)
}
