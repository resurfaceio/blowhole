package main

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"
	"log"
	"sync"
	"time"
)

type testParams struct {
	runCounter      int
	client          fasthttp.Client
	url             string
	rateLimit       float64
	concurrency     int
	respList        [6]int
	requests        int
	statusChan      chan int
	userCount       int
	master          bool
	worker          bool
	expectedWorkers int
}

func main() {
	params := &testParams{
		runCounter: 12,
		client: fasthttp.Client{
			MaxConnsPerHost:               1500,
			ReadTimeout:                   2 * time.Second,
			WriteTimeout:                  2 * time.Second,
			DisableHeaderNamesNormalizing: true,
		},
		url:             "http://localhost:8080/http-bin/png",
		rateLimit:       0,
		concurrency:     100,
		respList:        [6]int{},
		requests:        1000,
		statusChan:      make(chan int, 1000),
		userCount:       0,
		master:          false,
		worker:          true,
		expectedWorkers: 2,
	}

	if params.master {
		startDistributedTest(params)
	} else if params.worker {
		startDistributedWorker(params)
	} else {
		startLumpedTest(params)
	}
}

func startLumpedTest(params *testParams) {
	//respList <[100s, 200s, 300s, 400s, 500s, unknowns]>

	go statWorker(params)
	var wg sync.WaitGroup
	remainder := params.requests % params.concurrency
	userRequestTarget := (params.requests - remainder) / params.concurrency
	log.Printf("\n=================================\nTest Running\nConcurrency target: %d\nResquests target: %d\n=================================", params.concurrency, params.requests)
	pbar := progressbar.NewOptions(params.requests,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowIts(),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetItsString("requests"),
	)
	for i := 0; i < params.concurrency; i++ {
		wg.Add(1)
		go func(params *testParams, target int, userID int) {
			defer wg.Done()
			for i := 0; i < target; i++ {
				reqID := fmt.Sprintf("RID%03d.UID%05d.CID%06d", params.runCounter, userID, i)
				resp := sendRequest(params, reqID)
				params.statusChan <- resp.StatusCode()
				err := pbar.Add(1)
				if err != nil {
					log.Println(err)
				}
			}
		}(params, userRequestTarget, i)
	}
	wg.Add(1)
	go func(params *testParams, target int, userID int) {
		defer wg.Done()
		for i := 0; i < target; i++ {
			reqID := fmt.Sprintf("RID%03d.UID%05d.CID%06d", params.runCounter, userID, i)
			resp := sendRequest(params, reqID)
			params.statusChan <- resp.StatusCode()
			err := pbar.Add(1)
			if err != nil {
				log.Println(err)
			}
		}
	}(params, remainder, -1)

	wg.Wait()
	log.Printf("\n====================================================================\nResponse Codes Received:\n1xx: %d | 2xx: %d | 3xx: %d | 4xx: %d | 5xx: %d | Unknown: %d\n====================================================================", params.respList[0], params.respList[1], params.respList[2], params.respList[3], params.respList[4], params.respList[5])
}

func statWorker(params *testParams) {
work:
	for {
		input, open := <-params.statusChan
		if input != 0 {
			switch code := input; {
			case code < 200:
				params.respList[0]++
			case code >= 200 && code < 300:
				params.respList[1]++
			case code >= 300 && code < 400:
				params.respList[2]++
			case code >= 400 && code < 500:
				params.respList[3]++
			case code >= 500 && code < 600:
				params.respList[4]++
			default:
				params.respList[5]++
			}
		}
		if !open {
			break work
		}
	}
}

func sendRequest(params *testParams, id string) *fasthttp.Response {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.Set("id", id)
	req.SetRequestURI(params.url)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := fasthttp.Do(req, resp)
	if err != nil {
		log.Println(err)
	}
	if resp != nil {
		return resp
	} else {
		log.Println("No response returned***")
		return nil
	}
}

//type countingConn struct {
//	net.Conn
//	bytesRead, bytesWritten *int64
//}

//var fasthttpDialFunc = func(
//	bytesRead, bytesWritten *int64,
//) func(string) (net.Conn, error) {
//	return func(address string) (net.Conn, error) {
//		conn, err := net.Dial("tcp", address)
//		if err != nil {
//			return nil, err
//		}
//
//		wrappedConn := &countingConn{
//			Conn:         conn,
//			bytesRead:    bytesRead,
//			bytesWritten: bytesWritten,
//		}
//
//		return wrappedConn, nil
//	}
//}
