package core

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
)

var defaultTimeout = time.Duration(60 * time.Second)

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

	infoQ := make(chan *DownloadInfo)

	// To download info woker
	go toDownloadInfo(inQ, infoQ)

	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)

		go func() {
			resolver := DefaultResolver()
			// Build worker first
			worker := DownloadWorker{t, nil, resolver, nil}
			// Create and add client
			client := getHttpClient(resolver, &worker)
			worker.client = client

			worker.downloadWorker(infoQ, outQ)
			wg.Done()
		}()
	}
	wg.Wait()
}

func toDownloadInfo(inQ chan string, outQ chan *DownloadInfo) {
	for s := range inQ {
		info := newDownloadInfo(s)
		outQ <- info
	}
	close(outQ)
}

func (t *DownloadWorker) downloadWorker(inQ chan *DownloadInfo, outQ chan *data.PageResult) {
	for info := range inQ {
		// Set info for dailer
		t.currentInfo = info

		page := t.downloadUrl(info.Url)
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
