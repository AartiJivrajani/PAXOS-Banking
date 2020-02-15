package consensus

import (
	"PAXOS-Banking/common"
	"container/list"
	"net"
)

var BlockChainServer *Server

func InitServer(id int) *Server {
	return &Server{
		Id:   0,
		Log:  list.New(),
		Port: common.ServerPortMap[id],
		// we always start off with seq num = 1
		SeqNum:          1,
		Status:          common.FOLLOWER,
		AlreadyPromised: false,
		ServerConn:      make(map[int]net.Conn),
	}
}
func Start(id int) {
	BlockChainServer = InitServer(id)

}
