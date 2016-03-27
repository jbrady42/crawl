package main

import (
	"fmt"

	"github.com/jbrady42/crawl/crawl"
	"github.com/jbrady42/crawl/util"
)

func main() {
	inQ := util.NewStdinReader()
	outQ := make(chan *crawl.PageData)
	opts := crawl.CrawlOpts{1, 1.0}

	go func() {
		crawl.DownloadMain(inQ, outQ, opts)
		close(outQ)
	}()

	//Output
	for a := range outQ {
		fmt.Println(util.ToJSONStr(a))
	}
}
