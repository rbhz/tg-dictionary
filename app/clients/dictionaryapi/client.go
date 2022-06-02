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

// ErrNotFound is returned when word is not found
var ErrNotFound = errors.New("word not found")

// Client implements integration with DictionaryAPI
// docs: https://dictionaryapi.dev/
type Client struct {
	client  *http.Client
	context context.Context
}

// Get returns dictionary item by word
func (c *Client) Get(word string) (items []WordResponse, err error) {
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

// NewClient creates Client with default HTTP client
func NewClient(ctx context.Context) Client {
	return Client{client: http.DefaultClient, context: ctx}
}
