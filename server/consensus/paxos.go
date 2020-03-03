package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"container/list"
	"encoding/json"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

// apologies for committing this blunder :P
var (
	acceptedMsgTimeout bool
	timerStarted       bool
)

// SendPrepare sends the prepare messages to all the other clients of the ecosystem
// This is invoked once a client tries to initiate a transaction by sending it to the server
func (server *Server) SendPrepare() {

}

// waitForAccepts essentially waits for accept responses from a majority of servers in the
// application ecosystem
func (server *Server) waitForAccepts() {

}

// getElected starts the election process in order for the server to get elected.
func (server *Server) getElected() {
	server.Ballot.BallotNum += 1
	// send a prepare message to all the servers
	msg := &common.Message{
		FromId: server.Id,
		Type:   common.PREPARE_MESSAGE,
		ElectionMsg: &common.ElectionMessage{
			Type:   common.PREPARE_MESSAGE,
			Ballot: server.Ballot,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.PREPARE_MESSAGE)
}

func (server *Server) broadcastMessages(msg []byte, msgType string) {
	log.WithFields(log.Fields{
		"messageType": msgType,
	}).Info("Broadcasting message")
	for _, peer := range server.Peers {
		// assume for now that the connections exist
		_, err := server.ServerConn[peer].Write(msg)
		if err != nil {
			// TODO: probably cause our connection died. Redo the topology maybe?
		}
	}
}

func (server *Server) sendAcceptMessage(conn net.Conn) {
	msg := &common.Message{
		FromId:     server.Id,
		Type:       common.ACCEPT_MESSAGE,
		TxnMessage: nil,
		ElectionMsg: &common.ElectionMessage{
			Type:   common.ACCEPT_MESSAGE,
			Ballot: nil,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.ACCEPTED_MESSAGE)
}

func (server *Server) validateBallotNumber(reqBallotNum int) bool {
	if server.Ballot.BallotNum == reqBallotNum {
		return true
	}
	return false
}

func (server *Server) waitForAcceptedMessages() {
	timer := time.NewTimer(5 * time.Second)
	timerStarted = true
	select {
	case <-timer.C:
		acceptedMsgTimeout = true
		timerStarted = false
	}
}

func (server *Server) processPeerLocalLogs(conn net.Conn, logs []*common.AcceptedMessage) {
	block := &common.Block{
		SeqNum:       server.SeqNum + 1,
		Transactions: list.New(),
	}
	for _, localLog := range logs {
		innerList := list.New()
		for _, elem := range localLog.Txns {
			innerList.PushBack(elem)
		}
		block.Transactions.PushBack(innerList)
	}
	block.Transactions.PushBack(server.Log)
	log.WithFields(log.Fields{
		"blockchain": utils.GetLocalLogPrint(server.Log),
		"id":         server.Id,
	}).Info("current state of the server")

	txnArr := utils.ListToArrays(block.Transactions)
	msg := common.Message{
		Type: common.COMMIT_MESSAGE,
		BlockMessage: &common.BlockMessage{
			SeqNum: block.SeqNum,
			Txns:   txnArr,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.COMMIT_MESSAGE)
}

// processPrepareMessage allows the server to decide if it should
// elect a new leader and join its ballot.
func (server *Server) processPrepareMessage(conn net.Conn, msg *common.Message) {
	if msg.ElectionMsg.Ballot.BallotNum >= server.Ballot.BallotNum {
		log.WithFields(log.Fields{
			"current Ballot Number": server.Ballot.BallotNum,
			"new Ballot Number":     msg.ElectionMsg.Ballot.BallotNum,
			"Id":                    server.Id,
		}).Info("received prepare request from a higher ballot number")

		server.Ballot.BallotNum = msg.ElectionMsg.Ballot.BallotNum
		ackMsg := common.Message{
			FromId: server.Id,
			Type:   common.ELECTION_ACK_MESSAGE,
			ElectionMsg: &common.ElectionMessage{
				Type:   common.ELECTION_ACK_MESSAGE,
				Ballot: msg.ElectionMsg.Ballot,
			},
		}
		jAckMsg, _ := json.Marshal(ackMsg)
		_, err := server.ServerConn[msg.FromId].Write(jAckMsg)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("error writing the ACK message back to the server(who wants to be a leader)")
		}
	} else {
		log.WithFields(log.Fields{
			"current Ballot Number": server.Ballot.BallotNum,
			"Id":                    server.Id,
		}).Info("received prepare request from a lower ballot number")
	}
}

func (server *Server) sendAllLocalLogs(conn net.Conn) {
	logs := utils.LogToArray(server.Log)
	accMsg := &common.AcceptedMessage{
		Txns: logs,
		// TODO: which ballot number is this? are we sure its the peer's ballot number since we
		//assume that post election, both the leader and peer ballot number would be the same?
		Ballot: server.Ballot,
		//SeqNum: 0, // TODO: Do we need this?
	}
	msg := &common.Message{
		FromId:          server.Id,
		Type:            common.ACCEPTED_MESSAGE,
		AcceptedMessage: accMsg,
	}
	jMsg, _ := json.Marshal(msg)
	_, _ = conn.Write(jMsg)
}

func (server *Server) updateBlockchain(msg *common.BlockMessage) {
	l := list.New()
	for _, txn := range msg.Txns {
		l.PushBack(txn)
	}
	block := &common.Block{
		SeqNum:       msg.SeqNum,
		Transactions: l,
	}
	server.Blockchain.PushBack(block)
}

// execPaxosRun initiates a PAXOS run and then adds the transaction to the local block chain
func (server *Server) execPaxosRun(txn *common.TransferTxn) {
	// 1. perform leader election
	server.getElected()
}
