package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
)

var db *gorm.DB

type Site struct {
	gorm.Model
	Url     string `gorm:"not null;unique"`
	Visited bool
}

func connectDB() *gorm.DB {
	dbUrl := os.Getenv("DATABASE_URL")
	conStr, _ := pq.ParseURL(dbUrl)
	conStr += fmt.Sprintf(" sslmode=%v", "disable") //require
	log.Println(conStr)
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
	db = connectDB()
	setupDB(db)

	fmt.Println("DB Complete")

	inQ := util.NewStdinReader()

	for line := range inQ {
		fun(line)
		// importLink(line)
		// importPage(line)
	}
}

func importPage(line string) {
	page := data.PageDataFromLine(line)
	link := page.Data.Url

	var site Site
	db.Where(Site{Url: link}).Assign(Site{Visited: true}).FirstOrCreate(&site)
	log.Println("Added page:", link)
}

func importLink(link string) {
	var site Site
	db.Where(Site{Url: link}).Attrs(Site{Visited: false}).FirstOrCreate(&site)
	if site.Visited {
		log.Println("Existing url:", link)
	} else {
		log.Println("Added url:", link)
	}

	// db.Save(&site)

}

func getNextUrlList(limit int) {
	db = connectDB()
	setupDB(db)

	var res []Site
	db.Limit(limit).Where(map[string]interface{}{"visited": false}).Find(&res)
	// db.Find(&res)
	for _, a := range res {
		fmt.Println(a.Url)
	}
}

func main() {

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
				importMain(importLink)
			},
		},
		{
			Name:    "next_urls",
			Aliases: []string{"u"},
			Usage:   "Import links",
			Action: func(c *cli.Context) {
				getNextUrlList(100)
			},
		},
	}

	app.Run(os.Args)
}
