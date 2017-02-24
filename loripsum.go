package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var loripsumSizes = []string{"small", "medium", "large"}
var loripsumURL = "http://loripsum.net/api/%d/%s/plaintext"

// Loripsum fetches random text from the loripsum.net API
func Loripsum(paragraphs int, size int) string {
	var body string
	var pSize = loripsumSizes[size]
	url := fmt.Sprintf(loripsumURL, paragraphs, pSize)
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	body = fmt.Sprintf("%s", bytes)
	return body
}
