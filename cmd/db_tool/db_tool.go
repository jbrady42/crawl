package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"

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
	Url     string `gorm:"not null;unique_index"`
	Visited bool
	TLD     string `gorm:"index"`
	Queued  bool
	Success bool
	Message string
	IP      string
}

func newSite(urlS string) Site {
	return Site{Url: urlS, TLD: util.UrlTopHost(urlS)}
}

func connectDB() *gorm.DB {
	dbUrl := os.Getenv("DATABASE_URL")
	conStr, _ := pq.ParseURL(dbUrl)
	conStr += fmt.Sprintf(" sslmode=%v", "disable") //require
	// log.Println(conStr)
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
	// db = connectDB()
	// setupDB(db)

	log.Println("DB Complete")

	inQ := util.NewStdinReader()

	for line := range inQ {
		fun(line)
		// importLink(line)
		// importPage(line)
	}
}

func importLinksMain() {
	// db = connectDB()
	// setupDB(db)

	// log.Println("DB Complete")

	cache, _ := lru.New(500000)

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
	db.Where(newSite(link)).Attrs(Site{Visited: true}).FirstOrInit(&site)
	site.Success = success
	if !success {
		site.Message = page.Message
	}
	db.Save(&site)

	log.Println("Added page:", link)
}

func importResolve(line string) {
	page := data.ResolveResultFromLine(line)
	url := page.Url

	var site Site
	db.Where(newSite(url)).Find(&site)
	site.IP = page.IP.String()
	site.Message = page.Message
	db.Save(&site)

	// log.Println(site)

	log.Println("Added ip for:", url)
}

func importLink(link string, cache *lru.Cache) {
	key := md5.Sum([]byte(link))

	if !cache.Contains(key) {
		cache.Add(key, struct{}{})

		var site Site
		db.Where(newSite(link)).Attrs(Site{Visited: false}).FirstOrCreate(&site)
		if site.Visited {
			log.Println("Existing url:", link)
		} else {
			log.Println("Added url:", link)
			fmt.Println(link)
		}
	} else {
		log.Println("Existing cached:", link)
	}

	// db.Save(&site)

}

func getNextUrlList(limit int) {
	// db = connectDB()
	// setupDB(db)

	var res []Site
	db.Limit(limit).Order("random()").Where(map[string]interface{}{"visited": false}).Find(&res)
	// db.Find(&res)
	for _, a := range res {
		fmt.Println(a.Url)
	}
	log.Println("Done")
}

func main() {
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
				getNextUrlList(url_count)
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
	}

	app.Run(os.Args)
}
