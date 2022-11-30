package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var googleDomains = map[string]string{ //different country specific domains
	"com": "http://www.google.com/search?q=",
}

// structured search results, we can store it to db now
type SearchResults struct {
	ResultRank  int
	ResultURL   string
	ResultTitle string
	ResultDesc  string
}

// different types of browsers
var userAgents = []string{}

// choose random useragent
func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents) //returns the random number from 1 to len(userAgent)
	return userAgents[randNum]
}

// building google search queries
func buildGoogleUrls(searchTerm string, countryCode string, languageCode string, pages int, count int) ([]string, error) {
	toScrape := []string{}
	searchTerm = strings.Trim(searchTerm, " ") //removed spaces from searchterm
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)
	if googleBase, found := googleDomains[countryCode]; found {
		for i := 0; i < pages; i++ {
			start := i * count
			scrapeURL := fmt.Sprintf("%s%s&num=%d&hl=%s&start=%d&filter=0", googleBase, searchTerm, count, languageCode, start) //formatting the string for google querying
			toScrape = append(toScrape, scrapeURL)
		}
	} else {
		err := fmt.Errorf("country (%s) is currently not supported", countryCode)
		return nil, err
	}
	return toScrape, nil
}

func getScrapeClient(proxyString interface{}) *http.Client {
	switch v := proxyString.(type) {

	case string: //when we have proxy set, proxy is when we want to scrape anonymously
		proxyUrl, _ := url.Parse(v)
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

	default:
		return &http.Client{}

	}
}

// scrape the links corresponding to search term we are searching for
func GoogleScrape(searchTerm string, countryCode string, languageCode string, proxyString interface{}, pages int, count int, backoff int) ([]SearchResults, error) {
	results := []SearchResults{}
	resultCounter := 0
	googlePages, err := buildGoogleUrls(searchTerm, countryCode, languageCode, pages, count)
	if err != nil {
		return nil, err
	}
	for _, page := range googlePages {
		res, err := scrapeClientRequest(page, proxyString)
		if err != nil {
			return nil, err
		}
		data, err := googleResultParsing(res, resultCounter)
		if err != nil {
			return nil, err
		}
		resultCounter += len(data)
		for _, result := range data {
			results = append(results, result)
		}
		time.Sleep(time.Duration(backoff) * time.Second)
	}
	return results, nil
}

func main() {
	res, err := GoogleScrape("golang", "com", "en", nil, 1, 30, 10)
	if err == nil {
		for _, res := range res {
			fmt.Println(res)
		}
	}
}
