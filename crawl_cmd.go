package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/jbrady42/crawl/core"
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
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			// Extract
			Name:    "extract",
			Aliases: []string{"e"},
			Usage:   "Extract urls",
			Action: func(c *cli.Context) {
				extractMain()
			},
		},
		{
			// Download
			Name:    "download",
			Aliases: []string{"d"},
			Usage:   "Download urls",
			Action: func(c *cli.Context) {
				downloadMain()
			},
		},
	}

	app.Run(os.Args)
	// printMain()
}
