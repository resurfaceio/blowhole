package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"

	distributed "github.com/resurfaceio/blowhole/DistributedServices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type myIdentifyServer struct {
	distributed.UnimplementedIdentifyServer
	workerCount   int64
	reqPerWorker  int64
	concPerWorker int64
}

type myStatsServer struct {
	distributed.UnimplementedStatsServer
	statsChan *chan []int64
}

func (s myIdentifyServer) Create(ctx context.Context, request *distributed.IDRequest) (*distributed.IDResponse, error) {

	return &distributed.IDResponse{
		WorkerID:    s.workerCount,
		Requests:    s.reqPerWorker,
		Concurrency: s.concPerWorker,
	}, nil
}

func (s myStatsServer) Create(ctx context.Context, request *distributed.StatsRequest) (*distributed.StatsResponse, error) {
	*s.statsChan <- request.Responses

	return &distributed.StatsResponse{
		Status: 0,
	}, nil
}

func startDistributedTest(params *testParams) {
	expectedWorkers := params.expectedWorkers
	var reqPerWorker int64
	var concPerWorker int64
	if expectedWorkers > 1 {
		reqPerWorker = int64(math.Ceil(float64(params.requests) / float64(expectedWorkers)))
		concPerWorker = int64(math.Ceil(float64(params.concurrency) / float64(expectedWorkers)))
	} else {
		log.Fatalf("\n*****************\nCannot run distributed test with less than 2 'expected workers'\n*****************")
	}

	statsChan := make(chan []int64, 1000)

	lis, err := net.Listen("tcp", "localhost:9111")
	if err != nil {
		log.Fatalf("Could not create listener: %s", err)
	}

	serverRegistrar := grpc.NewServer()
	IDService := &myIdentifyServer{
		UnimplementedIdentifyServer: distributed.UnimplementedIdentifyServer{},
		workerCount:                 0,
		reqPerWorker:                reqPerWorker,
		concPerWorker:               concPerWorker,
	}
	StatsService := &myStatsServer{
		UnimplementedStatsServer: distributed.UnimplementedStatsServer{},
		statsChan:                &statsChan,
	}

	distributed.RegisterIdentifyServer(serverRegistrar, IDService)
	distributed.RegisterStatsServer(serverRegistrar, StatsService)

	err = serverRegistrar.Serve(lis)
	if err != nil {
		log.Fatalf("Could not launch server: %s", err)
	}
}

func startDistributedWorker(params *testParams) {
	type worker struct {
		concurrency int
		requests    int
		id          int
	}

	wrkr := worker{
		concurrency: 0,
		requests:    0,
		id:          0,
	}

	con, err := grpc.Dial("localhost:9111", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Unable to connect to GRPC server: %s", err)
	}
	time.Sleep(time.Second * 1)
	log.Printf("=============Worker connection status: %s=============\n", con.GetState())

	clientID := distributed.NewIdentifyClient(con)

	log.Printf("=============Talking to the master node=============\n")
	respID, err := clientID.Create(context.Background(), &distributed.IDRequest{})
	if err != nil {
		log.Fatalf("Worker ID request failed: %s", err)
	}
	wrkr.concurrency = int(respID.Concurrency)
	wrkr.requests = int(respID.Requests)
	wrkr.id = int(respID.WorkerID)

	wg := sync.WaitGroup{}
	clientStats := distributed.NewStatsClient(con)

	log.Printf("=============Work received, ready to start=============\n")

	//wg.Add(1)
	//wg.Wait()

	for i := 0; i < wrkr.concurrency; i++ {
		wg.Add(1)
		go func(params *testParams, target int, userID int) {
			var respCodes []int64
			defer wg.Done()
			for i := 0; i < target; i++ {
				reqID := fmt.Sprintf("RID%03d.UID%05d.CID%06d", params.runCounter, userID, i)
				respCode := sendRequest(params, reqID)
				if len(respCodes) < 50 {
					respCodes = append(respCodes, int64(respCode))
				} else {
					workerSendStats(respCodes, clientStats)
				}
			}
		}(params, wrkr.requests, i)
	}
	wg.Wait()
}

func workerSendStats(stats []int64, client distributed.StatsClient) {
	respStats, err := client.Create(context.Background(), &distributed.StatsRequest{Responses: stats})
	if err != nil {
		log.Fatalf("Stats failed to send: %s", err)
	}
	if respStats.Status != 1 {
		log.Fatalf("Stats failed to receive: status code %d", respStats.Status)
	}
}

//func workerWaitReady(wg *sync.WaitGroup, client) {
//
//}
