package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/jbrady42/crawl/core"
	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
)

func downloadMain() {
	inQ := util.NewStdinReader()
	outQ := make(chan *data.PageResult)
	// opts := crawl.CrawlOpts{workers, 1.0}

	// Setup crawler
	crawl := crawl.NewCrawler(workers, false)
	crawl.MaxPageBytes = sizeLimit

	go func() {
		crawl.Download(inQ, outQ)
		close(outQ)
	}()

	//Output
	for a := range outQ {
		fmt.Println(util.ToJSONStr(a))
	}
}

func resolveMain() {
	inQ := util.NewStdinReader()
	outQ := make(chan *data.ResolveResult)

	servers := strings.Split(resolverStr, ",")
	log.Println("Resolvers:", servers)
	// Setup crawler
	crawl := crawl.NewCrawler(workers, false)
	crawl.ResolveServers = servers

	go func() {
		crawl.Resolve(inQ, outQ)
		close(outQ)
	}()

	//Output
	for a := range outQ {
		fmt.Println(util.ToJSONStr(a))
	}
}

func extractMain() {
	inQ := util.NewStdinReader()
	outQ := make(chan *data.PageResult)

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

var workers int
var sizeLimit int
var resolverStr string

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
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "workers",
					Value:       1,
					Usage:       "Number of download workers",
					Destination: &workers,
				},
				cli.IntFlag{
					Name:        "max-bytes",
					Value:       0.0,
					Usage:       "Limit download page size rate. 0 for none.",
					Destination: &sizeLimit,
				},
			},
			Action: func(c *cli.Context) {
				downloadMain()
			},
		},
		{
			// Resolve
			Name:    "resolve",
			Aliases: []string{"r"},
			Usage:   "Resolve urls",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "workers",
					Value:       1,
					Usage:       "Number of resolve workers",
					Destination: &workers,
				},
				cli.StringFlag{
					Name:        "servers",
					Value:       "8.8.8.8",
					Usage:       "Comma separated list of resolve servers",
					Destination: &resolverStr,
				},
			},
			Action: func(c *cli.Context) {
				resolveMain()
			},
		},
	}

	app.Run(os.Args)
	// printMain()
}
