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
	servers      []string
}

type ResolveWorker struct {
	resolver     *dns_resolver.DnsResolver
	crawlResolve *Resolver
}

func New() *Resolver {
	return NewWithServers([]string{})
}

func NewWithServers(servers []string) *Resolver {
	cache, _ := lru.New(1000000)
	if len(servers) == 0 {
		servers = []string{"208.67.222.222", "208.67.220.220", "8.8.8.8", "8.8.4.4"}
	}
	worker := newResolver(servers)
	return &Resolver{worker, cache, servers}
}

func (t *Resolver) NewWorker() *ResolveWorker {
	res := newResolver(t.servers)
	return &ResolveWorker{res, t}
}

func newResolver(servers []string) *dns_resolver.DnsResolver {
	// Hand a copy because resolver mutates servers
	tmpServer := make([]string, len(servers))
	copy(tmpServer, servers)

	resolver := dns_resolver.New(tmpServer)
	resolver.ReuseConnection = true

	return resolver
}

func (t *ResolveWorker) Resolve(host string) (resolved net.IP, cname string, err error) {
	return resolveWithCache(host, t.resolver, t.crawlResolve.resolveCache)
}

// Resolve with the crawlers cache
func (t *Resolver) Resolve(host string) (resolved net.IP, cname string, err error) {
	return resolveWithCache(host, t.resolver, t.resolveCache)
}

func resolveWithCache(host string, resolver *dns_resolver.DnsResolver, cache *lru.Cache) (resolved net.IP, cname string, err error) {
	// Hit cache first
	tmp, found := cache.Get(host)
	if !found {

		// Do resolve
		resolved, cname, err = resolveWithCname(resolver, host)

		if err != nil {
			return nil, "", err
		} else {
			// resolved = newIP
			item := cacheItem{host, resolved, cname}
			cache.Add(host, item)
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
