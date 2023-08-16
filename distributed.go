package main

import (
	"context"
	distributed "github.com/resurfaceio/blowhole/DistributedServices"
	"google.golang.org/grpc"
	"log"
	"math"
	"net"
)

type myIdentifyServer struct {
	distributed.UnimplementedIdentifyServer
	workerCount   int64
	reqPerWorker  int64
	concPerWorker int64
}

type myStatsServer struct {
	distributed.UnimplementedStatsServer
	statsChan *chan string
}

func (s myIdentifyServer) Create(ctx context.Context, request *distributed.IDRequest) (*distributed.IDResponse, error) {

	return &distributed.IDResponse{
		WorkerID:    s.workerCount,
		Requests:    s.reqPerWorker,
		Concurrency: s.concPerWorker,
	}, nil
}

func (s myStatsServer) Create(ctx context.Context, request *distributed.StatsRequest) (*distributed.StatsResponse, error) {
	*s.statsChan <- request.String()

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

	statsChan := make(chan string, 1000)

	lis, err := net.Listen("tcp", ":9111")
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

func startDistributedWorker() {

}
