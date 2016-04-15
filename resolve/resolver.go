package resolve

import (
	"errors"
	"net"

	"github.com/bogdanovich/dns_resolver"
	"github.com/hashicorp/golang-lru"
)

type cacheItem struct {
	host  string
	ip    net.IP
	cname string
}

type Resolver struct {
	resolver     *dns_resolver.DnsResolver
	resolveCache *lru.Cache
}

func New() *Resolver {
	cache, _ := lru.New(1000000)
	res := defaultResolver()
	return &Resolver{res, cache}
}

func NewWithServers(servers []string) *Resolver {
	resolver := New()
	if len(servers) > 0 {
		resolver.resolver = newResolver(servers)
	}
	return resolver
}

func defaultResolver() (resolver *dns_resolver.DnsResolver) {
	resolver = dns_resolver.New([]string{"208.67.222.222", "208.67.220.220", "8.8.8.8", "8.8.4.4"})
	resolver.RetryTimes = 3
	resolver.ReuseConnection = true
	return resolver
}

func newResolver(servers []string) (resolver *dns_resolver.DnsResolver) {
	// Hand a copy because resolver mutates servers
	tmpServer := make([]string, len(servers))
	copy(tmpServer, servers)

	resolver = dns_resolver.New(tmpServer)
	resolver.ReuseConnection = true
	return resolver
}

// Resolve with the crawlers cache
func (t *Resolver) Resolve(host string) (resolved net.IP, cname string, err error) {
	// Hit cache first
	tmp, found := t.resolveCache.Get(host)
	if !found {

		// Do resolve
		resolved, cname, err = resolveWithCname(t.resolver, host)

		if err != nil {
			return nil, "", err
		} else {
			// resolved = newIP
			item := cacheItem{host, resolved, cname}
			t.resolveCache.Add(host, item)
		}
	} else {
		// log.Println("Resolve cached: ", host)
		item := tmp.(cacheItem)
		resolved = item.ip
		cname = item.cname
	}
	return resolved, cname, nil
}

func resolve(resolver *dns_resolver.DnsResolver, host string) (resolved net.IP, err error) {
	ip, err := resolver.LookupHost(host)
	if err != nil {
		return nil, err
	} else if len(ip) == 0 {
		return nil, errors.New("No results")
	} else {
		resolved = ip[0]
	}
	return resolved, nil
}

func resolveWithCname(resolver *dns_resolver.DnsResolver, host string) (resolved net.IP, name string, err error) {
	ips, nameList, err := resolver.LookupHostFull(host)
	if err != nil {
		return nil, "", err
	}

	// Handle ip
	if len(ips) == 0 {
		return nil, "", errors.New("No results")
	} else {
		resolved = ips[0]
	}

	if len(nameList) > 0 {
		name = nameList[0]
	}
	return resolved, name, nil
}
