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
	msg := &common.ElectionMessage{
		Type:   common.PREPARE_MESSAGE,
		Ballot: server.Ballot,
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg)
}

func (server *Server) broadcastMessages(msg []byte) {
	for _, peer := range server.Peers {
		// assume for now that the connections exist
		_, err := server.ServerConn[peer].Write(msg)
		if err != nil {
			// TODO: probably cause our connection died. Redo the topology maybe?
		}
	}
}

func (server *Server) sendAcceptMessage(conn net.Conn) {
	msg := &common.ElectionMessage{
		Type:   common.ACCEPT_MESSAGE,
		Ballot: nil,
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg)
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
	log.WithFields(log.Fields{
		"blockchain": utils.GetLocalLogPrint(server.Log),
		"id":         server.Id,
	}).Info("current state of the server")

	txnArr := utils.ListToArrays(block.Transactions)
	msg := common.Message{
		Type:                common.COMMIT_MESSAGE,
		TxnMessage:          nil,
		ElectionMsg:         nil,
		ReconcileSeqMessage: nil,
		AcceptedMessage:     nil,
		BlockMessage: &common.BlockMessage{
			SeqNum: block.SeqNum,
			Txns:   txnArr,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg)
}

// processElectionRequest allows the server to decide if it should
// elect a new leader and join its ballot
func (server *Server) processElectionRequest(conn net.Conn, msg *common.ElectionMessage) {
	if msg.Ballot.BallotNum >= server.Ballot.BallotNum {

		log.WithFields(log.Fields{
			"current Ballot Number": server.Ballot.BallotNum,
			"new Ballot Number":     msg.Ballot.BallotNum,
			"Id":                    server.Id,
		}).Info("received prepare request from a higher ballot number")

		server.Ballot.BallotNum = msg.Ballot.BallotNum
		ackMsg := common.ElectionMessage{
			Type:   common.ELECTION_ACK_MESSAGE,
			Ballot: msg.Ballot,
		}
		jAckMsg, _ := json.Marshal(ackMsg)
		_, _ = conn.Write(jAckMsg)
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
		Type:            common.ACCEPTED_MESSAGE,
		AcceptedMessage: accMsg,
	}
	jMsg, _ := json.Marshal(msg)
	_, _ = conn.Write(jMsg)
}

func (server *Server) updateBlockchain(msg *common.BlockMessage) {

}
