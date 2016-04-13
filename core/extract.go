package core

import (
	"log"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/PuerkitoBio/goquery"

	"github.com/jbrady42/crawl/data"
	"github.com/jbrady42/crawl/util"
)

func ExtractMain(inQ chan string, outQ chan *data.PageResult) {
	for s := range inQ {
		// Parse page data
		page := data.PageDataFromLine(s)
		// if len(page.Url) < 4 {
		// 	log.Printf("Error, not extracting. Bad url in line %s\n", line)
		// 	continue
		// }
		links := ExtractLinks(page.Data)

		// Basic filtering
		links = filterDupLinks(links)
		links = filterRegLinks(links)
		page.Links = links

		outQ <- page
	}
}

// TODO make sure urls are normalized
func ExtractLinks(page *data.PageData) (res []*url.URL) {
	pageReader := strings.NewReader(page.Body)
	//defer pageReader.Close()

	doc, err := goquery.NewDocumentFromReader(pageReader)
	if err != nil {
		log.Println("Error creating goquery doc")
		log.Println(err)
		return nil
	}

	// Set doc url
	curUrl := util.ParseUrlEscaped(page.Url)
	if curUrl == nil {
		log.Println("Error parsing doc url")
		return nil
	}

	doc.Url = curUrl

	urls := doc.Find("a[href]").Map(func(i int, s *goquery.Selection) string {
		val, _ := s.Attr("href")
		return val
	})

	// First push visited
	// uInfo := NewUrlInfo(curUrl, true)
	// res = append(res, uInfo)

	for _, item := range urls {
		parsedUrl := util.ParseUrlEscaped(item)
		if parsedUrl != nil {
			newU := doc.Url.ResolveReference(parsedUrl)

			// uInfo = NewUrlInfo(newU, false)
			// res = append(res, uInfo)

			// Transform urls
			// newU = opts.Extender.TransformUrl(newU)
			// newU = transformUrl(newU)

			res = append(res, newU)
		}
	}

	return res
}

func transformUrl(info *url.URL) *url.URL {
	return util.SiteRoot(info)
}

func filterDupLinks(links []*url.URL) []*url.URL {
	var tmp []*url.URL
	startLen := len(links)
	mapSet := make(map[string]struct{})

	for _, link := range links {
		urlS := link.String() //SuperTrim(urlIn.Url.String())

		// Add if its not there already
		if _, ok := mapSet[urlS]; !ok {
			tmp = append(tmp, link)
			mapSet[urlS] = struct{}{}
		} else {
			//log.Printf("filterDupLinks: filtering duplicate url %s\n", urlS)
		}
	}
	endLen := len(tmp)
	log.Printf("filterDupLinks: removed %d duplicates\n", startLen-endLen)
	return tmp
}

func filterRegLinks(links []*url.URL) []*url.URL {
	var filterTmp []*url.URL
	for _, link := range links {
		urlStr := link.String()
		if FilterUrl(link) {
			log.Printf("filterPageLinksWorker: filtering url %s\n", urlStr)
		} else {
			filterTmp = append(filterTmp, link)
		}
	}
	return filterTmp
}

// Probably should go elsewhere
func FilterUrl(inUrl *url.URL) bool {
	// Check if allowed scheme
	allowedSchemes := []string{"http", "https"}

	if !util.Conatins(allowedSchemes, inUrl.Scheme) {
		log.Printf("Filtering scheme %v\n", inUrl.Scheme)
		return true
	}

	// Host / Domain checks

	host, _ := publicsuffix.EffectiveTLDPlusOne(inUrl.Host)

	// Check for no domain
	if host == "" {
		log.Println("Filtering empty host")
		return true
	}

	// Check for allowed domains
	/*
		_, icann := publicsuffix.PublicSuffix(host)
		if !icann {
			log.Println("filtering bad host")
			return true
		}
	*/

	// Check port
	_, port, err := net.SplitHostPort(inUrl.Host)
	if err != nil {
		//log.Println(err)
	} else if port != "80" || port != "443" {
		log.Println("Filtering port ", port)
		return true
	}

	return false
}
