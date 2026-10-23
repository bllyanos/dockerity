// Command loadtest sends steady traffic to a URL while you deploy and exits
// non-zero if any request failed. See docs/runbooks/blue-green-check.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bllyanos/dockerity/internal/loadtest"
)

func main() {
	var o loadtest.Options
	flag.StringVar(&o.URL, "url", "", "URL to request (required)")
	flag.IntVar(&o.Rate, "rate", 50, "requests per second")
	flag.DurationVar(&o.Duration, "duration", time.Minute, "how long to send requests")
	flag.DurationVar(&o.Timeout, "timeout", 5*time.Second, "per-request timeout")
	flag.IntVar(&o.Workers, "workers", 8, "concurrent senders")
	flag.Parse()
	if o.URL == "" {
		flag.Usage()
		os.Exit(2)
	}

	if failed := run(o); failed > 0 {
		os.Exit(1)
	}
}

// run sends the traffic, prints the report and returns the failure count.
func run(o loadtest.Options) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Printf("Sending %d req/s to %s for %s (Ctrl-C to stop early)\n", o.Rate, o.URL, o.Duration)
	rep := loadtest.Run(ctx, o)
	fmt.Println(rep)
	for _, s := range rep.Samples {
		fmt.Println("  failure:", s)
	}
	return rep.Failed()
}
