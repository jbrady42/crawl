package core

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
	"github.com/jbrady42/syncmap"
	"github.com/temoto/robotstxt-go"
)

const (
	defaultTimeout    = time.Duration(60 * time.Second)
	hostCrawlDelay    = time.Duration(1 * time.Second)
	hostWorkerTimeout = 3 * time.Second
	maxBatchItems     = 1000
)

type DownloadWorker struct {
	crawler     *Crawler
	client      *http.Client
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

func writeStats(workerCount, urlCount int) {
	stats := fmt.Sprintf("%d\t%d", workerCount, urlCount)
	err := ioutil.WriteFile("/tmp/crawlstats", []byte(stats), 0644)

	if err != nil {
		log.Println("Can not write stats file")
	}
}

func (t *Crawler) downloadPerHost(inQ <-chan string, outQ chan<- *data.PageResult) {
	var wg sync.WaitGroup
	infoQ := make(chan *DownloadInfo)
	qMap := syncmap.New()
	closeChan := make(chan string, t.WorkerCount)

	go toDownloadInfo(inQ, infoQ)

	// // Worker timeout watcher
	closeMap := syncmap.New()
	go func() {
		for {
			time.Sleep(hostWorkerTimeout)
			count := 0
			for a := range qMap.Iter() {
				host := a.Key.(string)
				q := (a.Val).(chan *DownloadInfo)
				qLen := len(q)
				log.Println("worker: ", host, " length: ", qLen)
				count += qLen
				if qLen == 0 && !closeMap.Has(host) {
					log.Println("Closing download: ", host)
					close(q)
					// Mark as closing
					closeMap.Set(host, struct{}{})
				}
			}
			writeStats(qMap.Len(), count)
		}
	}()

	for info := range infoQ {
		var q chan *DownloadInfo
		groupKey := info.IP.String()

		// Find worker for host
		tmp, found := qMap.Get(groupKey)
		closing := closeMap.Has(groupKey)
		if found && !closing {
			q = tmp.(chan *DownloadInfo)
		} else {
			// Create new worker
			for qMap.Len() >= t.WorkerCount {
				log.Println("Waiting for free workers")
				// Wait for worker to finish
				<-closeChan
			}

			log.Println("Adding new worker", groupKey)
			q = make(chan *DownloadInfo, maxBatchItems)
			qMap.Set(groupKey, q)

			wg.Add(1)
			go func(host string, inQ chan *DownloadInfo) {
				time.Sleep(100 * time.Millisecond)
				t.launchDownloadWorker(inQ, outQ, &wg)

				// Clear host from maps
				log.Println("Removing worker ", host)
				qMap.Delete(host)
				closeMap.Delete(host)

				// Notify of finish when worker exits
				closeChan <- host
			}(groupKey, q)

			// // Other way of doing the timeout with per worker
			// // Timeout handler for empty q
			// go func(host string) {
			// 	for q, ok := qMap[groupKey]; ok; q, ok = qMap[groupKey] {
			// 		log.Println("Watcher", len(q))
			// 		time.Sleep(hostWorkerTimeout)
			// 		q, ok = qMap[groupKey]

			// 		if len(q) == 0 {
			// 			log.Println("Closing download: ", host)
			// 			close(q)
			// 			return
			// 		}
			// 	}

			// }(groupKey)
		}
		// Add into to correct host queue
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
	defer wg.Done()
	// Build worker first
	worker := DownloadWorker{t, nil, nil}
	// Create and add client
	client := httpClient(&worker)
	worker.client = client

	worker.downloadUrls(infoQ, outQ)
	log.Println("Worker finished")
}

func (t *DownloadWorker) downloadUrls(inQ <-chan *DownloadInfo, outQ chan<- *data.PageResult) {
	for info := range inQ {
		// Set info for dialer
		t.currentInfo = info
		urlStr := info.Url

		var page *data.PageResult

		if t.crawler.IgnoreRobots || t.allowedByRobots(urlStr) {
			page = t.downloadUrl(urlStr)
		} else {
			log.Println("Blocked by robots")
			page = data.NewFailedResult(urlStr, "Blocked by robots")
		}

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
	if resp != nil {
		defer resp.Body.Close()
	}
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
	//resp.Body.Close()

	pd := data.NewPageData(url, resp, body)
	log.Printf("Download complete: %s \n", url)
	return pd
}

func (t *DownloadWorker) dial(network, address string) (net.Conn, error) {
	parts := strings.Split(address, ":")
	hostPart := parts[0]

	var resolvedStr string

	if t.currentInfo.IP == nil {
		resolved, _, err := t.crawler.resolver.Resolve(hostPart)
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

func httpClient(worker *DownloadWorker) (client *http.Client) {
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

func (t *DownloadWorker) robotsTxt(inUrl *url.URL) *robotstxt.RobotsData {
	roboUrl := util.RobotsUrl(inUrl).String()

	tmp, ok := t.crawler.robotsCache.Get(roboUrl)
	if !ok {
		result := t.downloadUrl(roboUrl)
		text := result.Data.Body

		robots, err := robotstxt.FromString(text)
		if err != nil {
			log.Println("Robots error: ", err)
			return nil
		}
		t.crawler.robotsCache.Add(roboUrl, robots)
		return robots
	}
	// log.Println("Cached robots")
	robots := tmp.(*robotstxt.RobotsData)
	return robots
}

func (t *DownloadWorker) allowedByRobots(inUrl string) (allowed bool) {
	robotsUrl, _ := url.Parse(inUrl)
	// Fetch robots
	robots := t.robotsTxt(robotsUrl)
	if robots == nil {
		log.Println("No robots")
		return false
	}
	allowed = robots.TestAgent(robotsUrl.Path, t.crawler.UserAgent)
	return allowed
}
