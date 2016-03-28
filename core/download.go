package crawl

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/jbrady42/crawl/data"
)

func DownloadMain(inQ chan string, outQ chan *data.PageResult, opts CrawlOpts) {
	for s := range inQ {
		page := downloadUrl(s)
		outQ <- page
	}
}

func downloadUrl(url string) (page *data.PageResult) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading %s : %s\n", url, err)
		return data.NewFailedResult(url, err.Error())
	}

	var body []byte
	var reader io.Reader
	reader = resp.Body
	body, err = ioutil.ReadAll(reader)
	if err != nil {
		log.Println("Error reading response body")
	}
	// Close con
	resp.Body.Close()

	pd := data.NewPageData(url, resp, body)
	return pd
}
