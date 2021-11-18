# hammer

A simple but powerful HTTP load-tester

## Installation

`go install github.com/aopal/hammer`

## Usage

```
Usage: hammer [options] <url> [urls...]
  -c int
        Concurrency factor, number of requests to make concurrently. Defaults to the value of runtime.GOMAXPROCS(0) (default 12)
  -d duration
        Delay to wait after making a request
  -header value
        Set a request header in the form "header-name: header-value" (multiple invocations allowed)
  -http2
        Use HTTP/2 for requests
```
