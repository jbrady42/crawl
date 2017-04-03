package main

import (
  "fmt"
  "os"

  "github.com/codegangsta/cli"
  "github.com/jbrady42/crawl/util"
)

func filterMain() {
  inQ := util.NewStdinReader(0)
  outQ := make(chan string)

  go func() {
    filterQueryString(inQ, outQ)
    close(outQ)
  }()

  // Output
  for a := range outQ {
    fmt.Println(a)
  }
}

func filterQueryString(inQ, outQ chan string) {
  for a := range inQ {
    u := util.ParseUrl(a)
    if u != nil {
      // Remove query
      u.RawQuery = ""
      outQ <- u.String()
    }
    
  }
}

func main() {

  app := cli.NewApp()
  app.Name = "filter"
  app.Usage = ""
  app.Description = "A filter for crawl data"
  app.Version = "0.0.1"
  app.Commands = []cli.Command{
    {
      // Extract
      Name:    "no-query",
      Aliases: []string{"e"},
      Usage:   "Filter query strings",
      Action: func(c *cli.Context) {
        filterMain()
      },
    },
  }

  app.Run(os.Args)
}
