package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/PuerkitoBio/purell"
)

func NewStdinReader() chan string {
	out := make(chan string)
	// scan := bufio.NewScanner(os.Stdin)
	reader := bufio.NewReader(os.Stdin)
	go func() {
		line, err := reader.ReadString('\n')

		for err == nil {
			// line := scan.Text()
			line = strings.Trim(line, "\r\n \t")

			if len(line) > 0 {
				out <- line
			}

			// Next
			line, err = reader.ReadString('\n')
		}
		close(out)
	}()
	return out
}

func StdoutWriter(oq chan interface{}) {
	for a := range oq {
		fmt.Println(a)
	}
}

func ToJSON(data interface{}) []byte {
	json_data, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshaling response data")
		return nil
	}
	return json_data
}

func ToJSONStr(data interface{}) string {
	return string(ToJSON(data))
}

func ParseUrl(str string) (res *url.URL) {
	// Normalize
	normalized, err := purell.NormalizeURLString(str,
		purell.FlagsUsuallySafeGreedy|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveFragment)

	if err != nil {
		log.Printf("Error normalizing url %v\n", str)
		return nil
	}
	// Then parse
	res, err = url.Parse(normalized)
	if err != nil {
		log.Printf("Error parsing url %v\n", str)
		return nil
	}
	return res
}

func ParseUrlEscaped(str string) *url.URL {
	inUrl := ParseUrl(str)
	if inUrl == nil {
		return nil
	}
	// Better handle query strings
	q := inUrl.Query()
	inUrl.RawQuery = q.Encode()

	return inUrl
}

func EscapedUrlStr(str string) string {
	return ParseUrlEscaped(str).String()
}

func SiteRoot(info *url.URL) *url.URL {
	ret := &url.URL{
		Host:   info.Host,
		Scheme: info.Scheme,
		User:   info.User,
		//Path:   "/",
	}
	return ret
}

func Conatins(list []string, item string) bool {
	for _, a := range list {
		if a == item {
			return true
		}
	}
	return false
}

func UrlTopHost(urlStr string) string {
	uRl := ParseUrlEscaped(urlStr)
	host := ""
	if uRl != nil {
		host, _ = publicsuffix.EffectiveTLDPlusOne(uRl.Host)
	}

	if ip := net.ParseIP(uRl.Host); ip != nil {
		//log.Println("Matched IP as host")
		host = uRl.Host
	}
	return host
}
