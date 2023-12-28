# resurfaceio-blowhole

Load test servers with unique requests.

```
        .
       ":"
     ___:____     |"\/"|
   ,'        `.    \  /
   |  O        \___/  |
 ~^~^~^~^~^~^~^~^~^~^~^~^~ 

(ASCII art by Riitta Rasimus)
```

Blowhole is a command line tool written in Go to perform unique requests for a given URL.
Uniqueness is guaranteed by adding an "id" header to each request with the following format:

```
"id: RIDxxx.UIDxxxxx.CIDxxxxxx"
```

Where:

 - `RID`: run ID - same for each run. Increases for each run in a batch.
 - `UID`: user ID - same for each "user" (each goroutine sending a portion of the total requests).
 - `CID`: count ID - different with each request performed for a given user.

# Dependencies

- Go 1.20+


# Build

```bash
go build .
```

# Usage

## Basic usage

```bash
# 100 requests for a /json resource served at localhost:8000

./blowhole -n 100 -url "http://localhost:8000/json"

# Results:
# ============================================================
# Test "unnamed" running - Run: RID001
# 
# 2023/12/28 15:10:19 
# Test:               unnamed
# Run ID:             RID001
# Requests target:    100
# Concurrency level:  1
# 
#  100% |████████████████████████████████████████████████████████████| (100/100, 8 requests/s) [14s]  
# 
# 2023/12/28 15:10:33 
# Requests sent: 100
# Average RPS: 7
# Response codes received: 
#   1xx: 0 | 2xx: 100 | 3xx: 0 | 4xx: 0 | 5xx: 0 | Unknown: 0
# ============================================================
```

### Adding concurrency

```bash
# 100 requests for a /json resource served at localhost:8000
# 5 batches of 20 requests performed concurently

./blowhole -n 100 -c 5 -url "http://localhost:8000/json"
```

## More options

### Specifying a run ID

```bash
# 100 requests for a /json resource served at localhost:8000
# 5 batches of 20 requests performed concurently
# Each request contains a unique "id: RID012.UIDxxxxx.CIDxxxxxx" header

./blowhole -n 100 -c 5 -url "http://localhost:8000/json" -run 12
```

### Output to a file

Results can be written to a local file using the `-o` option.
Two lines will be logged for each run, with comma-separated values in each line as follows:

```
Line 1: Timestamp test_name,run_id,n,c
Line 2: Tiemstamp requests_sent,average_rps,failed_requests
```


```bash
# 100 requests for a /json resource served at localhost:8000
# 5 batches of 20 requests performed concurently
# Write results to a file named out.log

./blowhole -n 30000 -c 1000 -url "http://localhost:8000/json" -o "out.log"

# Results:
# ============================================================
# Test "unnamed" running - Run: RID001
# 
# 
#  100% |████████████████████████████████████████████████████████████| (30000/30000, 7493 requests/s) [4s]   
# 
# Error count:
#   + 80: "the server closed connection before (...)"
#   + 1: "timeout"
# ============================================================

# Contents of "out.log"
# 2023/12/28 15:17:26 test unnamed,RID001,30000,1000
# 2023/12/28 15:17:30 30000,7089,81
```

### Tweak client

Blowhole uses a client from the [fasthttp](https://github.com/valyala/fasthttp) library.
Currently, there are three parameters that can be adjusted for this client: `maxconn`, `wtimeout`, and `rtimeout`

```bash
# 100 requests for a /json resource served at localhost:8000
# Use up to 3 connections per host
# Wait for up to 2 seconds before a request timeout
# Wait for up to 1 second before a response timeout

./blowhole -n 100 -url "http://localhost:8000/json" -maxconn 3 -wtimeout 2000 -rtimeout 1000
```

### Distributed mode

Blowhole can perform the requests in distributed mode, for those times when you just need more cowbell.

In order to run blowhole in distributed mode, we need to specify the `-distributed` option:
```bash
# 100 requests for a /json resource served at localhost:8000
# Perform requests using distributed clients
# Use instance as coordinator for distributed workers

./blowhole -n 100 -url "http://localhost:8000/json" -distributed
```

Now that we have a coordinator, we need workers. For this, we need to specify the `-worker` flag in
addition to the `-distributed` one when running blowhole from other machines:
```bash
# 100 requests for a /json resource served at localhost:8000
# Perform requests using distributed clients
# Use instance as a distributed worker

./blowhole -n 100 -url "http://localhost:8000/json" -distributed -worker
```

### Batched runs
```bash
# Perform tests specified in "sample.yml" sequentially

./blowhole -f "sample.yml"
```
```bash
# Perform tests specified in "sample.yml" sequentially
# Write results to a file named out.log

./blowhole -f "sample.yml" -o "out.log"
```


## Options reference:

```go
  -url:         string  Target URL to perform requests to               (default "http://localhost:8000/")
  -n:           int     Number of requests to perform                   (default 1)
  -c:           int     Number of concurrent connections                (default 1)
  -run:         int     Run number to use as Run ID in "id" header      (default 1)
  -o:           string  Results are written to this path                (default "stdout")
  -distributed: bool    Distributed clients are used when set to true   (default false)
  -worker:      bool    Run as a distributed worker when set to true    (default false)
  -maxconn:     int     Maximum number of connections per each host     (default 1000)
  -wtimeout:    int     Maximum duration to write full request in ms    (default 500)
  -rtimeout:    int     Maximum duration to read full response in ms    (default 500)
  -file:        string  Path of YAML file describing a batch of runs
```


---
<small>&copy; 2016-2023 <a href="https://resurface.io">Graylog, Inc.</a></small>
