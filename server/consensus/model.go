package consensus

import (
	"container/list"
	"net"
	"sync"
)

var (
	LogLock sync.Mutex
)

type Server struct {
	Id   int        `json:"id"`
	Log  *list.List `json:"log"`
	Port int        `json:"port"`

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
}
