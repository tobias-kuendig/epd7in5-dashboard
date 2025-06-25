package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var quoteEndpoint = "https://api.zitat-service.de"

type quoteResponse struct {
	Quote  string `json:"quote"`
	Author string `json:"authorName"`
}

type quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

func fetchQuote() quote {
	resp, err := http.Get(quoteEndpoint + "/v1/quote?language=de")
	if err != nil {
		return quote{Text: fmt.Sprintf("failed to load quote: %s", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return quote{Text: fmt.Sprintf("failed to load quote: %s", resp.Status)}
	}

	var response quoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return quote{Text: fmt.Sprintf("failed to decode quote: %s", err)}
	}

	return quote{
		Text:   response.Quote,
		Author: response.Author,
	}
}
