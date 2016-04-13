package util

import (
	"log"
	"net"
	"net/url"

	"golang.org/x/net/publicsuffix"

	"github.com/PuerkitoBio/purell"
)

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
