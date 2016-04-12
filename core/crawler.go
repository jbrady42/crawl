package core

import "github.com/juju/ratelimit"

type Crawler struct {
	WorkerCount    int
	GroupByHost    bool
	RateLimitMB    float64
	RateBucket     *ratelimit.Bucket
	MaxPageBytes   int
	ResolveServers []string
}

func NewCrawler(workers int, groupHost bool) *Crawler {

	crawler := &Crawler{
		WorkerCount:  workers,
		GroupByHost:  groupHost,
		RateLimitMB:  0.0,
		RateBucket:   nil,
		MaxPageBytes: -1,
	}

	// Setup rate limiting
	//SetRateLimited(0.0)

	return crawler
}
