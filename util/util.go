package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"sort"
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
			line = strings.Trim(line, "\n \t")

			// fmt.Println(len(line))
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
	sort.Strings(list)
	i := sort.SearchStrings(list, item)
	cont := i < len(list) && list[i] == item
	return cont
}

var ipReg = regexp.MustCompile("^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$")

func UrlTopHost(urlStr string) string {
	uRl := ParseUrlEscaped(urlStr)
	host := ""
	if uRl != nil {
		host, _ = publicsuffix.EffectiveTLDPlusOne(uRl.Host)
	}
	isIp := ipReg.MatchString(uRl.Host)
	if isIp {
		//log.Println("Matched IP as host")
		host = uRl.Host
	}
	return host
}
