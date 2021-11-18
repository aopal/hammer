package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"runtime"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/sync/semaphore"
)

type hammerSettings struct {
	concurrencyFactor int
	client            *http.Client
	delay             time.Duration
	sem               *semaphore.Weighted
	requestHeaders    http.Header
	urls              []string
	useHTTP2          bool
}

func getSettings() *hammerSettings {
	settings := hammerSettings{
		requestHeaders: make(http.Header),
	}

	// set flags
	flag.IntVar(&settings.concurrencyFactor, "c", runtime.GOMAXPROCS(0), "Concurrency factor, number of requests to make concurrently. Defaults to the value of runtime.GOMAXPROCS(0)")
	flag.DurationVar(&settings.delay, "d", 0, "Delay to wait after making a request")
	flag.BoolVar(&settings.useHTTP2, "http2", false, "Use HTTP/2 for requests")
	flag.Func("header", "Set a request header in the form \"header-name: header-value\" (multiple invocations allowed)", func(str string) error {

		h := strings.Split(str, ":")
		if len(h) != 2 {
			return fmt.Errorf("Header must be in the form \"header-name: header-value\"")
		}

		settings.requestHeaders.Add(h[0], h[1])
		return nil
	})

	// parse flags
	flag.Parse()

	// post process flags
	settings.urls = flag.Args()

	settings.sem = semaphore.NewWeighted(int64(settings.concurrencyFactor))
	settings.client = &http.Client{}
	if settings.useHTTP2 {
		settings.client.Transport = &http2.Transport{}
	}

	return &settings
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <url> [urls...]\n", os.Args[0])
	flag.PrintDefaults()
}

func logStats(n int, startTime time.Time) {
	t := time.Now()
	elapsed := t.Sub(startTime)
	avgRate := float64(n) / float64(elapsed/time.Millisecond) * 1000.0 * 60.0

	fmt.Printf("\rCompleted at %d total requests in %v (average %0.2f requests/minute)...", n, elapsed.Round(1000*time.Millisecond), avgRate)
}

// Make a request against a url
func doRequest(settings *hammerSettings, i int) {
	defer settings.sem.Release(1)

	url := settings.urls[i%len(settings.urls)]
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Close = false
	req.Header = settings.requestHeaders.Clone()

	res, err := settings.client.Do(req)

	if err != nil {
		fmt.Println(err)
		return
	}

	if res.StatusCode != 200 {
		fmt.Println("\nReceived non 200 response:", res.StatusCode, url)
	}

	// read the response body
	go func() {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}()

	if settings.delay != 0 {
		time.Sleep(settings.delay)
	}
}

func loadTest(settings *hammerSettings) {
	fmt.Println("max concurrent requests:", settings.concurrencyFactor)
	fmt.Println("max parallel goroutines:", runtime.GOMAXPROCS(0))

	ctx := context.Background()
	start := time.Now()

	for i := 0; true; i++ {
		if err := settings.sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v", err)
			break
		}

		go doRequest(settings, i)

		go logStats(i, start)
	}
}

func main() {
	flag.Usage = usage
	settings := getSettings()
	loadTest(settings)
}
