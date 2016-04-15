package core

import (
	"github.com/hashicorp/golang-lru"
	"github.com/jbrady42/crawl/resolve"
	"github.com/juju/ratelimit"
)

type Crawler struct {
	UserAgent    string
	WorkerCount  int
	GroupByHost  bool
	RateLimitMB  float64
	RateBucket   *ratelimit.Bucket
	MaxPageBytes int
	IgnoreRobots bool
	resolver     *resolve.Resolver
	robotsCache  *lru.Cache
}

func NewCrawler(workers int, groupHost bool, servers []string) *Crawler {

	crawler := &Crawler{
		UserAgent:    "Smith",
		WorkerCount:  workers,
		GroupByHost:  groupHost,
		RateLimitMB:  0.0,
		RateBucket:   nil,
		MaxPageBytes: -1,
	}

	robotCache, _ := lru.New(500)
	crawler.robotsCache = robotCache

	crawler.resolver = resolve.NewWithServers(servers)

	// Setup rate limiting
	//SetRateLimited(0.0)

	return crawler
}
