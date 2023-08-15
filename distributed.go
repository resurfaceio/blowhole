package main

import (
	"log"
	"net"
	"net/rpc"
	"sync"
)

type master struct {
	targetRequests    int
	targetConcurrency int
	expectedWorkers   int
}
type worker struct {
	ID                int
	targetRequests    int
	targetConcurrency int
	ready             bool
}

type RPCHandler struct {
	workerCounter int
}

func startDistributedTest(params *testParams) {
	mstr := master{
		targetRequests:    params.requests,
		targetConcurrency: params.concurrency,
		expectedWorkers:   params.expectedWorkers,
	}

	wg := sync.WaitGroup{}
	l, err := net.Listen("tcp", "0.0.0.0:9111")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			rpc.Accept(l)
		}
	}()
	log.Println("***Listening on port 9111 for RPC requests***")

	wg.Wait()
}

func startDistributedWorker() {
	client, err := rpc.Dial("tcp", "localhost:9111")
	if err != nil {
		log.Fatal(err)
	}

	err = client.Call("RPCListner.RegisterWorker", 1, worker{
		ID:                nil,
		targetRequests:    nil,
		targetConcurrency: nil,
		ready:             false,
	})
}

func New() *RPCHandler {
	h := &RPCHandler{
		workerCounter: 1
	}
	err := rpc.Register(h)
	if err != nil {
		log.Fatal(err)
	}
	return h
}

func (rpch *RPCHandler) RegisterWorker(payload int, reply *worker) error {
	reply.ID =
	return nil
}
