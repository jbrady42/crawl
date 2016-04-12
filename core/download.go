package core

import (
	"bytes"
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
)

var defaultTimeout = time.Duration(60 * time.Second)
var hostCrawlDelay = time.Duration(1 * time.Second)
var maxBatchItems = 50

type DownloadWorker struct {
	crawler     *Crawler
	client      *http.Client
	resolver    *dns_resolver.DnsResolver
	currentInfo *DownloadInfo
}

type DownloadInfo struct {
	Url string
	IP  net.IP
}

func newDownloadInfo(s string) *DownloadInfo {
	var ip net.IP
	var urlStr string

	parts := strings.Split(s, "\t")
	if len(parts) == 2 {
		ip = net.ParseIP(parts[1])
		urlStr = parts[0]
	} else {
		urlStr = s
	}

	return &DownloadInfo{Url: urlStr, IP: ip}
}

func (t *Crawler) Download(inQ chan string, outQ chan *data.PageResult) {
	var wg sync.WaitGroup

	var infoQ chan *DownloadInfo
	var batchQ chan chan *DownloadInfo

	if t.GroupByHost {
		batchQ = make(chan chan *DownloadInfo)
		go toDownloadInfoBatches(inQ, batchQ)
	} else {
		infoQ = make(chan *DownloadInfo)
		go toDownloadInfo(inQ, infoQ)
	}

	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)
		if t.GroupByHost {
			go t.launchBatchDownloadWorker(batchQ, outQ, &wg)
		} else {
			go t.launchDownloadWorker(infoQ, outQ, &wg)
		}
	}

	log.Println("Waiting on workers")
	wg.Wait()
	log.Println("Exiting Download")
}

func (t *Crawler) launchBatchDownloadWorker(batchQ chan chan *DownloadInfo, outQ chan *data.PageResult, wg *sync.WaitGroup) {
	for q := range batchQ {
		// Add for the extra done in worker
		wg.Add(1)
		t.launchDownloadWorker(q, outQ, wg)
	}
	wg.Done()
}

func (t *Crawler) launchDownloadWorker(infoQ chan *DownloadInfo, outQ chan *data.PageResult, wg *sync.WaitGroup) {
	log.Println("Worker starting")
	resolver := DefaultResolver()
	// Build worker first
	worker := DownloadWorker{t, nil, resolver, nil}
	// Create and add client
	client := getHttpClient(resolver, &worker)
	worker.client = client

	worker.downloadWorker(infoQ, outQ)
	log.Println("Worker finished")
	wg.Done()
}

func toDownloadInfo(inQ chan string, outQ chan *DownloadInfo) {
	for s := range inQ {
		info := newDownloadInfo(s)
		outQ <- info
	}
	close(outQ)
}

func toDownloadInfoBatches(inQ chan string, outQ chan chan *DownloadInfo) {
	infoQ := make(chan *DownloadInfo, maxBatchItems)

	var currentHost net.IP
	var lastHost net.IP
	first := true

	for s := range inQ {
		info := newDownloadInfo(s)
		currentHost = info.IP

		if first {
			first = false
		}
		if lastHost != nil && bytes.Compare(currentHost, lastHost) != 0 {
			// Finish and enqueue batch

			outQ <- infoQ
			close(infoQ)
			// Start new batch
			log.Println("Adding batch ", lastHost)
			first = true
			infoQ = make(chan *DownloadInfo, maxBatchItems)
		}

		infoQ <- info
		lastHost = currentHost

	}
	// Check for any left over
	if len(infoQ) > 0 {
		close(infoQ)
		outQ <- infoQ
		log.Println("Adding last batch ")
	}

	close(outQ)
}

func (t *DownloadWorker) downloadWorker(inQ chan *DownloadInfo, outQ chan *data.PageResult) {
	for info := range inQ {
		// Set info for dailer
		t.currentInfo = info

		page := t.downloadUrl(info.Url)
		outQ <- page

		// Sleep if needed
		if t.crawler.GroupByHost && len(inQ) > 0 {
			time.Sleep(hostCrawlDelay)
		}
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

	if t.currentInfo.IP == nil {
		resolved, err := resolv(t.resolver, hostPart)
		if err != nil {
			return nil, err
		}
		resolvedStr = resolved.String()
	} else {
		resolvedStr = t.currentInfo.IP.String()
		// log.Println("Using resolved ip", resolvedStr)
	}

	// Recombine port
	if len(parts) > 1 {
		resolvedStr += ":" + parts[1]
	}

	return net.Dial(network, resolvedStr)
}

func getHttpClient(resolver *dns_resolver.DnsResolver, worker *DownloadWorker) (client *http.Client) {
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
