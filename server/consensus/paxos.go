package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

// apologies for committing this blunder :P
var (
	acceptedMsgTimeout = false
	timerStarted       = false
	clientTxn          *common.TransferTxn
)

// getElected starts the election process in order for the server to get elected.
func (server *Server) getElected() {
	server.Ballot.BallotNum += 1
	// send a prepare message to all the servers
	msg := &common.Message{
		FromId: server.Id,
		Type:   common.ELECTION_PREPARE_MESSAGE,
		ElectionMsg: &common.ElectionMessage{
			Type:   common.ELECTION_PREPARE_MESSAGE,
			Ballot: server.Ballot,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.ELECTION_PREPARE_MESSAGE)
}

func (server *Server) writeToServer(toServer int, msg []byte, messageType string) {
	var (
		err error
	)
	if _, OK := server.ServerConn[toServer]; !OK {
		log.WithFields(log.Fields{
			"fromServerId": server.Id,
			"toServerId":   toServer,
		}).Error("connection obj not found, unable to write")
		return
	}
	_, err = server.ServerConn[toServer].Write(msg)
	if err != nil {
		log.WithFields(log.Fields{
			"error":       err.Error(),
			"toServer":    toServer,
			"messageType": messageType,
		}).Error("error writing message to server")
		delete(server.ServerConn, toServer)
		//go server.reconnectToServer(toServer)
	}
}

func (server *Server) broadcastMessages(msg []byte, msgType string) {
	log.WithFields(log.Fields{
		"messageType": msgType,
	}).Info("Broadcasting message")
	for _, peer := range server.Peers {
		server.writeToServer(peer, msg, msgType)
	}
}

func (server *Server) sendAcceptMessage() {
	msg := &common.Message{
		FromId:     server.Id,
		Type:       common.ELECTION_ACCEPT_MESSAGE,
		TxnMessage: nil,
		ElectionMsg: &common.ElectionMessage{
			Type:   common.ELECTION_ACCEPT_MESSAGE,
			Ballot: nil,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.PaxosState = 3
	log.WithFields(log.Fields{
		"new paxos state": server.PaxosState,
	}).Info("Changed to new paxos state")
	server.broadcastMessages(jMsg, common.ELECTION_ACCEPT_MESSAGE)
}

func (server *Server) validateBallotNumber(reqBallotNum int) bool {
	if server.Ballot.BallotNum == reqBallotNum {
		return true
	}
	return false
}

func (server *Server) waitForAcceptedMessages() {
	timer := time.NewTimer(10 * time.Second)
	timerStarted = true
	log.Info("started timer(waiting for all ACCEPTED messages)")
	for {
		select {
		case <-timer.C:
			acceptedMsgTimeout = true
			timerStarted = false
			break
		}
	}
}

func (server *Server) processPeerLocalLogs(logs []*common.AcceptedMessage) {
	// before creating and adding a new block, increment the seq number
	server.SeqNum += 1 // TODO: [Aarti] - Check this logic with Rakshith once

	log.WithFields(log.Fields{
		"new seq number": server.SeqNum,
	}).Info("INCREMENTING SEQUENCE NUMBER")

	block := &common.Block{
		SeqNum:       server.SeqNum,
		Transactions: make([]*common.TransferTxn, 0),
	}
	for _, localTxns := range logs {
		for _, txn := range localTxns.Txns {
			block.Transactions = append(block.Transactions, txn)
		}
	}
	// TODO: Confirm this logic once?
	// add the server's own local log as well
	for _, txn := range server.Log {
		block.Transactions = append(block.Transactions, txn)
	}
	log.WithFields(log.Fields{
		"local log": utils.GetLocalLogPrint(server.Log),
		"id":        server.Id,
	}).Info("current local state of the server, after paxos")

	// evict the local transactions of the leader now, since they are already in a block.
	server.Log = nil
	server.Log = make([]*common.TransferTxn, 0)

	msg := common.Message{
		FromId:       server.Id,
		Type:         common.COMMIT_MESSAGE,
		BlockMessage: block,
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.COMMIT_MESSAGE)
	// TODO: !!! Confirm this logic once
	// add the newly created block to the server itself.
	server.Blockchain = append(server.Blockchain, block)

	server.updateRedisData()
	// PHEW! PAXOS IS DONE! finally, send a response to the client.
	server.sendResponseToClientAfterPaxos()
}

func (server *Server) updateRedisData() {
	jBlockChain, _ := json.Marshal(server.Blockchain)
	_, err := server.RedisConn.Set(fmt.Sprintf(common.REDIS_BLOCKCHAIN_KEY, server.Id), jBlockChain, 0).Result()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error inserting the blockchain to redis")
	}
	jLog, _ := json.Marshal(server.Log)
	_, err = server.RedisConn.Set(fmt.Sprintf(common.REDIS_LOG_KEY, server.Id), jLog, 0).Result()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error inserting the local log to redis")
	}
}

// sendResponseToClientAfterPaxos checks if the client transaction can still be carried out.
func (server *Server) sendResponseToClientAfterPaxos() {
	var (
		clientResponse *common.Response
		jResp          []byte
	)
	server.PaxosState = 0
	log.WithFields(log.Fields{
		"new paxos state": server.PaxosState,
	}).Info("Changed to new paxos state")
	possible := server.checkIfTxnPossible(clientTxn)
	log.Info("checking in sendResponseToClientAfterPaxos")
	if possible {
		server.execLocalTxn(clientTxn)
		clientResponse = &common.Response{
			MessageType: common.SERVER_TXN_COMPLETE,
		}
	} else {
		clientResponse = &common.Response{
			MessageType: common.INSUFFICIENT_FUNDS,
		}
	}
	jResp, _ = json.Marshal(clientResponse)
	_, err := server.getClientConnection().Write(jResp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error writing message to client")
	}
	// invalidate the clientTxn after sending response to the client.
	clientTxn = nil
}

// processPrepareMessage allows the server to decide if it should
// elect a new leader and join its ballot.
func (server *Server) processPrepareMessage(msg *common.Message) {
	if msg.ElectionMsg.Ballot.BallotNum >= server.Ballot.BallotNum {
		log.WithFields(log.Fields{
			"current Ballot Number": server.Ballot.BallotNum,
			"new Ballot Number":     msg.ElectionMsg.Ballot.BallotNum,
			"Id":                    server.Id,
		}).Info("received prepare request from a higher ballot number")

		server.Ballot.BallotNum = msg.ElectionMsg.Ballot.BallotNum
		ackMsg := common.Message{
			FromId: server.Id,
			Type:   common.ELECTION_PROMISE_MESSAGE,
			ElectionMsg: &common.ElectionMessage{
				Type:   common.ELECTION_PROMISE_MESSAGE,
				Ballot: msg.ElectionMsg.Ballot,
			},
		}
		server.PaxosState = 2
		log.WithFields(log.Fields{
			"new paxos state": server.PaxosState,
		}).Info("Changed to new paxos state")
		jAckMsg, _ := json.Marshal(ackMsg)
		server.writeToServer(msg.FromId, jAckMsg, common.ELECTION_PROMISE_MESSAGE)
	} else {
		log.WithFields(log.Fields{
			"current Ballot Number": server.Ballot.BallotNum,
			"Id":                    server.Id,
		}).Info("received prepare request from a lower ballot number")
	}
}

func (server *Server) sendAllLocalLogs(msg *common.Message) {
	accMsg := &common.AcceptedMessage{
		Txns: server.Log,
		// TODO: which ballot number is this? are we sure its the peer's ballot number since we
		// assume that post election, both the leader and peer ballot number would be the same?
		Ballot: server.Ballot,
		//SeqNum: 0, // TODO: Do we need this?
	}
	commonMessage := &common.Message{
		FromId:          server.Id,
		Type:            common.ELECTION_ACCEPTED_MESSAGE,
		AcceptedMessage: accMsg,
	}
	jMsg, _ := json.Marshal(commonMessage)
	server.PaxosState = 4
	log.WithFields(log.Fields{
		"new paxos state": server.PaxosState,
	}).Info("Changed to new paxos state")
	server.writeToServer(msg.FromId, jMsg, common.ELECTION_ACCEPT_MESSAGE)
}

func (server *Server) updateBlockchain(msg *common.Block) {
	block := &common.Block{
		SeqNum:       msg.SeqNum,
		Transactions: msg.Transactions,
	}
	// increment the sequence number of the server
	server.SeqNum = msg.SeqNum

	log.WithFields(log.Fields{
		"existing blockchain": utils.GetBlockchainPrint(server.Blockchain),
	}).Info("blockchain before update")

	server.Blockchain = append(server.Blockchain, block)

	log.WithFields(log.Fields{
		"chain": utils.GetBlockchainPrint(server.Blockchain),
	}).Info("blockchain after update")
	// evict the local transactions now, since they were already sent to the leader and the leader
	// has sent back a block
	server.Log = nil
	server.Log = make([]*common.TransferTxn, 0)

	server.updateRedisData()
}

// execPaxosRun initiates a PAXOS run and then adds the transaction to the local block chain
func (server *Server) execPaxosRun(txn *common.TransferTxn) {
	// 1. perform leader election
	log.Info("PAXOS RUN START")
	clientTxn = txn
	server.PaxosState = 1
	log.WithFields(log.Fields{
		"new paxos state": server.PaxosState,
	}).Info("Changed to new paxos state")
	server.getElected()
}
