package crawl

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func DownloadMain(inQ chan string, outQ chan *PageResult, opts CrawlOpts) {
	for s := range inQ {
		page := downloadUrl(s)
		outQ <- page
	}
}

func downloadUrl(url string) (page *PageResult) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading %s : %s\n", url, err)
		return NewFailedResult(url, err.Error())
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

	pd := NewPageData(url, resp, body)
	return pd
}
