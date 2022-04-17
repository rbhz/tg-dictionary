package mymemory

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleResponse = `{
	"responseData": {
	  "translatedText": "Здравствуйте",
	  "match": 1
	},
	"quotaFinished": false,
	"mtLangSupported": null,
	"responseDetails": "",
	"responseStatus": 200,
	"responderId": "228",
	"exception_code": null,
	"matches": [
	  {
		"id": "589140219",
		"segment": "Hello",
		"translation": "Здравствуйте",
		"source": "en-GB",
		"target": "ru-RU",
		"quality": "74",
		"reference": null,
		"usage-count": 2,
		"subject": "All",
		"created-by": "MateCat",
		"last-updated-by": "MateCat",
		"create-date": "2021-11-05 13:50:59",
		"last-update-date": "2021-11-05 13:50:59",
		"match": 1
	  },
	  {
		"id": "596601254",
		"segment": "Hello",
		"translation": "Привет",
		"source": "en-GB",
		"target": "ru-RU",
		"quality": "74",
		"reference": null,
		"usage-count": 2,
		"subject": "All",
		"created-by": "MateCat",
		"last-updated-by": "MateCat",
		"create-date": "2022-01-27 17:07:12",
		"last-update-date": "2022-01-27 17:07:12",
		"match": 0.99
	  }
	]
}`

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func ptrStr(s string) *string {
	return &s
}

func TestTranslate(t *testing.T) {
	validUrl := "https://api.mymemory.translated.net/get?langpair=en%7Cru&q=hello"
	word := "hello"
	t.Run("success", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validUrl, req.URL.String())
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(exampleResponse)),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := MyMemoryClient{client: httpClient, context: context.TODO()}
		tranlation, err := client.Translate(word, "en", "ru")

		assert.NoError(t, err)
		expected := TranslationResponse{
			Result: TranslationResult{Text: "Здравствуйте", Match: 1},
			Matches: []TranslationMatch{
				{
					ID:          "589140219",
					Segment:     "Hello",
					Translation: "Здравствуйте",
					Source:      "en-GB",
					Target:      "ru-RU",
					Quality:     "74",
					Reference:   nil,
					UsageCount:  2,
					Subject:     "All",
					Match:       1,
				},
				{
					ID:          "596601254",
					Segment:     "Hello",
					Translation: "Привет",
					Source:      "en-GB",
					Target:      "ru-RU",
					Quality:     "74",
					Reference:   nil,
					UsageCount:  2,
					Subject:     "All",
					Match:       0.99,
				},
			},
			QuotaFinished:   false,
			ResponseDetails: "",
			ResponseStatus:  200,
			ResponderID:     "228",
			ExceptionCode:   nil,
		}
		assert.Equal(t, expected, tranlation)
	})
	t.Run("reqest error", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validUrl, req.URL.String())
				return &http.Response{}, http.ErrServerClosed
			}),
		}
		client := MyMemoryClient{client: httpClient, context: context.TODO()}
		tranlation, err := client.Translate(word, "en", "ru")
		assert.ErrorIs(t, err, http.ErrServerClosed)
		assert.Equal(t, TranslationResponse{}, tranlation)
	})
	t.Run("invalid response", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validUrl, req.URL.String())
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("Invalid JSON")),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := MyMemoryClient{client: httpClient, context: context.TODO()}
		tranlation, err := client.Translate(word, "en", "ru")
		assert.Error(t, err)
		assert.Equal(t, TranslationResponse{}, tranlation)
	})
	t.Run("error status", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validUrl, req.URL.String())
				return &http.Response{
					StatusCode: 400,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"status": "ERROR"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := MyMemoryClient{client: httpClient, context: context.TODO()}
		tranlation, err := client.Translate(word, "en", "ru")
		assert.Error(t, err)
		assert.Equal(t, TranslationResponse{}, tranlation)
	})
	t.Run("test same as input", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validUrl, req.URL.String())
				return &http.Response{
					StatusCode: 200,
					Body: ioutil.NopCloser(
						bytes.NewBufferString(`{"responseData": {"translatedText": "Hello", "match": 1}}`),
					),
					Header: make(http.Header),
				}, nil
			}),
		}
		client := MyMemoryClient{client: httpClient, context: context.TODO()}
		_, err := client.Translate(word, "en", "ru")
		assert.ErrorIs(t, err, ErrUnknown)

	})
}
