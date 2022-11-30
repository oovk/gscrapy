package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

// GET request from client to server with url and proxystring
func scrapeClientRequest(searchURL string, proxyString interface{}) (*http.Response, error) {
	baseClient := getScrapeClient(proxyString)
	req, _ := http.NewRequest("GET", searchURL, nil) //get request to the respective url
	req.Header.Set("User-Agent", randomUserAgent())

	res, err := baseClient.Do(req)
	if res.StatusCode != 200 {
		err := fmt.Errorf("scrapper received a non-200 status code suggesting a ban")
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return res, nil

}

// Parse the results and put it into document and then the struct we made, finding the needed information using html tags
func googleResultParsing(response *http.Response, rank int) ([]SearchResults, error) {
	doc, err := goquery.NewDocumentFromResponse(response) //respose received is converted to doc format so that the text can be modified to put into struct
	if err != nil {
		return nil, err
	}
	results := []SearchResults{}
	sel := doc.Find("div.g") //this will contain every request
	rank++
	//we are traversing through html document here, getting all the text between tags that we need
	for i := range sel.Nodes {
		item := sel.Eq(i)
		linkTag := item.Find("a") //html link tag <a>
		link, _ := linkTag.Attr("href")
		titleTag := item.Find("h3.r") //these tags are used by google
		descTag := item.Find("span.st")
		desc := descTag.Text()
		title := titleTag.Text()
		link = strings.Trim(link, " ")

		if link != "" && link != "#" && !strings.HasPrefix(link, "/") {
			result := SearchResults{
				rank,
				link,
				title,
				desc,
			}
			results = append(results, result)
			rank++
		}
	}
	return results, err
}

func main() {
	res, err := GoogleScrape("golang", "com", "en", nil, 1, 30, 10)
	if err == nil {
		for _, res := range res {
			fmt.Println(res)
		}
	}
}
