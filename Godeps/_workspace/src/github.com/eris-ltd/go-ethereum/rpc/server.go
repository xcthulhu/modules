package rpc

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/xeth"
)

var jsonlogger = logger.NewLogger("JSON")

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
	pipe     *xeth.JSXEth
}

func (s *JsonRpcServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.listener.Close()
			break out
		}
	}

	jsonlogger.Infoln("Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	jsonlogger.Infoln("Starting JSON-RPC server")
	go s.exitHandler()
	rpc.Register(&EthereumApi{pipe: s.pipe})
	rpc.HandleHTTP()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			jsonlogger.Infoln("Error starting JSON-RPC:", err)
			break
		}
		jsonlogger.Debugln("Incoming request.")
		go jsonrpc.ServeConn(conn)
	}
}

func NewJsonRpcServer(pipe *xeth.JSXEth, port int) (*JsonRpcServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
	}, nil
}
