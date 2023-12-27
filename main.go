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

type respStatus struct {
	code int
	err  string
}

type testParams struct {
	runCounter      int
	client          fasthttp.Client
	url             string
	rateLimit       float64
	concurrentUsers int
	responseCodes   [6]int
	errorCount      map[string]int
	totalRequests   int
	statusChan      chan respStatus
	userCount       int
	master          bool
	worker          bool
	expectedWorkers int
	wg              sync.WaitGroup
	pbar            *progressbar.ProgressBar
}

const separator string = "===================="

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
		concurrentUsers: *c,
		responseCodes:   [6]int{},
		totalRequests:   *n,
		statusChan:      make(chan respStatus, 1000),
		errorCount:      make(map[string]int),
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
			progressbar.OptionShowElapsedTimeOnFinish(),
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
	log.SetPrefix("\n\n")
	go statusWorker(params)

	remainder := params.totalRequests % params.concurrentUsers
	requestsPerUser := (params.totalRequests - remainder) / params.concurrentUsers
	log.Printf("%[1]s\nTest Running\nConcurrency target: %d\nRequests target: %d\n%[1]s%[1]s\n\n", separator, params.concurrentUsers, params.totalRequests)

	var i int
	params.wg.Add(params.concurrentUsers)
	for i = 0; i < params.concurrentUsers; i++ {
		go iterate(params, requestsPerUser, i)
	}
	params.wg.Add(1)
	go iterate(params, remainder, i)

	params.wg.Wait()

	params.wg.Add(1)
	close(params.statusChan)

	params.wg.Wait()

	log.Printf("%[1]s%[1]s\nResponses received: %d\nResponse Codes Received:\n1xx: %d | 2xx: %d | 3xx: %d | 4xx: %d | 5xx: %d | Unknown: %d\n\n", separator,
		params.responseCodes[0]+params.responseCodes[1]+params.responseCodes[2]+params.responseCodes[3]+params.responseCodes[4]+params.responseCodes[5],
		params.responseCodes[0], params.responseCodes[1], params.responseCodes[2], params.responseCodes[3], params.responseCodes[4], params.responseCodes[5])

	log.SetPrefix("")
	log.Printf("%[1]s%[1]s%[1]s\nError count:", separator)
	log.SetFlags(0)
	for e, c := range params.errorCount {
		log.Printf(" - %d: \"%s\"", c, e)
	}
	log.Printf("%[1]s%[1]s%[1]s%[1]s\n\n", separator)
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

func statusWorker(params *testParams) {
	// responseCodes <[100s, 200s, 300s, 400s, 500s, unknowns]>
	defer params.wg.Done()

	for input := range params.statusChan {
		switch code, e := input.code, input.err; code != 0 {
		case code >= 100 && code < 200:
			params.responseCodes[0]++
		case code >= 200 && code < 300:
			params.responseCodes[1]++
		case code >= 300 && code < 400:
			params.responseCodes[2]++
		case code >= 400 && code < 500:
			params.responseCodes[3]++
		case code >= 500 && code < 600:
			params.responseCodes[4]++
		default:
			params.responseCodes[5]++
			if e != "" {
				params.errorCount[e]++
			}
		}
	}
}

func sendRequest(params *testParams, id string) (res respStatus) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.Set("id", id)
	req.SetRequestURI(params.url)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := params.client.Do(req, resp)
	if err != nil {
		res.code = -1
		res.err = err.Error()
	} else if resp == nil {
		res.code = -1
		res.err = "Error: empty response"
	} else {
		res.code = resp.StatusCode()
	}
	return

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
