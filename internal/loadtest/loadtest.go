// Package loadtest sends steady traffic to a URL and counts failures, to
// check that a blue-green deploy causes zero failed requests.
package loadtest

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Options configure a run.
type Options struct {
	URL      string
	Rate     int           // requests per second
	Duration time.Duration // how long to send requests
	Timeout  time.Duration // per request
	Workers  int           // concurrent senders
}

// Report summarizes a run.
type Report struct {
	Total   int
	OK      int
	Non2xx  int
	Errors  int
	Samples []string // a few failure descriptions
}

// Failed is the number of requests that did not succeed.
func (r Report) Failed() int { return r.Non2xx + r.Errors }

func (r Report) String() string {
	return fmt.Sprintf("total=%d ok=%d non-2xx=%d errors=%d", r.Total, r.OK, r.Non2xx, r.Errors)
}

const maxSamples = 10

// Run sends requests at the configured rate until the duration passes or ctx
// is cancelled.
func Run(ctx context.Context, o Options) Report {
	client := &http.Client{Timeout: o.Timeout}
	ctx, cancel := context.WithTimeout(ctx, o.Duration)
	defer cancel()

	jobs := make(chan struct{})
	var (
		mu  sync.Mutex
		rep Report
		wg  sync.WaitGroup
	)
	record := func(ok bool, non2xx bool, sample string) {
		mu.Lock()
		defer mu.Unlock()
		rep.Total++
		switch {
		case ok:
			rep.OK++
		case non2xx:
			rep.Non2xx++
		default:
			rep.Errors++
		}
		if sample != "" && len(rep.Samples) < maxSamples {
			rep.Samples = append(rep.Samples, sample)
		}
	}
	for range max(o.Workers, 1) {
		wg.Go(func() {
			for range jobs {
				res, err := client.Get(o.URL)
				if err != nil {
					record(false, false, time.Now().Format("15:04:05.000")+" "+err.Error())
					continue
				}
				_ = res.Body.Close()
				if res.StatusCode >= 200 && res.StatusCode < 300 {
					record(true, false, "")
				} else {
					record(false, true, fmt.Sprintf("%s HTTP %d", time.Now().Format("15:04:05.000"), res.StatusCode))
				}
			}
		})
	}

	tick := time.NewTicker(time.Second / time.Duration(max(o.Rate, 1)))
	defer tick.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-tick.C:
			select {
			case jobs <- struct{}{}:
			case <-ctx.Done():
				break loop
			}
		}
	}
	close(jobs)
	wg.Wait()
	return rep
}
