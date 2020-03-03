package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"encoding/json"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

// apologies for committing this blunder :P
var (
	acceptedMsgTimeout = false
	timerStarted       = false
	clientTxn          *common.TransferTxn
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
		Type:   common.ELECTION_PREPARE_MESSAGE,
		ElectionMsg: &common.ElectionMessage{
			Type:   common.ELECTION_PREPARE_MESSAGE,
			Ballot: server.Ballot,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.ELECTION_PREPARE_MESSAGE)
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
	select {
	case <-timer.C:
		acceptedMsgTimeout = true
		timerStarted = false
	}
}

func (server *Server) processPeerLocalLogs(logs []*common.AcceptedMessage) {
	block := &common.Block{
		SeqNum:       server.SeqNum + 1,
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

	msg := common.Message{
		Type: common.COMMIT_MESSAGE,
		BlockMessage: &common.BlockMessage{
			SeqNum: server.SeqNum + 1,
			Txns:   block.Transactions,
		},
	}
	jMsg, _ := json.Marshal(msg)
	server.broadcastMessages(jMsg, common.COMMIT_MESSAGE)
	// TODO: !!! Confirm this logic once
	// add the newly created block to the server itself.
	server.Blockchain = append(server.Blockchain, block)
	// PHEW! PAXOS IS DONE! finally, send a response to the client.
	server.sendResponseToClientAfterPaxos()
}

// sendResponseToClientAfterPaxos checks if the client transaction can still be carried out.
func (server *Server) sendResponseToClientAfterPaxos() {
	var (
		clientResponse *common.Response
		jResp          []byte
	)
	// evict the local transactions now, since they are already in a block.
	server.Log = nil
	server.Log = make([]*common.TransferTxn, 0)
	possible := server.checkIfTxnPossible(clientTxn)
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
	_, _ = server.getClientConnection().Write(jResp)
	// invalidate the clientTxn after sending response to the client.
	clientTxn = nil
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
			Type:   common.ELECTION_PROMISE_MESSAGE,
			ElectionMsg: &common.ElectionMessage{
				Type:   common.ELECTION_PROMISE_MESSAGE,
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

func (server *Server) sendAllLocalLogs(msg *common.Message) {
	accMsg := &common.AcceptedMessage{
		Txns: server.Log,
		// TODO: which ballot number is this? are we sure its the peer's ballot number since we
		//assume that post election, both the leader and peer ballot number would be the same?
		Ballot: server.Ballot,
		//SeqNum: 0, // TODO: Do we need this?
	}
	commonMessage := &common.Message{
		FromId:          server.Id,
		Type:            common.ELECTION_ACCEPTED_MESSAGE,
		AcceptedMessage: accMsg,
	}
	jMsg, _ := json.Marshal(commonMessage)
	_, _ = server.ServerConn[msg.FromId].Write(jMsg)
}

func (server *Server) updateBlockchain(msg *common.BlockMessage) {
	block := &common.Block{
		SeqNum:       msg.SeqNum,
		Transactions: msg.Txns,
	}
	server.Blockchain = append(server.Blockchain, block)
	log.WithFields(log.Fields{
		"chain": utils.GetBlockchainPrint(server.Blockchain),
	}).Info("created block from new txns recvd")
}

// execPaxosRun initiates a PAXOS run and then adds the transaction to the local block chain
func (server *Server) execPaxosRun(txn *common.TransferTxn) {
	// 1. perform leader election
	log.Info("PAXOS RUN START")
	clientTxn = txn
	server.getElected()
}
