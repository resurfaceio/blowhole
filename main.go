package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	log.Println("Starting test***")
	runner(100, "http://localhost:8080/http-bin/")
}

func runner(users int, url string) {
	//respList <[100s, 200s, 300s, 400s, 500s, unknowns]>
	var respList [6]int
	var wg sync.WaitGroup
	log.Println("Runner started***")
	run := 3
	for i := 0; i < users; i++ {
		wg.Add(1)
		go user(url, i, run, respList, &wg)
	}
	wg.Wait()
	log.Println(respList)
}

func user(url string, userID int, run int, respList [6]int, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("New user created***")
	var client http.Client
	for i := 0; i < 1000; i++ {
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("ID", fmt.Sprintf("RID%03d.UID%05d.CID%06d", run, userID, i))
		if err != nil {
			log.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		switch code := resp.StatusCode; {
		case code < 200:
			respList[0]++
		case code >= 200 && code < 300:
			respList[1]++
		case code >= 300 && code < 400:
			respList[2]++
		case code >= 400 && code < 500:
			respList[3]++
		case code >= 500 && code < 600:
			respList[4]++
		default:
			respList[5]++
		}
	}
	return
}
