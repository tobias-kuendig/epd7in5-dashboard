package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
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

var categoryIds = []int{
	266, // Programmieren
	16,  // Leben
	32,  // Menschen
	5,   // Liebe
	7,   // Glück
	264, // English
	260, // Investment
	62,  // Erfolg
	25,  // Geld
	14,  // Zeit
	306, // Arbeit
	45,  // Ziel
	23,  // Weisheit
	648, // Tier
	154, // Hoffnung
	37,  // Zukunft
	38,  // Tod
	39,  // Wahrheit
	160, // Erziehung
}

var languages = []string{
	"en",
	"de",
}

var errInvalidQuote = fmt.Errorf("invalid quote")

func fetchQuoteRetry(maxRetries int) (quote, error) {
	var q quote
	var err error
	for i := 0; i < maxRetries; i++ {
		q, err = fetchQuote()
		if err == nil {
			return q, nil
		}
		if errors.Is(err, errInvalidQuote) {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		return quote{}, err
	}
	return quote{}, fmt.Errorf("failed to fetch quote after %d retries: %w", maxRetries, err)
}

func fetchQuote() (quote, error) {
	categoryId := categoryIds[rand.Intn(len(categoryIds))]

	language := "en"
	if categoryId != 264 {
		language = languages[rand.Intn(len(languages))]
	}

	resp, err := http.Get(fmt.Sprintf(quoteEndpoint+"/v1/quote?language=%s&categoryId=%d", language, categoryId))
	if err != nil {
		return quote{}, fmt.Errorf("%w: %w", errInvalidQuote, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return quote{}, fmt.Errorf("invalid status code: %w: %w", errInvalidQuote, err)
	}

	var response quoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return quote{}, fmt.Errorf("decoing failed: %w: %w", errInvalidQuote, err)
	}

	return quote{
		Text:   response.Quote,
		Author: response.Author,
	}, nil
}
