package ya_dictionary

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleResponse = `{
	"head": {},
	"def": [
		{
			"text": "time",
			"pos": "noun",
			"tr": [
				{
					"text": "время",
					"pos": "существительное",
					"syn": [
						{"text": "раз"},
						{"text": "тайм"}
					],
					"mean": [
						{"text": "timing"},
						{"text": "fold"},
						{"text": "half"}
					],
					"ex": [
						{
							"text": "prehistoric time",
							"tr": [
								{"text": "доисторическое время"}
							]
						},
						{
							"text": "hundredth time",
							"tr": [
								{"text": "сотый раз"}
							]
						},
						{
							"text": "time-slot",
							"tr": [
								{"text": "тайм-слот"}
							]
						}
					]
				}
			]
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
	validUrl := "https://dictionary.yandex.net/api/v1/dicservice.json/lookup?key=test&lang=en-ru&text=time"
	APItoken := "test"
	word := "time"
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
		client := YandexDictionaryClient{client: httpClient, apiToken: APItoken, context: context.TODO()}
		tranlation, err := client.Translate(word, "en", "ru")

		assert.NoError(t, err)
		expected := TranslationResponse{
			Definitions: []Definition{
				{
					Text:         "time",
					PartOfSpeech: "noun",
					Translations: []Translation{
						{
							Text:         "время",
							PartOfSpeech: "существительное",
							Examples: []Example{
								{
									Text: "prehistoric time",
									Translations: []textItem{
										{Text: "доисторическое время"},
									},
								},
								{
									Text: "hundredth time",
									Translations: []textItem{
										{Text: "сотый раз"},
									},
								},
								{
									Text: "time-slot",
									Translations: []textItem{
										{Text: "тайм-слот"},
									},
								},
							},
							Synonyms: []textItem{
								{Text: "раз"},
								{Text: "тайм"},
							},
							Meanings: []textItem{
								{Text: "timing"},
								{Text: "fold"},
								{Text: "half"},
							},
						},
					},
				},
			},
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
		client := YandexDictionaryClient{client: httpClient, apiToken: APItoken, context: context.TODO()}
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
		client := YandexDictionaryClient{client: httpClient, apiToken: APItoken, context: context.TODO()}
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
		client := YandexDictionaryClient{client: httpClient, apiToken: APItoken, context: context.TODO()}
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
						bytes.NewBufferString(`{"head":{},"def":[]}`),
					),
					Header: make(http.Header),
				}, nil
			}),
		}
		client := YandexDictionaryClient{client: httpClient, apiToken: APItoken, context: context.TODO()}
		_, err := client.Translate(word, "en", "ru")
		assert.ErrorIs(t, err, ErrUnknown)

	})
}
