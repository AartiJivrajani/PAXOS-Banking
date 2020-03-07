package consensus

import (
	"PAXOS-Banking/common"
	"PAXOS-Banking/utils"
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
	peerLocalLogs            = make([]*common.AcceptedMessage, 0)
	recvdAcceptMsgMap        = make(map[int]bool)
)

type Server struct {
	Id int `json:"id"`
	// log maintained by the server
	Blockchain []*common.Block `json:"blockchain"`
	Port       int             `json:"port"`

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

	Log []*common.TransferTxn

	Ballot *common.Ballot

	PaxosState int
	// PaxosState represents the state this node is currently in during a paxos run.
	// Valid states are
	// 0 = follower
	// 1 = sent/receive prepare message
	// 2 = received/sent acks from majority, and elected as leader
	// 3 = sent/received accept
	// 4 = received/sent local logs from peers
}

// InitServer creates a new server instance and initializes all its parameters
// with default values. It also starts book-keeping the peers of this particular server
func InitServer(id int) *Server {
	server := &Server{
		Id:         id,
		Blockchain: make([]*common.Block, 0),
		Port:       common.ServerPortMap[id],
		// we always start off with seq num = 0
		SeqNum:           0,
		Status:           common.FOLLOWER,
		AlreadyPromised:  false,
		ServerConn:       make(map[int]net.Conn),
		Peers:            make([]int, 0),
		AssociatedClient: id,
		Log:              make([]*common.TransferTxn, 0),
		Ballot: &common.Ballot{
			BallotNum: 0,
			ProcessId: id,
		},
		// all nodes are followers initially
		PaxosState: 0,
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

func (server *Server) reconnectToServer(toServer int) {
	var (
		conn net.Conn
		err  error
		d    time.Duration
		b    = &backoff.Backoff{
			Min:    10 * time.Second,
			Max:    1 * time.Minute,
			Factor: 2,
			Jitter: false,
		}
	)
	log.WithFields(log.Fields{
		"fromServer": server.Id,
		"toServer":   toServer,
	}).Debug("re-establishing connection to server")
	PORT := ":" + strconv.Itoa(common.ServerPortMap[toServer])
	d = b.Duration()
	for {
		conn, err = net.Dial("tcp", PORT)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("error connecting to server")
			if b.Attempt() <= common.MaxReconnectAttempts {
				time.Sleep(d)
				continue
			} else {
				log.Panic("error connecting to server, shutting down...")
			}
		} else {
			log.WithFields(log.Fields{
				"fromServer": server.Id,
				"toServer":   toServer,
			}).Debug("connection established with server")
			break
		}
	}
	server.ServerConn[toServer] = conn
}

func (server *Server) getClientConnection() net.Conn {
	var (
		conn net.Conn
		err  error
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
		"clientId": server.Id,
	}).Debug("establishing connection to client")
	PORT := ":" + strconv.Itoa(common.ClientPortMap[server.Id])
	d = b.Duration()
	for {
		conn, err = net.Dial("tcp", PORT)
		if err != nil {
			log.WithFields(log.Fields{
				"serverId": server.Id,
				"clientId": server.Id,
				"error":    err.Error(),
			}).Debug("error connecting to client")
			if b.Attempt() <= common.MaxReconnectAttempts {
				time.Sleep(d)
				continue
			} else {
				log.Panic("error connecting to client, shutting down...")
			}
		} else {
			log.WithFields(log.Fields{
				"clientId": server.Id,
				"serverId": server.Id,
			}).Debug("connection established with client")
			break
		}
	}
	return conn
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
	possible := server.checkIfTxnPossible(transferRequest)
	if possible {
		server.execLocalTxn(transferRequest)
		resp := &common.Response{
			MessageType: common.SERVER_TXN_COMPLETE,
		}
		jResp, _ := json.Marshal(resp)
		_, _ = conn.Write(jResp)
	} else {
		server.execPaxosRun(transferRequest)
		// TODO: What do we do with the current client's request
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

// handleIncomingConnections simply decodes the incoming requests
// and forwards them to the right handlers
func (server *Server) handleIncomingConnections(conn net.Conn) {
	log.Info("handling incoming connections......")
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
			"request_type": request.Type,
			"fromId":       request.FromId,
		}).Debug("Request received")
		switch request.Type {
		case common.COMMIT_MESSAGE:
			server.updateBlockchain(request.BlockMessage)
		case common.ELECTION_ACCEPTED_MESSAGE:
			// since we are broadcasting the accept messages, we will receive the accepted messages
			// 4 times(twice from each peer). Maintain the IDs of the peers which have
			// already sent the accepted messages
			if _, OK := recvdAcceptMsgMap[request.FromId]; !OK {
				log.WithFields(log.Fields{
					"fromId":  request.FromId,
					"message": request.AcceptedMessage,
				}).Info("received ACCEPTED MESSAGE from...")
				peerLocalLogs = append(peerLocalLogs, request.AcceptedMessage)
				recvdAcceptMsgMap[request.FromId] = true
			}
			if server.validateBallotNumber(request.AcceptedMessage.Ballot.BallotNum) && server.PaxosState == 3 {
				server.PaxosState = 4
				log.WithFields(log.Fields{
					"new paxos state": server.PaxosState,
				}).Info("Changed to new paxos state")
				if !timerStarted {
					log.Info("starting timer(waiting for all ACCEPTED messages)")
					go server.waitForAcceptedMessages()
					// this is required, since the Accept messages from the other peer servers come
					// in quite fast, and during this interval, the timerStarted variable is not
					// set to True. Due to this race condition, the timer is started once again.
					// Sleep for 5 seconds to avoid such a condition.
					time.Sleep(5 * time.Second)
					go func() {
						for {
							if acceptedMsgTimeout {
								log.Info("time out!")
								server.processPeerLocalLogs(peerLocalLogs)
								peerLocalLogs = nil
								peerLocalLogs = make([]*common.AcceptedMessage, 0)
								recvdAcceptMsgMap = nil
								recvdAcceptMsgMap = make(map[int]bool)
								break
							}
						}
					}()
				}
			}
		case common.ELECTION_PREPARE_MESSAGE:
			server.PaxosState = 1
			log.WithFields(log.Fields{
				"new paxos state": server.PaxosState,
			}).Info("Changed to new paxos state")
			server.processPrepareMessage(request)
		case common.ELECTION_ACCEPT_MESSAGE:
			if server.PaxosState == 2 {
				server.PaxosState = 3
				log.WithFields(log.Fields{
					"new paxos state": server.PaxosState,
				}).Info("Changed to new paxos state")
				server.sendAllLocalLogs(request)
			}
		case common.ELECTION_PROMISE_MESSAGE:
			// Yay! You are a leader :D
			if server.validateBallotNumber(request.ElectionMsg.Ballot.BallotNum) && server.PaxosState == 1 {
				server.PaxosState = 2
				log.WithFields(log.Fields{
					"new paxos state": server.PaxosState,
				}).Info("Changed to new paxos state")
				server.sendAcceptMessage()
			}
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
		//----------------------- MESSAGES RECEIVED FROM CLIENT -----------------------
		case common.TRANSACTION_MESSAGE:
			server.processTxnRequest(server.getClientConnection(), request.TxnMessage)
		case common.SHOW_BALANCE:
			server.processBalanceRequest(server.getClientConnection())
		case common.SHOW_LOG_MESSAGE:
			logStr = utils.GetLocalLogPrint(server.Log)
			server.writeResponse(server.getClientConnection(), &common.Response{
				MessageType: common.SHOW_LOG_MESSAGE,
				Balance:     0,
				// Check if the clientId here needs to be 0 or not (was 0)
				ClientId:    server.Id,
				ToBePrinted: logStr,
			})
		case common.SHOW_BLOCKCHAIN_MESSAGE:
			logStr = utils.GetBlockchainPrint(server.Blockchain)
			server.writeResponse(server.getClientConnection(), &common.Response{
				MessageType: common.SHOW_BLOCKCHAIN_MESSAGE,
				// Check if this needs the client Id to be 0 or not
				ClientId:    server.Id,
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

// ClearRedisData is called every time a server is shut down, so that we don't have to manually
// remove the entries from redis before the next run
func ClearRedisData() {
	log.Info("Clearing local log from REDIS")
	BlockChainServer.RedisConn.Del(fmt.Sprintf(common.REDIS_LOG_KEY, BlockChainServer.Id))
	log.Info("Clearing blockchain from REDIS")
	BlockChainServer.RedisConn.Del(fmt.Sprintf(common.REDIS_BLOCKCHAIN_KEY, BlockChainServer.Id))
}

func Start(id int) {
	BlockChainServer = InitServer(id)
	go BlockChainServer.startListener()
	BlockChainServer.createTopology()
	//BlockChainServer.checkAndReconcile()
}
