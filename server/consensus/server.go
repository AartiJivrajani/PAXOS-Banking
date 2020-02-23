package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"container/list"
	"encoding/json"
	"net"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	redis "gopkg.in/redis.v5"

	"github.com/jpillora/backoff"
)

var (
	// TODO: See if we really need this?
	BlockchainLock           sync.Mutex
	BlockChainServer         *Server
	waitForReconcileResponse = make(chan []*common.ReconcileSeqMessage)
)

type Server struct {
	Id int `json:"id"`
	// log maintained by the server
	Blockchain *list.List `json:"blockchain"`
	Port       int        `json:"port"`

	// SeqNum + id are used to distinguish among values proposed by different leaders
	// This SeqNum is locally and monotonically incremented
	SeqNum int `json:"seq_num"`
	// whether the server is the leader or the follower currently
	Status string `json:"status"`
	// whether the server has already promised to follow another server.
	AlreadyPromised bool `json:"already_promised"`

	// mapping of the server ID v/s connection object(to maintain the network topology)
	ServerConn map[int]net.Conn

	// book-keep the other peers of this server
	Peers []int
	// each server serves a single client associated to it
	AssociatedClient int

	RedisConn *redis.Client

	Log *list.List
}

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
		Log:              list.New(),
	}
	server.RedisConn = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

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
			Min:    10 * time.Second,
			Max:    1 * time.Minute,
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

func (server *Server) processBalanceRequest(conn net.Conn) {
	balance := server.getLocalBalance()
	resp := &common.Response{
		MessageType: common.SHOW_BALANCE,
		Balance:     balance,
		ClientId:    server.Id,
	}
	jResp, err := json.Marshal(resp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"id":    server.Id,
		}).Error("error marshalling the balance response")
	}
	_, _ = conn.Write(jResp)
	return
}

func (server *Server) processElectionRequest(conn net.Conn, electionRequest *common.PrepareMessage) {

}

// handleIncomingConnections simply decodes the incoming requests
// and forwards them to the right handlers
func (server *Server) handleIncomingConnections(conn net.Conn) {
	var (
		request                 *common.Message
		err                     error
		logStr                  string
		numReconcileSeqMessages int
		seqNumbersRecvd         = make([]*common.ReconcileSeqMessage, 0)
	)
	d := json.NewDecoder(conn)
	for {
		err = d.Decode(&request)
		if err != nil {
			continue
		}
		log.WithFields(log.Fields{
			"request":      request,
			"request_type": request.Type,
		}).Debug("Request received from a client")
		switch request.Type {
		case common.RECONCILE_SEQ_NUMBERS:
			numReconcileSeqMessages += 1
			seqNumbersRecvd = append(seqNumbersRecvd, request.ReconcileSeqMessage)
			// only once the server receives 2 requests, should we process them.
			if numReconcileSeqMessages == 2 {
				waitForReconcileResponse <- seqNumbersRecvd
				// reset the reconciliation counter and seqNumbersReceived
				seqNumbersRecvd = nil
				seqNumbersRecvd = make([]*common.ReconcileSeqMessage, 0)
				numReconcileSeqMessages = 0
			}
		case common.RECONCILE_REQ_MESSAGE:
			server.handleReconcileRequestMessage(conn)
		case common.TRANSACTION_MESSAGE:
			server.processTxnRequest(conn, request.TxnMessage)
		case common.PREPARE_MESSAGE:
			server.processElectionRequest(conn, request.PrepareMsg)
		case common.SHOW_BALANCE:
			server.processBalanceRequest(conn)
		case common.SHOW_LOG_MESSAGE:
			logStr = utils.GetLocalLogPrint(server.Log)
			server.writeResponse(conn, &common.Response{
				MessageType: common.SHOW_LOG_MESSAGE,
				Balance:     0,
				ClientId:    0,
				ToBePrinted: logStr,
			})
		case common.SHOW_BLOCKCHAIN_MESSAGE:
			logStr = utils.GetBlockchainPrint(server.Blockchain)
			server.writeResponse(conn, &common.Response{
				MessageType: common.SHOW_BLOCKCHAIN_MESSAGE,
				Balance:     0,
				ClientId:    0,
				ToBePrinted: logStr,
			})
		}
	}
}

// writeResponse sends a response back to the client
func (server *Server) writeResponse(conn net.Conn, resp *common.Response) {
	jResp, err := json.Marshal(resp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"resp":  resp,
		}).Error("error marshalling response message")
		return
	}
	_, err = conn.Write(jResp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error sending response back to the client")
	}
	return
}

// startClientListener opens a connection to the servers' respective clients and listens to
// it. The client sends all its transaction requests to the server.
func (server *Server) startListener() {
	var (
		err error
	)
	PORT := ":" + strconv.Itoa(common.ServerPortMap[server.Id])
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Error("error establishing connection to the server port, shutting down... ")
		return
	}
	for {
		c, err := listener.Accept()
		if err != nil {
			log.Error("error starting the server listener, shutting down...")
			return
		}
		go server.handleIncomingConnections(c)
	}
}

func Start(id int) {
	BlockChainServer = InitServer(id)
	go BlockChainServer.startListener()

	BlockChainServer.createTopology()

	BlockChainServer.checkAndReconcile()
}
