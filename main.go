package main

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"
	"log"
	"math"
	"sync"
)

type testParams struct {
	runCounter     int
	client         fasthttp.Client
	url            string
	paths          []string
	rateLimit      float64
	concurrency    int
	respList       [6]int
	requestsTarget int
	statusChan     chan int
	userCount      int
}

func main() {
	params := &testParams{
		runCounter: 12,
		client: fasthttp.Client{
			MaxConnsPerHost: 1024,
		},
		url:            "http://localhost:8080/http-bin/",
		paths:          []string{},
		rateLimit:      0,
		concurrency:    100,
		respList:       [6]int{},
		requestsTarget: 1000000,
		statusChan:     make(chan int, 1000),
		userCount:      0,
	}
	runTest(params)
}

func runTest(params *testParams) {
	//respList <[100s, 200s, 300s, 400s, 500s, unknowns]>

	go statWorker(params)
	var wg sync.WaitGroup
	userRequestTarget := int(math.Ceil(float64(params.requestsTarget) / float64(params.concurrency)))
	log.Printf("\n=================================\nTest Running\nConcurrency target: %d\nResquests target: %d\n=================================", params.concurrency, params.requestsTarget)
	pbar := progressbar.NewOptions(params.requestsTarget,
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
				sendRequest(params, reqID)
				err := pbar.Add(1)
				if err != nil {
					log.Println(err)
				}
			}
		}(params, userRequestTarget, i)
	}

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

func sendRequest(params *testParams, id string) {
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
		params.statusChan <- resp.StatusCode()
	} else {
		log.Println("No response returned***")
	}
}
