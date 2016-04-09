package crawl

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bogdanovich/dns_resolver"
	"github.com/jbrady42/crawl/data"
	"github.com/juju/ratelimit"
)

var defaultTimeout = time.Duration(60 * time.Second)

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

func (t *Crawler) Download(inQ chan string, outQ chan *data.PageResult) {
	var wg sync.WaitGroup
	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)
		go func() {
			resolver := DefaultResolver()
			worker := DownloadWorker{t, nil, resolver, nil}
			client := getClient(resolver, &worker)
			worker.client = client

			worker.downloadWorker(inQ, outQ)
			wg.Done()
		}()
	}
	wg.Wait()
}

func getClient(resolver *dns_resolver.DnsResolver, worker *DownloadWorker) (client *http.Client) {
	trans := &http.Transport{
		Dial: func(network, address string) (net.Conn, error) {
			return worker.dial(network, address)
		},
		TLSHandshakeTimeout: 40 * time.Second,
		DisableKeepAlives:   true,
	}

	// client := &http.Client{}
	client = &http.Client{
		Timeout:   defaultTimeout,
		Transport: trans,
	}

	return client
}

type DownloadWorker struct {
	crawler    *Crawler
	client     *http.Client
	resolver   *dns_resolver.DnsResolver
	resolvedIp net.IP
}

func (t *DownloadWorker) downloadWorker(inQ chan string, outQ chan *data.PageResult) {
	for s := range inQ {
		// resolv := NewResolver()
		// client := getClient(resolv)
		parts := strings.Split(s, "\t")
		if len(parts) == 2 {
			t.resolvedIp = net.ParseIP(parts[1])
			s = parts[0]
		}
		page := t.downloadUrl(s)
		outQ <- page
	}
}

func (t *DownloadWorker) downloadUrl(url string) (page *data.PageResult) {

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("Error downloading %s : %s\n", url, err)
		return data.NewFailedResult(url, err.Error())
	}

	var body []byte
	var reader io.Reader
	reader = resp.Body

	// Set up partial reading
	if t.crawler.MaxPageBytes > 0 {
		reader = io.LimitReader(resp.Body, int64(t.crawler.MaxPageBytes))
	}

	body, err = ioutil.ReadAll(reader)
	if err != nil {
		log.Println("Error reading response body")
	}
	// Close con
	resp.Body.Close()

	pd := data.NewPageData(url, resp, body)
	log.Printf("Download complete: %s \n", url)
	return pd
}

func (t *DownloadWorker) dial(network, address string) (net.Conn, error) {
	parts := strings.Split(address, ":")
	hostPart := parts[0]

	var resolvedStr string

	if t.resolvedIp == nil {
		resolved, err := resolv(t.resolver, hostPart)
		if err != nil {
			return nil, err
		}
		resolvedStr = resolved.String()
	} else {
		resolvedStr = t.resolvedIp.String()
		// log.Println("Using resolved ip", resolvedStr)
	}

	// Recombine port
	if len(parts) > 1 {
		resolvedStr += ":" + parts[1]
	}

	return net.Dial(network, resolvedStr)
}
