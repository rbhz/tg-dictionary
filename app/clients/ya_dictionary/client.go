package ya_dictionary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
)

var ErrUnknown = errors.New("failed to translate text")

// MyMemoryClient implements integration with yandex dictionary API
// docs: https://yandex.com/dev/dictionary/doc/dg/concepts/api-overview.html
type YandexDictionaryClient struct {
	apiToken string
	client   *http.Client
	context  context.Context
}

func (c YandexDictionaryClient) Translate(text string, from string, to string) (TranslationResponse, error) {
	var result TranslationResponse
	req, err := http.NewRequest(
		http.MethodGet, "https://dictionary.yandex.net/api/v1/dicservice.json/lookup", nil,
	)
	if c.context != nil {
		req = req.WithContext(c.context)
	}
	query := req.URL.Query()
	query.Add("key", c.apiToken)
	query.Add("lang", fmt.Sprintf("%s-%s", from, to))
	query.Add("text", text)
	req.URL.RawQuery = query.Encode()
	if err != nil {
		return result, fmt.Errorf("create request: %w", err)
	}
	response, err := c.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("create execute request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("read response body: %w", err)
	}

	if response.StatusCode != 200 {
		log.Error().
			Str("status", response.Status).
			Str("body", string(body)).
			Msg("unsuccessfull response from yandex dictionary API")
		return result, fmt.Errorf("unsuccessfull API response %v", response.StatusCode)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Definitions) == 0 {
		return result, ErrUnknown
	}
	return result, nil
}

func NewYaDictionaryClient(ctx context.Context, apiToken string) YandexDictionaryClient {
	return YandexDictionaryClient{apiToken: apiToken, client: http.DefaultClient, context: ctx}
}
