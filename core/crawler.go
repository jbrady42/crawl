package core

import (
	"github.com/hashicorp/golang-lru"
	"github.com/juju/ratelimit"
)

type Crawler struct {
	UserAgent      string
	WorkerCount    int
	GroupByHost    bool
	RateLimitMB    float64
	RateBucket     *ratelimit.Bucket
	MaxPageBytes   int
	ResolveServers []string
	IgnoreRobots   bool
	resolveCache   *lru.Cache
	robotsCache    *lru.Cache
}

func NewCrawler(workers int, groupHost bool) *Crawler {

	crawler := &Crawler{
		UserAgent:    "Smith",
		WorkerCount:  workers,
		GroupByHost:  groupHost,
		RateLimitMB:  0.0,
		RateBucket:   nil,
		MaxPageBytes: -1,
	}

	cache, _ := lru.New(1000000)
	robotCache, _ := lru.New(500)
	crawler.resolveCache = cache
	crawler.robotsCache = robotCache

	// Setup rate limiting
	//SetRateLimited(0.0)

	return crawler
}
