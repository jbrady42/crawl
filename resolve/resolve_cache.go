package resolve

import (
	"net"
	"time"
)

const cacheTTL = 20 * time.Minute

type cacheItem struct {
	host      string
	ip        net.IP
	cname     string
	expiresAt time.Time
}

func (t *cacheItem) expired() bool {
	return t.expiresAt.Before(time.Now())
}

func newCacheItem(host, cname string, ip net.IP) *cacheItem {
	item := &cacheItem{
		host:      host,
		ip:        ip,
		cname:     cname,
		expiresAt: time.Now().Add(cacheTTL),
	}
	return item
}
