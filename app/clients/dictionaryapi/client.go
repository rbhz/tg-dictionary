package dictionaryapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
)

var ErrNotFound = errors.New("word not found")

// DictionaryAPIClient implements integration with DictionaryAPI
// docs: https://dictionaryapi.dev/
type DictionaryAPIClient struct {
	client  *http.Client
	context context.Context
}

func (c *DictionaryAPIClient) Get(word string) (items []WordResponse, err error) {
	req, err := http.NewRequest(
		http.MethodGet, "https://api.dictionaryapi.dev/api/v2/entries/en/"+word, nil,
	)
	if err != nil {
		return items, fmt.Errorf("create request: %w", err)
	}
	if c.context != nil {
		req = req.WithContext(c.context)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return items, fmt.Errorf("fetch dictionaryapi.dev: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return items, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return nil, ErrNotFound
		}
		log.Error().
			Str("status", resp.Status).
			Str("body", string(body)).
			Msg("unsuccessfull response from dictionaryapi")
		return items, fmt.Errorf("unsuccessfull API response %v", resp.StatusCode)
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return items, fmt.Errorf("unmarshal response: %w", err)
	}
	return items, nil
}

// NewDictionaryAPIClient creates DictionaryAPIClient with default HTTP client
func NewDictionaryAPIClient(ctx context.Context) DictionaryAPIClient {
	return DictionaryAPIClient{client: http.DefaultClient, context: ctx}
}
