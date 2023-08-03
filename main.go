package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
)

type testParams struct {
	runCounter int
	client http.Client
	url string
	paths []string
	rateLimit float64
	concurency int
	respList [6]int
	requestsTarget int
	userCount int
}

func main() {
	params := &testParams{
		runCounter: 4,
		client:     http.Client{},
		url:        "https://google.com",
		paths:      []string{},
		rateLimit:  0,
		concurency: 100,
		respList:   [6]int{},
		requestsTarget: 10000,
		userCount: 0,
	}
	runTest(params)
}

func runTest(params *testParams) {
	//respList <[100s, 200s, 300s, 400s, 500s, unknowns]>

	var wg sync.WaitGroup
	userRequestTarget := int(math.Ceil(float64(params.requestsTarget)/float64(params.concurency)))
	log.Println("test running***")
	for i := 0; i < params.concurency; i++ {
		wg.Add(1)
		go func(params *testParams, target int, userID int) {
			defer wg.Done()
			log.Println("New user created***")
			for i := 0; i < target; i++ {
				req, err := http.NewRequest("GET", params.url, nil)
				req.Header.Set("ID", fmt.Sprintf("RID%03d.UID%05d.CID%06d", params.runCounter, userID, i))
				if err != nil {
					log.Fatal(err)
				}
				resp, err := params.client.Do(req)
				if err != nil {
					if resp == nil {
						log.Fatalf("\n==================\nNo host was found at %s\nDouble check the URL and try again.\n==================", params.url)
					}
					log.Fatal(err)
				}
				switch code := resp.StatusCode; {
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
		}(params, userRequestTarget, i)
	}

	wg.Wait()
	log.Printf("\n====================================================================\nResponse Codes Received:\n1xx: %d | 2xx: %d | 3xx: %d | 4xx: %d | 5xx: %d | Unknown: %d\n====================================================================", params.respList[0], params.respList[1], params.respList[2], params.respList[3], params.respList[4], params.respList[5])
}
