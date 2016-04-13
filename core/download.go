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
var maxBatchItems = 500

type DownloadWorker struct {
	crawler     *Crawler
	client      *http.Client
	resolver    *dns_resolver.DnsResolver
	currentInfo *DownloadInfo
}

func (t *Crawler) Download(inQ <-chan string, outQ chan<- *data.PageResult) {
	if t.GroupByHost {
		t.downloadPerHost(inQ, outQ)
	} else {
		t.downloadAll(inQ, outQ)
	}
}

func (t *Crawler) downloadAll(inQ <-chan string, outQ chan<- *data.PageResult) {

	var wg sync.WaitGroup
	infoQ := make(chan *DownloadInfo)

	go toDownloadInfo(inQ, infoQ)

	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)

		go t.launchDownloadWorker(infoQ, outQ, &wg)
	}

	log.Println("Waiting on workers")
	wg.Wait()
	log.Println("Exiting Download")
}

func (t *Crawler) downloadPerHost(inQ <-chan string, outQ chan<- *data.PageResult) {
	var wg sync.WaitGroup

	infoQ := make(chan *DownloadInfo)

	qMap := make(map[string]chan *DownloadInfo)

	go toDownloadInfo(inQ, infoQ)

	closeChan := make(chan string, t.WorkerCount)

	// Worker timeout watcher
	closeMap := make(map[string]struct{})
	go func() {
		for {
			time.Sleep(1 * time.Second)
			for host, q := range qMap {
				if _, closed := closeMap[host]; len(q) == 0 && !closed {
					log.Println("Closing download: ", host)
					close(q)
					// Mark as closing
					closeMap[host] = struct{}{}
				}
			}
		}
	}()

	for info := range infoQ {
		log.Println(len(qMap))
		groupKey := info.IP.String()
		q, found := qMap[groupKey]

		if !found {
			log.Println("Number of workers ", len(qMap))
			if len(qMap) >= t.WorkerCount {
				log.Println("Waiting for free workers")

				// Wait for worker to finish
				doneGroup := <-closeChan
				log.Println("Removing worker ", doneGroup)

				// Clear host from maps
				delete(qMap, doneGroup)
				delete(closeMap, doneGroup)
			}

			log.Println("Adding new worker", groupKey)
			q = make(chan *DownloadInfo, maxBatchItems)
			qMap[groupKey] = q

			// Notify of finish when worker exits
			wg.Add(1)
			go func(host string) {
				t.launchDownloadWorker(q, outQ, &wg)
				closeChan <- host
			}(groupKey)

			// Other way of doing the timeout with per worker
			// // Timeout handler for empty q
			// go func(host string) {
			// 	q, ok := qMap[groupKey]

			// 	for ok {
			// 		log.Println("Watcher", len(q))
			// 		time.Sleep(1 * time.Second)
			// 		q, ok = qMap[groupKey]

			// 		if len(q) == 0 {
			// 			log.Println("Closing download: ", host)
			// 			close(q)
			// 			return
			// 		}
			// 	}

			// }(groupKey)
		}

		q <- info
	}

	log.Println("Waiting on workers")
	wg.Wait()
	log.Println("Exiting Download")
}

func (t *Crawler) launchBatchDownloadWorker(batchQ <-chan chan *DownloadInfo, outQ chan<- *data.PageResult, wg *sync.WaitGroup) {
	for q := range batchQ {
		// Add for the extra done in worker
		wg.Add(1)
		t.launchDownloadWorker(q, outQ, wg)
	}
	wg.Done()
}

func (t *Crawler) launchDownloadWorker(infoQ <-chan *DownloadInfo, outQ chan<- *data.PageResult, wg *sync.WaitGroup) {
	log.Println("Worker starting")
	resolver := DefaultResolver()
	// Build worker first
	worker := DownloadWorker{t, nil, resolver, nil}
	// Create and add client
	client := httpClient(resolver, &worker)
	worker.client = client

	worker.downloadUrls(infoQ, outQ)
	log.Println("Worker finished")
	wg.Done()
}

func (t *DownloadWorker) downloadUrls(inQ <-chan *DownloadInfo, outQ chan<- *data.PageResult) {
	for info := range inQ {
		// Set info for dialer
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
	if err != nil {
		log.Print("Error creating request: ", url)
		log.Printf("%s\n", err)
		return data.NewFailedResult(url, err.Error())
	}
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
		resolved, err := t.crawler.resolve(t.resolver, hostPart)
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

func httpClient(resolver *dns_resolver.DnsResolver, worker *DownloadWorker) (client *http.Client) {
	trans := &http.Transport{
		Dial: func(network, address string) (net.Conn, error) {
			return worker.dial(network, address)
		},
		TLSHandshakeTimeout: 40 * time.Second,
		DisableKeepAlives:   true,
	}

	client = &http.Client{
		Timeout:   defaultTimeout,
		Transport: trans,
	}
	return client
}

// Input handling

func toDownloadInfo(inQ <-chan string, outQ chan<- *DownloadInfo) {
	for s := range inQ {
		info := newDownloadInfo(s)
		outQ <- info
	}
	close(outQ)
}

func toDownloadInfoBatches(inQ <-chan string, outQ chan<- chan *DownloadInfo) {
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
		outQ <- infoQ
		close(infoQ)
		log.Println("Adding last batch ")
	}

	close(outQ)
}
