package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var client *fasthttp.HostClient
var jobQueue chan Job
var stats []*Stat

var hostAddr, dataDir *string
var httpConnections, httpTimeout *int
var waitGroup sync.WaitGroup

type Stat struct {
	Method        string
	Lck           sync.Mutex
	TotalLatency  time.Duration
	TotalElapsed  time.Duration
	RequestsCount int
	ConnErrors    int
	ContentErrors int
}

type Job struct {
	URL       string
	Validator func(err error, latency time.Duration, respCode int, bodData []byte, stat *Stat)
	Stat      *Stat
}

func main() {
	hostAddr = flag.String("host", "127.0.0.1:8080", "Base URL of search application server")
	dataDir = flag.String("datadir", "data", "Dir, where datafiles are located")
	httpConnections = flag.Int("conn", 2, "Number of simulatenous connections")
	httpTimeout = flag.Int("timeout", 10, "Requests timeout in seconds")
	flag.Parse()

	initWorkers()
	log.Println("Start ")
	makeMediaItemsJobs()
	makeSearchJobs()
	makeEPGJobs()
	log.Println("Finish ")
	teardownWorkers()
	printStats()

}

func makeEPGJobs() {
	stat := &Stat{Method: "/api/v1/epg"}
	stats = append(stats, stat)
	tStart := time.Now()

	for i := 0; i < 30000; i++ {
		jobQueue <- Job{
			fmt.Sprintf("http://%s/%s", *hostAddr, "/api/v1/epg?limit=0"),
			respValidator,
			stat,
		}
	}
	stat.Lck.Lock()
	defer stat.Lck.Unlock()
	stat.TotalElapsed = time.Now().Sub(tStart)
}

func makeMediaItemsJobs() {
	stat := &Stat{Method: "/api/v1/media_items"}
	stats = append(stats, stat)
	tStart := time.Now()

	for i := 0; i < 30000; i++ {
		jobQueue <- Job{
			fmt.Sprintf("http://%s/%s", *hostAddr, "/api/v1/media_items?limit=1"),
			respValidator,
			stat,
		}
	}
	stat.Lck.Lock()
	defer stat.Lck.Unlock()
	stat.TotalElapsed = time.Now().Sub(tStart)

}

func makeSearchJobs() {
	stat := &Stat{Method: "/api/v1/search"}
	stats = append(stats, stat)
	tStart := time.Now()

	for i := 0; i < 30000; i++ {
		jobQueue <- Job{
			fmt.Sprintf("http://%s/%s", *hostAddr, "/api/v1/search?limit=1&query=term"),
			respValidator,
			stat,
		}
	}
	stat.Lck.Lock()
	defer stat.Lck.Unlock()
	stat.TotalElapsed = time.Now().Sub(tStart)

}

func respValidator(err error, latency time.Duration, respCode int, bodyData []byte, stat *Stat) {
	stat.Lck.Lock()
	defer stat.Lck.Unlock()
	stat.RequestsCount++
	if err != nil {
		stat.ConnErrors++
		return
	}
	stat.TotalLatency += latency
}

func initWorkers() {
	jobQueue = make(chan Job, 100)

	client = &fasthttp.HostClient{Addr: *hostAddr}

	for i := 0; i < *httpConnections; i++ {
		waitGroup.Add(1)
		go func() {
			for job := range jobQueue {
				doRequest(job)
			}
			waitGroup.Done()
		}()
	}
}

func teardownWorkers() {
	close(jobQueue)
	waitGroup.Wait()
}

func doRequest(job Job) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(job.URL)

	resp := fasthttp.AcquireResponse()
	t := time.Now()
	err := client.DoTimeout(req, resp, time.Duration(*httpTimeout)*time.Second)

	if job.Validator != nil {
		job.Validator(err, time.Now().Sub(t), resp.StatusCode(), resp.Body(), job.Stat)
	}
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
}

func printStat(stat *Stat) {
	fmt.Printf(
		" %-30s%10d%10d%16v%16d%16d\n",
		stat.Method,
		stat.RequestsCount,
		int(float64(stat.RequestsCount)/stat.TotalElapsed.Seconds()),
		stat.TotalLatency/time.Duration(stat.RequestsCount),
		stat.ConnErrors,
		stat.ContentErrors,
	)
}

func printStats() {
	fmt.Printf(" %-30s%10s%10s%16s%16s%16s\n", "Method", "Requests", "RPS", "Avg Latency", "Socket errors", "Сontent errors")
	fmt.Printf(" %-30s%10s%10s%16s%16s%16s\n", "------", "--------", "---", "-----------", "-------------", "--------------")
	for _, s := range stats {
		printStat(s)
	}
}
