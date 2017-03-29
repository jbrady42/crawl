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
	inQ := util.NewStdinReader(workers)
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
	inQ := util.NewStdinReader(0)
	outQ := make(chan *data.ResolveResult, workers)

	var servers []string
	if resolverStr != "" {
		servers = strings.Split(resolverStr, ",")
	} else {
		servers = []string{"208.67.222.222", "208.67.220.220", "8.8.8.8", "8.8.4.4"}
	}
	log.Println("Resolvers:", servers)
	// Setup crawler
	crawl := core.NewCrawler(workers, false, servers)

	// Set cache size
	if cacheSize > 0 {
		crawl.Resolver.ResetCache(cacheSize)
	}

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
	inQ := util.NewStdinReader(0)
	outQ := make(chan *data.PageResult)

	go func() {
		core.ExtractMain(inQ, outQ, siteRoot)
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
	inQ := util.NewStdinReader(0)
	for a := range inQ {
		fmt.Println(a)
	}
}

var workers int
var sizeLimit int
var resolverStr string
var groupHost bool
var ignoreRobot bool
var siteRoot bool
var cacheSize int

func main() {

	app := cli.NewApp()
	app.Name = "crawl"
	app.Usage = ""
	app.Description = "A crawler of the Unix philosophy"
	app.Version = "0.0.5"
	app.Commands = []cli.Command{
		{
			// Extract
			Name:    "extract",
			Aliases: []string{"e"},
			Usage:   "Extract urls",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "root",
					Usage:       "Only extract site roots",
					Destination: &siteRoot,
				},
			},
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
					Value:       "",
					Usage:       "Comma separated list of resolve servers",
					Destination: &resolverStr,
				},
				cli.IntFlag{
					Name:        "cache",
					Value:       0,
					Usage:       "Max cache items",
					Destination: &cacheSize,
				},
			},
			Action: func(c *cli.Context) {
				resolveMain()
			},
		},
	}

	app.Run(os.Args)
	log.Println("Exiting crawl")
}
