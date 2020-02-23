package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
	"container/list"
	"encoding/json"
	"fmt"
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

// handleReconcileRequestMessage returns a list of sequence numbers of the server's current block chain
func (server *Server) handleReconcileRequestMessage(conn net.Conn) {
	// send a list of all sequence numbers
	seqNumbers := make([]int, 0)
	for block := server.Blockchain.Front(); block != nil; block = block.Next() {
		seqNumbers = append(seqNumbers, block.Value.(*common.Block).SeqNum)
	}
	resp := &common.Message{
		Type: common.RECONCILE_SEQ_NUMBERS,
		ReconcileSeqMessage: &common.ReconcileSeqMessage{
			Id:                  server.Id,
			ReconcileSeqNumbers: seqNumbers,
		},
	}
	jResp, _ := json.Marshal(resp)
	_, _ = conn.Write(jResp)
}

// handleIncomingConnections simply decodes the incoming requests
// and forwards them to the right handlers
func (server *Server) handleIncomingConnections(conn net.Conn) {
	var (
		request                 *common.Message
		err                     error
		logStr                  string
		numReconcileSeqMessages int
		seqNumbersRecvd         []*common.ReconcileSeqMessage
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
				// reset the reconciliation counter
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

func (server *Server) sendReconcileRequest() {
	var (
		request *common.Message
	)
	for _, peer := range server.Peers {
		if server.ServerConn[peer] != nil {
			request = &common.Message{
				Type: common.RECONCILE_REQ_MESSAGE,
			}
			jReq, _ := json.Marshal(request)
			_, _ = server.ServerConn[peer].Write(jReq)
		} else {
			//TODO: Take care of the case when the connections are down and new connections need to be
			// established before continuing
		}
	}
}

func (server *Server) reconcile(val string) {
	var (
		blockChain    *list.List
		blockArrChain *common.BlockArrChain
		err           error
	)
	err = json.Unmarshal([]byte(val), &blockArrChain)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error unmarshalling blockchain data received from redis")
		return
	}
	blockChain = utils.GetBlockChainFromArr(blockArrChain)
	server.Blockchain = blockChain
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
	// redis lookup
	val, err := server.RedisConn.Get(fmt.Sprintf(common.REDIS_BLOCKCHAIN_KEY, server.Id)).Result()
	if err != redis.Nil {
		return
	} else {
		server.reconcile(val)
		// TODO: update the local log also
		return
	}
}

func Start(id int) {
	BlockChainServer = InitServer(id)
	go BlockChainServer.startListener()

	BlockChainServer.createTopology()

	BlockChainServer.checkAndReconcile()
}
