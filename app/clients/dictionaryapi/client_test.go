package dictionaryapi

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleResponse = `[
	{
		"word": "hello",
		"phonetic": "həˈləʊ",
		"phonetics": [
		{
			"text": "həˈləʊ",
			"audio": "//ssl.gstatic.com/dictionary/static/sounds/20200429/hello--_gb_1.mp3"
		},
		{
			"text": "hɛˈləʊ"
		}
		],
		"origin": "early 19th century: variant of earlier hollo ; related to holla.",
		"meanings": [
		{
			"partOfSpeech": "exclamation",
			"definitions": [
			{
				"definition": "used as a greeting or to begin a phone conversation.",
				"example": "hello there, Katie!",
				"synonyms": ["syn1", "syn2"],
				"antonyms": ["an1", "an2"]
			}
			]
		},
		{
			"partOfSpeech": "verb",
			"definitions": [
			{
				"definition": "say or shout ‘hello’.",
				"example": "I pressed the phone button and helloed",
				"synonyms": [],
				"antonyms": []
			}
			]
		}
		]
	}
]
`

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func ptrStr(s string) *string {
	return &s
}

func TestGet(t *testing.T) {
	validURL := "https://api.dictionaryapi.dev/api/v2/entries/en/hello"
	word := "hello"
	t.Run("success", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validURL, req.URL.String())
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(exampleResponse)),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := Client{client: httpClient, context: context.TODO()}
		items, err := client.Get(word)
		assert.NoError(t, err)
		expected := []WordResponse{
			{
				Word:     "hello",
				Phonetic: "həˈləʊ",
				Phonetics: []Phonetic{
					{
						Text:  "həˈləʊ",
						Audio: ptrStr("//ssl.gstatic.com/dictionary/static/sounds/20200429/hello--_gb_1.mp3"),
					},
					{Text: "hɛˈləʊ", Audio: nil},
				},
				Origin: "early 19th century: variant of earlier hollo ; related to holla.",
				Meanings: []Meaning{
					{
						PartOfSpeech: "exclamation",
						Definitions: []Definition{
							{
								Definition: "used as a greeting or to begin a phone conversation.",
								Example:    "hello there, Katie!",
								Synonyms:   []string{"syn1", "syn2"},
								Antonyms:   []string{"an1", "an2"},
							},
						},
					},
					{
						PartOfSpeech: "verb",
						Definitions: []Definition{
							{
								Definition: "say or shout ‘hello’.",
								Example:    "I pressed the phone button and helloed",
								Synonyms:   []string{},
								Antonyms:   []string{},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected, items)
	})
	t.Run("reqest error", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validURL, req.URL.String())
				return &http.Response{}, http.ErrServerClosed
			}),
		}
		client := Client{client: httpClient, context: context.TODO()}
		items, err := client.Get(word)
		assert.ErrorIs(t, err, http.ErrServerClosed)
		assert.Nil(t, items)
	})
	t.Run("invalid response", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validURL, req.URL.String())
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("Invalid JSON")),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := Client{client: httpClient, context: context.TODO()}
		items, err := client.Get(word)
		assert.Error(t, err)
		assert.Nil(t, items)
	})
	t.Run("error status", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validURL, req.URL.String())
				return &http.Response{
					StatusCode: 400,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"status": "ERROR"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := Client{client: httpClient, context: context.TODO()}
		items, err := client.Get(word)
		assert.Error(t, err)
		assert.Nil(t, items)
	})
	t.Run("error status 404", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, validURL, req.URL.String())
				return &http.Response{
					StatusCode: 404,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"status": "ERROR"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
		client := Client{client: httpClient, context: context.TODO()}
		items, err := client.Get(word)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, items)
	})
}
