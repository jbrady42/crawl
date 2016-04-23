package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

func NewStdinReader(buffer int) chan string {
	out := make(chan string, buffer)
	// scan := bufio.NewScanner(os.Stdin)
	reader := bufio.NewReader(os.Stdin)
	go func() {
		line, err := reader.ReadString('\n')

		for err == nil {
			// line := scan.Text()
			line = strings.Trim(line, "\n \t\r")

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

func Conatins(list []string, item string) bool {
	for _, a := range list {
		if a == item {
			return true
		}
	}
	return false
}
