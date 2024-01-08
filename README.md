# resurfaceio-blowhole

Load test servers with unique requests.

```
        .
       ":"
     ___:____     |"\/"|
   ,'        `.    \  /
   |  O        \___/  |
 ~^~^~^~^~^~^~^~^~^~^~^~^~ 

ASCII art by Riitta Rasimus
```

Blowhole is a command line tool written in Go to perform [unique requests](#unique-requests) for a given URL.

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

# Unique requests

Uniqueness is guaranteed by adding an "id" header to each request with the following format:

```
"id: RIDxxx.UIDxxxxx.CIDxxxxxx"
```

Where:

 - `RID`: run ID - same for each run. Increases for each run in a batch of runs (see [batched runs](#batched-runs)). Starts with `RID001`
 - `UID`: user ID - same for each "user" (the concurrency level corresponds to the number of goroutines sending a portion of the total requests).
 - `CID`: count ID - different with each request performed for a given user.


For example:

```bash
./blowhole -n 10 -c 2 -url http://localhost:8080/get
```

One run of 10 requests and 2 concurrent users sends 10 requests with the following 10 unique `id` headers:

```
Header  1: "id: RID001.UID00000.CID000000"
Header  2: "id: RID001.UID00000.CID000001"
Header  3: "id: RID001.UID00000.CID000002"
Header  4: "id: RID001.UID00000.CID000003"
Header  5: "id: RID001.UID00000.CID000004"
Header  6: "id: RID001.UID00001.CID000000"
Header  7: "id: RID001.UID00001.CID000001"
Header  8: "id: RID001.UID00001.CID000002"
Header  9: "id: RID001.UID00001.CID000003"
Header 10: "id: RID001.UID00001.CID000004"
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

Two lines will be logged for each run, with comma-separated values in each line as follows:

```
Line 1: Start_timestamp test_name,run_id,n,c
Line 2: End_timestamp requests_sent,average_rps,failed_requests
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

A collection of runs can be bundled as a test. All runs in a test can be specified in a YAML document as follows:

`sample.yml`
```yaml
name: mytest
url: http://localhost:8080/json
runs:
  - requests: 5
    concurrency: 3
  - requests: 900
    concurrency: 10
  - requests: 100
    concurrency: 100
    url: http://myurl/foo
  - id: RID999
    requests: 1000
    concurrency: 10
```

A test named "mytest" is specified above, with 4 runs equivalent to the following commands:

```bash
./blowhole -n 5    -c 3   -url http://localhost:8080/json -run 1 ;
./blowhole -n 900  -c 10  -url http://localhost:8080/json -run 2 ;
./blowhole -n 100  -c 100 -url http://myurl/foo           -run 3 ;
./blowhole -n 1000 -c 10  -url http://localhost:8080/json -run 999
```

Instead, all runs defined in the `sample.yml` file can be performed in one go like so:

```bash
# Perform tests specified in "sample.yml" sequentially

./blowhole -f "sample.yml"
```

See [the batch spec reference](#batch-yaml-spec-reference) for more information about each field in the spec.

Each field in the run spec overrides any command-line option, except for those regarding the fasthttp client (`maxconn`, `wtimeout`, `rtimeout`)

```bash
# Perform tests specified in "sample.yml" sequentially, ignoring value passed with -n option
# Wait for up to 100 milliseconds before a response timeout

./blowhole -f "sample.yml" -n 99 -rtimeout 100
```

If a field is not declared in the spec, it can be passed as a command-line option:

```bash
# Perform tests specified in "sample.yml" sequentially
# Write results to a file named out.log

./blowhole -f "sample.yml" -o "out.log"
```


## Command-line options reference:

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

## Batch YAML spec reference:

```go
// top-level fields for each test spec
name         string     Name of the test. If not specified, defaults to "unnamed"
url          string     Target URL. Required
output       string     Output file path. If not specified, results are written to stdout
distributed  bool       Distributed clients are used when set to true 
worker       bool       Run as a distributed worker when set to true
runs         []runConf  Collection of runs

// field for each run (runConf)
requests    int         Number of requests to perform
concurrency int         Number of concurrent connections
url         string      Target URL to perform requests to
id          string      String to replace `RIDxxx` substring in "id" header
```


---
<small>&copy; 2016-2024 <a href="https://resurface.io">Graylog, Inc.</a></small>
