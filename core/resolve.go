package core

import (
	"log"
	"sync"
	"time"

	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
)

func (t *Crawler) ResolveWorker(inQ <-chan string, outQ chan<- *data.ResolveResult) {
	var wg sync.WaitGroup
	wg.Add(t.WorkerCount)
	for i := 0; i < t.WorkerCount; i++ {
		time.Sleep(25 * time.Millisecond)

		go func(i int) {
			defer wg.Done()
			t.resolveWorker(inQ, outQ)
			log.Println("Closing worker ", i)
		}(i)
	}
	log.Println("Waiting on workers")
	wg.Wait()
}

func (t *Crawler) resolveWorker(inQ <-chan string, outQ chan<- *data.ResolveResult) {
	resolveWorker := t.Resolver.NewWorker()

	for urlStr := range inQ {
		url := util.ParseUrl(urlStr)
		host := url.Host

		var res *data.ResolveResult

		resolved, cname, err := resolveWorker.Resolve(host)
		if err != nil {
			res = data.NewErrorResolveResult(urlStr, err)
			log.Println(err.Error(), urlStr)
		} else {
			res = data.NewResolveResult(urlStr, resolved, cname)
			log.Println("Resolved:", urlStr)
		}
		outQ <- res
	}
}
