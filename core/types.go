package core

import (
	"net"
	"strings"
)

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
