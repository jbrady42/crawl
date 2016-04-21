package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/hashicorp/golang-lru"
	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
)

var db *gorm.DB

type Site struct {
	gorm.Model
	Url      string `gorm:"not null;unique_index"`
	Visited  bool
	TLD      string `gorm:"index"`
	QueuedAt *time.Time
	Success  bool
	Message  string
	IP       string
	Cname    string
}

func newSite(urlS string) Site {
	return Site{Url: urlS, TLD: util.UrlTopHost(urlS)}
}

func connectDB() *gorm.DB {
	dbUrl := os.Getenv("DATABASE_URL")
	conStr, _ := pq.ParseURL(dbUrl)
	conStr += fmt.Sprintf(" sslmode=%v", "disable") //require

	db, err := gorm.Open("postgres", conStr)
	if err != nil {
		log.Fatal("Can't connect to db")
	}
	return db
}

func withDB(fun func(*gorm.DB)) {
	db := connectDB()
	fun(db)
	db.Close()
}

// Migrate and seed
func setupDB(db *gorm.DB) {
	db.LogMode(false)

	// log.Println("Migrating")
	db.AutoMigrate(&Site{})

	// seedDB(db)
}

func seedDB(db *gorm.DB) {
	var count int
	db.Model(&Site{}).Count(&count)
	if count == 0 {
		log.Println("Inserting base record")

		db.Create(&Site{Url: "http://www.cnn.com", Visited: true})
	}
}

func importMain(fun func(string)) {
	inQ := util.NewStdinReader()

	for line := range inQ {
		fun(line)
	}
}

func importLinksMain() {
	cache, _ := lru.New(1000000)

	inQ := util.NewStdinReader()

	for line := range inQ {
		importLink(line, cache)
	}
}

func importPage(line string) {
	page := data.PageDataFromLine(line)
	link := page.Data.Url
	success := page.Success

	var site Site
	db.Where(newSite(link)).FirstOrInit(&site)
	site.Success = success
	if !success {
		site.Message = page.Message
	}

	// Clear queued
	site.QueuedAt = nil
	site.Visited = true
	db.Save(&site)

	log.Println("Added page:", link)
}

func importResolve(line string) {
	page := data.ResolveResultFromLine(line)
	url := page.Url

	var site Site
	db.Where(newSite(url)).Find(&site)
	site.IP = page.IP.String()
	site.Cname = page.Cname
	site.Message = page.Message
	db.Save(&site)

	log.Println("Added ip for:", url)
}

func importQueued() {
	inQ := util.NewStdinReader()

	// PG max params 65535
	batchSize := 1000
	count := 0

	var ids []int
	var buff []string

	for line := range inQ {
		newLine := strings.Trim(line, "\n\r\t")
		parts := strings.Split(newLine, "\t")
		id, _ := strconv.Atoi(parts[0])
		ids = append(ids, id)

		if len(parts) < 3 {
			log.Println(parts)
		}
		item := strings.Join(parts[1:3], "\t")
		buff = append(buff, item)

		if len(ids) > batchSize {
			log.Println("Saving batch", count)
			db.Table("sites").Where("id IN (?)", ids).Updates(map[string]interface{}{"queued_at": &currentTime})
			log.Println("Writing	 batch", count)
			printList(buff)

			ids = nil
			ids = []int{}
			buff = nil
			buff = []string{}
			count += 1
		}
	}

	if len(ids) > 0 {
		log.Println("Saving last batch", count)
		db.Table("sites").Where("id IN (?)", ids).Updates(map[string]interface{}{"queued_at": &currentTime})
		log.Println("Writing	 batch", count)
		printList(buff)
	}

}

func printList(list []string) {
	for _, s := range list {
		fmt.Println(s)
	}
}

func importLink(link string, cache *lru.Cache) {
	key := md5.Sum([]byte(link))

	if !cache.Contains(key) {
		cache.Add(key, struct{}{})

		var site Site
		newUrl := db.Where(newSite(link)).Find(&site).RecordNotFound()
		if newUrl {
			site := newSite(link)

			db.Create(&site)
			log.Println("Added url:", link)
			fmt.Println(link)

		} else {
			log.Println("Existing url:", link)
		}
	} else {
		log.Println("Existing cached:", link)
	}
}

func nextUrlList(limit int) {
	var res []Site
	db.Limit(limit).Order("random()").Where(map[string]interface{}{"visited": false}).Find(&res)
	// db.Find(&res)
	for _, a := range res {
		fmt.Println(a.Url)
	}
	log.Println("Done")
}

var currentTime time.Time

func main() {
	currentTime = time.Now()

	db = connectDB()
	setupDB(db)

	var url_count int
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			// Extract
			Name:    "pages",
			Aliases: []string{"p"},
			Usage:   "Import pages",
			Action: func(c *cli.Context) {
				importMain(importPage)
			},
		},
		{
			// Download
			Name:    "links",
			Aliases: []string{"l"},
			Usage:   "Import links",
			Action: func(c *cli.Context) {
				importLinksMain()
			},
		},
		{
			Name:    "next_urls",
			Aliases: []string{"n"},
			Usage:   "Import links",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "limit",
					Value:       10,
					Usage:       "Number of urls",
					Destination: &url_count,
				},
			},
			Action: func(c *cli.Context) {
				nextUrlList(url_count)
			},
		},
		{
			// Resolve
			Name:    "resolve",
			Aliases: []string{"r"},
			Usage:   "Import ips",
			Action: func(c *cli.Context) {
				importMain(importResolve)
			},
		},
		{
			// Queue
			Name:    "queue",
			Aliases: []string{"r"},
			Usage:   "Import queued",
			Action: func(c *cli.Context) {
				importQueued()
			},
		},
	}

	app.Run(os.Args)
}
