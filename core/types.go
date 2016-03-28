package crawl

type Page struct {
	Url       string
	Visited   bool
	Processed bool
	VisitedAt string
}

type CrawlOpts struct {
	Workers   int
	RateLimit float64
}
