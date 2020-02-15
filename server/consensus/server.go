package consensus

import (
	"PAXOS-Banking/common"
	"container/list"
	"encoding/json"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/jpillora/backoff"
)

var BlockChainServer *Server

// InitServer creates a new server instance and initializes all its parameters
// with default values. It also starts book-keeping the peers of this particular server
func InitServer(id int) *Server {
	server := &Server{
		Id:         id,
		Blockchain: list.New(),
		Port:       common.ServerPortMap[id],
		// we always start off with seq num = 1
		SeqNum:           1,
		Status:           common.FOLLOWER,
		AlreadyPromised:  false,
		ServerConn:       make(map[int]net.Conn),
		Peers:            make([]int, 0),
		AssociatedClient: id,
	}
	if id == 1 {
		server.Peers = append(server.Peers, 2, 3)
	} else if id == 2 {
		server.Peers = append(server.Peers, 1, 3)
	} else if id == 3 {
		server.Peers = append(server.Peers, 1, 2)
	}
	return server
}

// createTopology establishes a connection between the server and all its peers
// in case of a connection error, 3 re-connection attempts are made with exponential back-off
// retry. Otherwise, an error is thrown and the process exits. The connections established with
// the peers are stored on the server in the `ServerConn` map
func (server *Server) createTopology() {
	var (
		err  error
		conn net.Conn
		d    time.Duration
		b    = &backoff.Backoff{
			//These are the defaults
			Min:    100 * time.Millisecond,
			Max:    10 * time.Second,
			Factor: 2,
			Jitter: false,
		}
	)
	log.WithFields(log.Fields{
		"serverId": server.Id,
		"peers":    server.Peers,
	}).Debug("establishing peer connections")

	for _, peer := range server.Peers {
		PORT := ":" + strconv.Itoa(common.ServerPortMap[peer])
		d = b.Duration()
		for {
			conn, err = net.Dial("tcp", PORT)
			if err != nil {
				log.WithFields(log.Fields{
					"serverId":   server.Id,
					"peerServer": peer,
					"error":      err.Error(),
				}).Debug("error connecting to server")
				if b.Attempt() <= common.MaxReconnectAttempts {
					time.Sleep(d)
					continue
				} else {
					log.Panic("error connecting to peer server, shutting down...")
				}
			} else {
				log.WithFields(log.Fields{
					"serverId":   server.Id,
					"peerServer": peer,
				}).Debug("connection established with server")
				break
			}
		}
		server.ServerConn[peer] = conn
	}
}

// processTxnRequest first checks the client balance in the local blockchain. If the amount to be
// transferred is greater than the local balance, a PAXOS run is made, in order to
// fetch the transactions from all the other replicas. Else, a block is added to the blockchain and
// the transaction is carried out locally.
func (server *Server) processTxnRequest(conn net.Conn, transferRequest *common.TransferTxn) {
	// check the current balance
	runPaxos := server.checkIfTxnPossible(transferRequest)
	if !runPaxos {
		server.execLocalTxn(transferRequest)
	} else {
		server.execPaxosRun(transferRequest)
	}
}

// handleClientConnections simply decodes the client transaction requests
// and forwards them to the right handlers
func (server *Server) handleClientConnections(conn net.Conn) {
	var (
		transferRequest *common.TransferTxn
		err             error
	)
	d := json.NewDecoder(conn)
	for {
		err = d.Decode(&transferRequest)
		if err != nil {
			continue
		}
		log.WithFields(log.Fields{
			"request": transferRequest,
		}).Debug("Request recvd from a client")
		go server.processTxnRequest(conn, transferRequest)
	}
}

// startClientListener opens a connection to the servers' respective clients and listens to
// it. The client sends all its transaction requests to the server.
func (server *Server) startClientListener() {
	var (
		err error
	)
	PORT := ":" + strconv.Itoa(common.ClientPortMap[server.Id])
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Error("error establishing connection to the client, shutting down... ")
		return
	}
	for {
		c, err := listener.Accept()
		if err != nil {
			log.Error("error starting the client listener, shutting down...")
			return
		}
		go server.handleClientConnections(c)
	}
}

func Start(id int) {
	BlockChainServer = InitServer(id)
	BlockChainServer.createTopology()

	go BlockChainServer.startClientListener()
}
