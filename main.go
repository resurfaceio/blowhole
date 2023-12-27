package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	name            string
	runID           string
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

const separator string = "============================================================"

var initMessage string = "\nTest: %21s\nRun ID: %18s\nRequests target: %7d\nConcurrency level: %2d"
var resultMessage string = "\nRequests sent: %d\nResponse codes received: \n  1xx: %d | 2xx: %d | 3xx: %d | 4xx: %d | 5xx: %d | Unknown: %d"

func main() {
	runc := flag.Int("run", 1, "int. Run counter to use as RID in id header")
	c := flag.Int("c", 1, "int. Number of concurrent connections")
	n := flag.Int("n", 1, "int. Number of requests to perform")
	targetUrl := flag.String("url", "http://localhost:8000/", "string. Target URL to perform requests for")
	readTimeout := flag.Int("rtimeout", 500, "int. Maximum duration for full response reading (including body) in milliseconds")
	writeTimeout := flag.Int("wtimeout", 500, "int. Maximum duration for full request writing (including body) in milliseconds")
	maxConnections := flag.Int("maxconn", 1000, "int. Maximum number of connections per each host which may be established.")
	isDistributed := flag.Bool("distributed", false, "bool. Blowhole will perform requests using distributed clients if set.")
	isWorker := flag.Bool("worker", false, "bool. Blowhole instance will act as distributed worker if set. It has no effect unless \"distributed\" is also set.")
	output := flag.String("o", "", "string. Output destination for results. If not set, defaults to stdout.")
	batchFile := flag.String("file", "", "string. Path of YAML file describing a batch of runs")
	flag.Parse()

	var batch batchSpec
	if *batchFile != "" {
		batch.getConf(*batchFile)
	} else {
		batch = batchSpec{
			Name: "unnamed",
			Url:  *targetUrl,
			Runs: []runConf{{
				Requests:    *n,
				Concurrency: *c,
			}},
			IsDistributed: *isDistributed,
			IsWorker:      *isWorker,
			Output:        *output,
		}

	}

	if batch.Output != "" {
		file, err := os.OpenFile(batch.Output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		log.SetOutput(file)

		initMessage = "test %s,%s,%d,%d"
		resultMessage = "%d, %[7]d"
	}

	for i, run := range batch.Runs {
		params := &testParams{
			name: batch.Name,
			client: fasthttp.Client{
				MaxConnsPerHost:               *maxConnections,
				ReadTimeout:                   time.Duration(*readTimeout) * time.Millisecond,
				WriteTimeout:                  time.Duration(*writeTimeout) * time.Millisecond,
				DisableHeaderNamesNormalizing: true,
			},
			url:             batch.Url,
			rateLimit:       0,
			concurrentUsers: run.Concurrency,
			responseCodes:   [6]int{},
			totalRequests:   run.Requests,
			statusChan:      make(chan respStatus, 1000),
			errorCount:      make(map[string]int),
			userCount:       0,
			master:          batch.IsDistributed && !batch.IsWorker,
			worker:          batch.IsDistributed && batch.IsWorker,
			expectedWorkers: 2,
			pbar: progressbar.NewOptions(run.Requests,
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionShowIts(),
				progressbar.OptionSetWidth(len(separator)),
				progressbar.OptionShowCount(),
				progressbar.OptionSetItsString("requests"),
				progressbar.OptionShowElapsedTimeOnFinish(),
			),
		}

		if *runc == 1 {
			params.runID = fmt.Sprintf("RID%03d", i+1)
		} else {
			params.runID = fmt.Sprintf("RID%03d", *runc)
		}

		if run.CustomID != "" {
			params.runID = run.CustomID
		}

		if run.CustomURL != "" {
			params.url = run.CustomURL
		}

		if params.master {
			startDistributedTest(params)
		} else if params.worker {
			startDistributedWorker(params)
		} else {
			startLumpedTest(params)
		}

	}

}

func startLumpedTest(params *testParams) {
	go statusWorker(params)

	remainder := params.totalRequests % params.concurrentUsers
	requestsPerUser := (params.totalRequests - remainder) / params.concurrentUsers

	fmt.Printf("%s\nTest \"%s\" running - Run: %s\n\n", separator, params.name, params.runID)
	log.Printf(initMessage, params.name, params.runID, params.totalRequests, params.concurrentUsers)
	fmt.Println()

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

	fmt.Print("\n\n")
	log.Printf(resultMessage, params.responseCodes[0]+params.responseCodes[1]+params.responseCodes[2]+params.responseCodes[3]+params.responseCodes[4]+params.responseCodes[5],
		params.responseCodes[0], params.responseCodes[1], params.responseCodes[2], params.responseCodes[3], params.responseCodes[4], params.responseCodes[5])

	if len(params.errorCount) != 0 {
		fmt.Println("Error count:")
	}
	for e, c := range params.errorCount {
		fmt.Printf("  + %d: \"%s\"\n", c, e)
	}
	fmt.Println(separator)
}

func iterate(params *testParams, target int, userID int) {
	defer params.wg.Done()

	for i := 0; i < target; i++ {
		reqID := fmt.Sprintf("%s.UID%05d.CID%06d", params.runID, userID, i)
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
