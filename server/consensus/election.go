package consensus

import (
	"PAXOS-Banking/common"
	"net"
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

}

// processElectionMessage allows the server to decide if it should
// elect a new leader and join its ballot
func (server *Server) processElectionMessage(conn net.Conn, msg *common.PrepareMessage) {
	// 1. If the server has already promised its loyalty to another server, return a false

	// 2. Check the process id and seq number of the incoming request with the local process id+seq num.

	// 3. Decide if the leader should be elected based on [2]
}
