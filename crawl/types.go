package crawl

import (
	"net/http"
	"time"
)

type Page struct {
	Url       string
	Visited   bool
	Processed bool
	VisitedAt string
}

type FetchResult struct {
	Data    *PageData
	Success bool
	Message string
}

type PageData struct {
	Url        string
	Body       string
	Timestamp  string
	Status     string
	StatusCode int
	Proto      string
	Header     http.Header
	Trailer    http.Header
}

type CrawlOpts struct {
	Workers   int
	RateLimit float64
}

func NewPageData(url string, resp *http.Response, body []byte) *PageData {
	pd := PageData{
		Url:        url,
		Body:       string(body),
		Timestamp:  (time.Now()).String(),
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Proto:      resp.Proto,
		Header:     resp.Header,
		Trailer:    resp.Trailer,
	}
	return &pd
}

func NewFailedResult(url string, reason string) *PageData {
	pd := PageData{
		Url:        url,
		Body:       "",
		Timestamp:  (time.Now()).String(),
		Status:     "",
		StatusCode: -1,
		Proto:      "",
		Header:     nil,
		Trailer:    nil,
	}

	fr := FetchResult{
		Data:    &pd,
		Success: false,
		Message: "failed: " + reason,
	}
	_ = fr
	return &pd
}
