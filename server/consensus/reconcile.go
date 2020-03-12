package consensus

import (
	"PAXOS-Banking/common"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/redis.v5"
)

// handleReconcileRequestMessage returns a list of sequence numbers of the server's current block chain
func (server *Server) handleReconcileRequestMessage(destServer int) {
	// send a list of all sequence numbers
	seqNumber := server.SeqNum
	resp := &common.Message{
		FromId: server.Id,
		Type:   common.RECONCILE_SEQ_NUMBER,
		ReconcileSeqMessage: &common.ReconcileSeqMessage{
			Id:                 server.Id,
			ReconcileSeqNumber: seqNumber,
		},
	}
	jResp, _ := json.Marshal(resp)
	server.writeToServer(destServer, jResp, common.RECONCILE_SEQ_NUMBER)
}

func (server *Server) sendReconcileRequest() {
	var (
		request *common.Message
	)
	for _, peer := range server.Peers {
		request = &common.Message{
			FromId: server.Id,
			Type:   common.RECONCILE_REQ_MESSAGE,
		}
		jReq, _ := json.Marshal(request)
		log.WithFields(log.Fields{
			"toServer": peer,
		}).Info("sending reconcile request to....")
		server.writeToServer(peer, jReq, common.RECONCILE_REQ_MESSAGE)
	}
}

func (server *Server) reconcile(blockChainVal string, localLogVal string) {
	var (
		blockChain []*common.Block
		localLog   []*common.TransferTxn
		err        error
	)
	log.Info("uh-oh! Reconciliation needed :O")
	err = json.Unmarshal([]byte(blockChainVal), &blockChain)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error unmarshalling blockchain data received from redis")
		return
	} else {
		server.Blockchain = blockChain
	}

	err = json.Unmarshal([]byte(localLogVal), &localLog)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error unmarshalling local log data received from redis")
	} else {
		server.Log = localLog
	}

	server.sendReconcileRequest()
	seqNumbers := <-waitForReconcileResponse
	server.handleReconciliation(seqNumbers)
}

func (server *Server) handleReconciliation(msg []*common.ReconcileSeqMessage) {
	log.Info("handling reconciliation.....")
	var (
		maxSeqNum         int
		maxSeqNumServerId int
		request           *common.Message
	)
	for _, seqMsg := range msg {
		if seqMsg.ReconcileSeqNumber > maxSeqNum {
			maxSeqNum = seqMsg.ReconcileSeqNumber
			maxSeqNumServerId = seqMsg.Id
		}
	}
	log.WithFields(log.Fields{
		"serverId": maxSeqNumServerId,
	}).Info("server with MAX blockchain seq number(for reconciliation)")

	request = &common.Message{
		FromId: server.Id,
		Type:   common.RECONCILE_BLOCKCHAIN_REQUEST,
	}
	jReq, _ := json.Marshal(request)
	server.writeToServer(maxSeqNumServerId, jReq, common.RECONCILE_BLOCKCHAIN_REQUEST)
}

func (server *Server) sendBlockchain(destServer int) {
	// send my current blockchain
	resp := &common.Message{
		FromId:     server.Id,
		Type:       common.RECONCILE_BLOCKCHAIN_RESPONSE,
		Blockchain: server.Blockchain,
	}

	jMsg, _ := json.Marshal(resp)
	server.writeToServer(destServer, jMsg, common.RECONCILE_BLOCKCHAIN_RESPONSE)
}

func (server *Server) receiveBlockchain(blkchain []*common.Block) {
	log.WithFields(log.Fields{
		"blockchain": blkchain,
	}).Info("blockchain received from peer upon reconciliation")

	server.Blockchain = blkchain
}

// checkAndReconcile first checks if the server is a zombie or a baby process
// a zombie process is a server which crashed/failed and came back up. In such a case, the following is done -
// 1. A Redis lookup for the key: SERVER-BLOCKCHAIN-<id>
// 2. If this key is found, the reconciliation algorithm is invoked which helps the server
//    build its block chain and local log once again.
// a baby process is a process which just started, and has no history of storing any log/blockchain previously
func (server *Server) checkAndReconcile() {
	var (
		blockChain, localLog string
		err                  error
		reconcile            = false
	)
	log.Info("Let's see if we need to reconcile!")
	// redis lookup
	blockChain, err = server.RedisConn.Get(fmt.Sprintf(common.REDIS_BLOCKCHAIN_KEY, server.Id)).Result()
	if err == redis.Nil {
		log.Info("no blockchain found for SERVER-BLOCKCHAIN-KEY")
	} else {
		reconcile = true
	}
	localLog, err = server.RedisConn.Get(fmt.Sprintf(common.REDIS_LOG_KEY, server.Id)).Result()
	if err == redis.Nil {
		log.Info("no local log found for SERVER-LOG-KEY")
	} else {
		reconcile = true
	}
	if reconcile {
		server.reconcile(blockChain, localLog)
	}
	return
}
