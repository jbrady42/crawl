package core

import (
	"bytes"
	"encoding/json"
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
	hostWorkerTimeout = 1 * time.Second
	statsInterval     = 3 * time.Second
	maxBatchItems     = 1000
)

type DownloadWorker struct {
	crawler     *Crawler
	client      *http.Client
	currentInfo *DownloadInfo
}

type HostWorker struct {
	inQ     chan *DownloadInfo
	outQ    chan<- *data.PageResult
	key     string
	closing bool
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

func writeStats(stats CrawlStats) {
	// Build JSON
	data, err := json.Marshal(stats)
	if err != nil {
		log.Println("Error preparing stats")
		return
	}
	// Write to file
	err = ioutil.WriteFile("/tmp/crawl_stats", data, 0644)
	if err != nil {
		log.Println("Can not write stats file")
	}
}

func crwalStatsWorker(hostMap *syncmap.Map) {
	for {
		time.Sleep(statsInterval)
		var count, closeCount int
		countMap := make(map[string]int)
		for tmp := range hostMap.Iter() {
			var tmpCount int
			worker := tmp.Val.(*HostWorker)
			tmpCount = len(worker.inQ)
			if worker.closing {
				log.Println("worker:", worker.key, "close waiting")
				closeCount++
				tmpCount++
			}
			count += tmpCount
			countMap[worker.key] = tmpCount
		}
		stats := CrawlStats{
			Workers:     hostMap.Len(),
			Closing:     closeCount,
			Urls:        count,
			WorkerCount: countMap,
		}
		writeStats(stats)
	}
}

func workerCloseWatcher(worker *HostWorker) {
	for {
		time.Sleep(hostWorkerTimeout)
		qLen := len(worker.inQ)
		log.Println("worker: ", worker.key, "length:", qLen)
		if qLen == 0 {
			log.Println("Closing download:", worker.key)
			close(worker.inQ)
			worker.closing = true
			return
		}
	}
}

func (t *Crawler) downloadPerHost(inQ <-chan string, outQ chan<- *data.PageResult) {
	var wg sync.WaitGroup
	infoQ := make(chan *DownloadInfo)
	closeChan := make(chan string, t.WorkerCount)
	workerMap := syncmap.New()

	// Transform input
	go toDownloadInfo(inQ, infoQ)
	//Launch worker stats
	go crwalStatsWorker(workerMap)
	// Find or create worker
	for info := range infoQ {
		var q chan *DownloadInfo
		hostKey := info.IP.String()

		// Find worker for host
		tmp, found := workerMap.Get(hostKey)
		if found && !tmp.(*HostWorker).closing {
			q = tmp.(*HostWorker).inQ
		} else {
			// Create new worker
			for workerMap.Len() >= t.WorkerCount {
				log.Println("Waiting for free workers")
				// Wait for worker to finish
				<-closeChan
			}

			log.Println("Adding new worker", hostKey)
			q = make(chan *DownloadInfo, maxBatchItems)

			worker := &HostWorker{inQ: q, outQ: outQ, key: hostKey}
			workerMap.Set(hostKey, worker)

			wg.Add(1)
			go t.launchHostDownloadWorker(&wg, worker, workerMap, closeChan)

			// Worker empty timeout
			go workerCloseWatcher(worker)
		}

		// Add into to correct host queue
		q <- info
	}

	log.Println("Waiting on workers")
	wg.Wait()
	log.Println("Exiting Download")
}

func (t *Crawler) launchHostDownloadWorker(wg *sync.WaitGroup, worker *HostWorker, workerMap *syncmap.Map, closeChan chan string) {
	time.Sleep(100 * time.Millisecond)
	t.launchDownloadWorker(worker.inQ, worker.outQ, wg)

	// Clear host from maps
	workerMap.Delete(worker.key)
	log.Println("Removing worker", worker.key)
	// Notify of finish when worker exits
	closeChan <- worker.key
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
		resolved, _, err := t.crawler.Resolver.Resolve(hostPart)
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
