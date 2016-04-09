package crawl

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/bogdanovich/dns_resolver"
	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
)

type ResolveWorker struct {
	resolver *dns_resolver.DnsResolver
}

func DefaultResolver() (resolver *dns_resolver.DnsResolver) {
	resolver = dns_resolver.New([]string{"208.67.222.222", "208.67.220.220", "8.8.8.8", "8.8.4.4"})
	resolver.RetryTimes = 3
	resolver.ReuseConnection = true
	return resolver
}

func NewResolver(servers []string) (resolver *dns_resolver.DnsResolver) {
	// Hand a copy because resolver mutates servers
	tmpServer := make([]string, len(servers))
	copy(tmpServer, servers)

	resolver = dns_resolver.New(tmpServer)
	resolver.ReuseConnection = true
	return resolver
}

func (t *Crawler) Resolve(inQ chan string, outQ chan *data.ResolveResult) {
	var wg sync.WaitGroup
	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)
		go func() {
			resolver := NewResolver(t.ResolveServers)
			worker := ResolveWorker{resolver}

			worker.resolveWorker(inQ, outQ)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (t *ResolveWorker) resolveWorker(inQ chan string, outQ chan *data.ResolveResult) {
	for urlStr := range inQ {
		url := util.ParseUrl(urlStr)

		var res *data.ResolveResult
		ip, err := resolv(t.resolver, url.Host)
		if err != nil {
			res = data.NewErrorResolveResult(urlStr, err)
			log.Println(err.Error(), urlStr)
		} else {
			res = data.NewResolveResult(urlStr, ip)
			log.Println("Resolved:", urlStr)
		}

		outQ <- res
	}
}

func resolv(resolver *dns_resolver.DnsResolver, host string) (resolved net.IP, err error) {
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
