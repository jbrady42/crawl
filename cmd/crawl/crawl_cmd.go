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
	outQ := make(chan *data.PageResult, workers)

	// Setup crawler
	servers := []string{} // Use default servers

	crawl := core.NewCrawler(workers, groupHost, servers)
	crawl.MaxPageBytes = sizeLimit
	crawl.IgnoreRobots = ignoreRobot

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
	outQ := make(chan *data.ResolveResult, workers)

	servers := strings.Split(resolverStr, ",")
	log.Println("Resolvers:", servers)
	// Setup crawler
	crawl := core.NewCrawler(workers, false, servers)

	go func() {
		crawl.ResolveWorker(inQ, outQ)
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
		core.ExtractMain(inQ, outQ)
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
var groupHost bool
var ignoreRobot bool

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
				cli.BoolFlag{
					Name:        "host",
					Usage:       "Urls are input in groups by host",
					Destination: &groupHost,
				},
				cli.BoolFlag{
					Name:        "bad-robot",
					Usage:       "Disable robots.txt checking",
					Destination: &ignoreRobot,
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
