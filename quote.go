package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var quoteEndpoint = "https://zenquotes.io/api"

type quoteResponse struct {
	Quote  string `json:"q"`
	Author string `json:"a"`
}

type quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

func fetchQuote() quote {
	resp, err := http.Get(quoteEndpoint + "/random")
	if err != nil {
		return quote{Text: fmt.Sprintf("failed to load quote: %s", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return quote{Text: fmt.Sprintf("failed to load quote: %s", resp.Status)}
	}

	var quotes []quoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&quotes); err != nil {
		return quote{Text: fmt.Sprintf("failed to decode quote: %s", err)}
	}

	if len(quotes) == 0 {
		return quote{Text: "no quote found"}
	}

	return quote{
		Text:   quotes[0].Quote,
		Author: quotes[0].Author,
	}
}
