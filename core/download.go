package crawl

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jbrady42/crawl/data"
	"github.com/juju/ratelimit"
)

var defaultTimeout = time.Duration(60 * time.Second)

type Crawler struct {
	WorkerCount  int
	GroupByHost  bool
	RateLimitMB  float64
	RateBucket   *ratelimit.Bucket
	MaxPageBytes int
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

func (t *Crawler) Download(inQ chan string, outQ chan *data.PageResult) {
	var wg sync.WaitGroup
	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		go func() {
			t.downloadWorker(inQ, outQ)
			wg.Done()
		}()
	}
	wg.Wait()
	// for s := range inQ {
	// 	page := downloadUrl(s)
	// 	outQ <- page
	// }
}

func (t *Crawler) downloadWorker(inQ chan string, outQ chan *data.PageResult) {
	for s := range inQ {
		page := t.downloadUrl(s)
		outQ <- page
	}
}

func (t *Crawler) downloadUrl(url string) (page *data.PageResult) {
	// client := &http.Client{}
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading %s : %s\n", url, err)
		return data.NewFailedResult(url, err.Error())
	}

	var body []byte
	var reader io.Reader
	reader = resp.Body

	// Set up partial reading
	if t.MaxPageBytes > 0 {
		reader = io.LimitReader(resp.Body, int64(t.MaxPageBytes))
	}

	body, err = ioutil.ReadAll(reader)
	if err != nil {
		log.Println("Error reading response body")
	}
	// Close con
	resp.Body.Close()

	pd := data.NewPageData(url, resp, body)
	return pd
}
