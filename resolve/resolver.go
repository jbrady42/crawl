package resolve

import (
	"errors"
	"net"

	"github.com/bogdanovich/dns_resolver"
	"github.com/hashicorp/golang-lru"
)

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
func (t *Resolver) Resolve(host string) (resolved net.IP, err error) {
	// Hit cache first
	ip, found := t.resolveCache.Get(host)
	if !found {
		newIP, err := resolve(t.resolver, host)
		if err != nil {
			return nil, err
		} else {
			resolved = newIP
			t.resolveCache.Add(host, newIP)
		}
	} else {
		// log.Println("Resolve cached: ", host)
		resolved = ip.(net.IP)
	}
	return resolved, nil
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
