package data

import (
	"encoding/json"
	"log"
	"net"
)

type ResolveResult struct {
	Url     string
	IP      net.IP
	Message string
}

func NewErrorResolveResult(host string, err error) *ResolveResult {
	return &ResolveResult{host, nil, err.Error()}
}

func NewResolveResult(host string, ip net.IP) *ResolveResult {
	return &ResolveResult{host, ip, ""}
}

func ResolveResultFromLine(line string) *ResolveResult {
	var res ResolveResult

	data := []byte(line)
	err := json.Unmarshal(data, &res)
	if err != nil {
		log.Println("Error marshaling line")
		log.Println(err)
	}
	return &res
}
