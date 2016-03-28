package main

import (
	"fmt"

	"github.com/jbrady42/crawl/crawl"
	"github.com/jbrady42/crawl/util"
)

func downloadMain() {
	inQ := util.NewStdinReader()
	outQ := make(chan *crawl.PageResult)
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

func extractMain() {
	inQ := util.NewStdinReader()
	outQ := make(chan *crawl.PageResult)

	go func() {
		crawl.ExtractMain(inQ, outQ)
		close(outQ)
	}()

	// Output
	for a := range outQ {
		links := a.Links
		for _, l := range links {
			fmt.Println(l.String())
		}
	}
}

func printMain() {
	inQ := util.NewStdinReader()
	for a := range inQ {
		fmt.Println(a)
	}
}

func main() {
	// printMain()
	// downloadMain()
	extractMain()
}
