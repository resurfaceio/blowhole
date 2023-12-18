package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"
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
	wg              sync.WaitGroup
	pbar            *progressbar.ProgressBar
}

const separator string = "=================================="

func main() {
	run := flag.Int("run", 1, "int. Run counter to use as RID in id header")
	c := flag.Int("c", 1, "int. Number of concurrent connections")
	n := flag.Int("n", 1, "int. Number of requests to perform")
	targetUrl := flag.String("url", "http://localhost:8000/", "string. Target URL to perform requests for")
	readTimeout := flag.Int("rtimeout", 500, "int. Maximum duration for full response reading (including body) in milliseconds")
	writeTimeout := flag.Int("wtimeout", 500, "int. Maximum duration for full request writing (including body) in milliseconds")
	maxConnections := flag.Int("maxconn", 1000, "int. Maximum number of connections per each host which may be established.")
	flag.Parse()

	params := &testParams{
		runCounter: *run,
		client: fasthttp.Client{
			MaxConnsPerHost:               *maxConnections,
			ReadTimeout:                   time.Duration(*readTimeout) * time.Millisecond,
			WriteTimeout:                  time.Duration(*writeTimeout) * time.Millisecond,
			DisableHeaderNamesNormalizing: true,
		},
		url:             *targetUrl,
		rateLimit:       0,
		concurrency:     *c,
		respList:        [6]int{},
		requests:        *n,
		statusChan:      make(chan int, 1000),
		userCount:       0,
		master:          false,
		worker:          false,
		expectedWorkers: 2,
		pbar: progressbar.NewOptions(*n,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowIts(),
			progressbar.OptionFullWidth(),
			progressbar.OptionShowCount(),
			progressbar.OptionSetItsString("requests"),
		),
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

	remainder := params.requests % params.concurrency
	userRequestTarget := (params.requests - remainder) / params.concurrency
	log.Printf("\n%s\nTest Running\nConcurrency target: %d\nRequests target: %d\n%s", separator, params.concurrency, params.requests, separator)
	for i := 0; i < params.concurrency; i++ {
		params.wg.Add(1)
		go iterate(params, userRequestTarget, i)
	}
	params.wg.Add(1)
	go iterate(params, remainder, -1)

	params.wg.Wait()
	close(params.statusChan)
	log.Printf("\n%s%s\nResponses received: %d\nResponse Codes Received:\n1xx: %d | 2xx: %d | 3xx: %d | 4xx: %d | 5xx: %d | Unknown: %d\n%s%s", separator, separator,
		params.respList[0]+params.respList[1]+params.respList[2]+params.respList[3]+params.respList[4]+params.respList[5],
		params.respList[0], params.respList[1], params.respList[2], params.respList[3], params.respList[4], params.respList[5], separator, separator)
}

func iterate(params *testParams, target int, userID int) {
	defer params.wg.Done()
	for i := 0; i < target; i++ {
		reqID := fmt.Sprintf("RID%03d.UID%05d.CID%06d", params.runCounter, userID, i)
		params.statusChan <- sendRequest(params, reqID)
		err := params.pbar.Add(1)
		if err != nil {
			log.Println(err)
		}
	}
}

func statWorker(params *testParams) {
	for input := range params.statusChan {
		switch code := input; input != 0 {
		case code >= 100 && code < 200:
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
}

func sendRequest(params *testParams, id string) int {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.Set("id", id)
	req.SetRequestURI(params.url)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := params.client.Do(req, resp)
	if err != nil || resp == nil {
		return -1
	}
	return resp.StatusCode()
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
