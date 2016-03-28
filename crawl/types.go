package crawl

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Page struct {
	Url       string
	Visited   bool
	Processed bool
	VisitedAt string
}

type PageResult struct {
	Data    *PageData
	Success bool
	Message string

	Links []*url.URL
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

func NewPageData(url string, resp *http.Response, body []byte) *PageResult {
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

	fr := PageResult{
		Data:    &pd,
		Success: true,
		Message: "",
	}
	return &fr
}

func NewFailedResult(url string, reason string) *PageResult {
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

	fr := PageResult{
		Data:    &pd,
		Success: false,
		Message: "failed: " + reason,
	}
	return &fr
}

func PageDataFromLine(line string) *PageResult {
	var page PageResult

	data := []byte(line)
	err := json.Unmarshal(data, &page)
	if err != nil {
		log.Println("Error marshaling line")
		log.Println(err)
	}
	return &page
}
