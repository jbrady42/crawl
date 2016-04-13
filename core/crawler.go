package core

import (
	"github.com/hashicorp/golang-lru"
	"github.com/juju/ratelimit"
)

type Crawler struct {
	WorkerCount    int
	GroupByHost    bool
	RateLimitMB    float64
	RateBucket     *ratelimit.Bucket
	MaxPageBytes   int
	ResolveServers []string
	resolveCache   *lru.Cache
}

func NewCrawler(workers int, groupHost bool) *Crawler {

	crawler := &Crawler{
		WorkerCount:  workers,
		GroupByHost:  groupHost,
		RateLimitMB:  0.0,
		RateBucket:   nil,
		MaxPageBytes: -1,
	}

	cache, _ := lru.New(1000000)
	crawler.resolveCache = cache

	// Setup rate limiting
	//SetRateLimited(0.0)

	return crawler
}
