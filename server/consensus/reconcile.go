package consensus

import (
	"PAXOS-Banking/common"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/redis.v5"
)

// handleReconcileRequestMessage returns a list of sequence numbers of the server's current block chain
func (server *Server) handleReconcileRequestMessage(conn net.Conn) {
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
	_, _ = conn.Write(jResp)
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
		server.writeToServer(peer, jReq, common.RECONCILE_REQ_MESSAGE)
	}
}

func (server *Server) reconcile(blockChainVal string, localLogVal string) {
	var (
		blockChain []*common.Block
		localLog   []*common.TransferTxn
		err        error
	)
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
	request = &common.Message{
		FromId: server.Id,
		Type:   common.RECONCILE_BLOCKCHAIN_REQUEST,
	}
	jReq, _ := json.Marshal(request)
	server.writeToServer(maxSeqNumServerId, jReq, common.RECONCILE_BLOCKCHAIN_REQUEST)

}

func (server *Server) sendBlockchain(conn net.Conn) {
	// send my current blockchain
	resp := &common.Message{
		FromId:     server.Id,
		Type:       common.RECONCILE_BLOCKCHAIN_RESPONSE,
		Blockchain: server.Blockchain,
	}

	jMsg, _ := json.Marshal(resp)
	_, _ = conn.Write(jMsg)

}

func (server *Server) receiveBlockchain(blkchain []*common.Block) {
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
	)
	// redis lookup
	blockChain, err = server.RedisConn.Get(fmt.Sprintf(common.REDIS_BLOCKCHAIN_KEY, server.Id)).Result()
	if err == redis.Nil {
		return
	} else {
		localLog, err = server.RedisConn.Get(fmt.Sprintf(common.REDIS_LOG_KEY, server.Id)).Result()
		if err == redis.Nil {
			return
		}
		server.reconcile(blockChain, localLog)
		return
	}
}
