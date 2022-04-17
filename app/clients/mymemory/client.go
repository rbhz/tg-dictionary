package mymemory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

var ErrUnknown = errors.New("failed to translate query")

// MyMemoryClient implements integration with mymemory translations API
// docs: https://mymemory.translated.net/doc/spec.php
type MyMemoryClient struct {
	apiToken *string
	client   *http.Client
	context  context.Context
}

func (c MyMemoryClient) Translate(q string, from string, to string) (TranslationResponse, error) {
	var result TranslationResponse
	req, err := http.NewRequest(
		http.MethodGet, "https://api.mymemory.translated.net/get", nil,
	)
	if c.context != nil {
		req = req.WithContext(c.context)
	}
	query := req.URL.Query()
	query.Add("q", q)
	query.Add("langpair", fmt.Sprintf("%s|%s", from, to))
	if c.apiToken != nil {
		query.Add("key", *c.apiToken)
	}
	req.URL.RawQuery = query.Encode()
	if err != nil {
		return result, fmt.Errorf("failed to create request: %w", err)
	}
	response, err := c.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("failed to create execute request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode != 200 {
		log.Error().
			Str("status", response.Status).
			Str("body", string(body)).
			Msg("unsuccessfull response from mymemory translated API")
		return result, fmt.Errorf("unsuccessfull API response %v", response.StatusCode)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if strings.ToUpper(result.Result.Text) == strings.ToUpper(q) {
		return result, ErrUnknown
	}
	return result, nil
}

func NewMymemoryClient(ctx context.Context, apiToken *string) MyMemoryClient {
	return MyMemoryClient{apiToken: apiToken, client: http.DefaultClient, context: ctx}
}
