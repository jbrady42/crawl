package crawl

import (
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/jbrady42/crawl/util"
)

func ExtractMain(inQ chan string, outQ chan *PageResult) {
	for s := range inQ {
		// Parse page data
		page := PageDataFromLine(s)
		// if len(page.Url) < 4 {
		// 	log.Printf("Error, not extracting. Bad url in line %s\n", line)
		// 	continue
		// }
		links := ExtractLinks(page.Data)

		// Basic filtering
		links = filterDupLinks(links)
		page.Links = links

		outQ <- page
	}
}

// TODO make sure urls are normalized
func ExtractLinks(page *PageData) (res []*url.URL) {
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

			res = append(res, newU)
		}
	}

	return res
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
