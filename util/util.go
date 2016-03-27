package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

func NewStdinReader() chan string {
	out := make(chan string)
	scan := bufio.NewScanner(os.Stdin)
	go func() {
		for scan.Scan() {
			line := scan.Text()
			strings.Trim(line, "\n \t")
			if len(line) > 0 {
				out <- line
			}
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
