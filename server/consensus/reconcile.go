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
	seqNumbers := make([]int, 0)
	for _, block := range server.Blockchain {
		seqNumbers = append(seqNumbers, block.SeqNum)
	}
	resp := &common.Message{
		FromId: server.Id,
		Type:   common.RECONCILE_SEQ_NUMBERS,
		ReconcileSeqMessage: &common.ReconcileSeqMessage{
			Id:                  server.Id,
			ReconcileSeqNumbers: seqNumbers,
		},
	}
	jResp, _ := json.Marshal(resp)
	_, _ = conn.Write(jResp)
}

func (server *Server) sendReconcileRequest() {
	var (
		request *common.Message
		err     error
	)
	for _, peer := range server.Peers {
		if server.ServerConn[peer] != nil {
			request = &common.Message{
				FromId: server.Id,
				Type:   common.RECONCILE_REQ_MESSAGE,
			}
			jReq, _ := json.Marshal(request)
			_, err = server.ServerConn[peer].Write(jReq)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
					"peer":  peer,
				}).Info("error writing reconcile request message to the peer")
			}
		} else {
			//TODO: Take care of the case when the connections are down and new connections need to be
			// established before continuing
		}
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
