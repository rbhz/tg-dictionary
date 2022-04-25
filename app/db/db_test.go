package db

import (
	"testing"
	"tg-dictionary/app/clients/dictionaryapi"
	"tg-dictionary/app/clients/ya_dictionary"

	"github.com/stretchr/testify/assert"
)

func ptrStr(s string) *string {
	return &s
}

func TestNewDictionaryItem(t *testing.T) {
	getDictionaryResponse := func() []dictionaryapi.WordResponse {
		return []dictionaryapi.WordResponse{
			{
				Word:     "test",
				Phonetic: "phon1",
				Phonetics: []dictionaryapi.Phonetic{
					{Text: "phon_in1", Audio: ptrStr("phon_audio1")},
				},
				Origin: "origin22",
				Meanings: []dictionaryapi.Meaning{
					{
						PartOfSpeech: "pos1",
						Definitions: []dictionaryapi.Definition{
							{
								Definition: "def11",
								Example:    "ex11",
								Synonyms:   []string{"syn111", "syn112"},
								Antonyms:   []string{"an111", "an112"},
							},
						},
					},
					{
						PartOfSpeech: "pos2",
						Definitions: []dictionaryapi.Definition{
							{
								Definition: "def12",
								Example:    "ex12",
								Synonyms:   []string{"syn121", "syn122"},
								Antonyms:   []string{"an121", "an122"},
							},
						},
					},
				},
			},
			{
				Word:     "test",
				Phonetic: "phon2",
				Phonetics: []dictionaryapi.Phonetic{
					{Text: "phon_in2", Audio: ptrStr("phon_audio2")},
				},
				Origin: "origin2",
				Meanings: []dictionaryapi.Meaning{
					{
						PartOfSpeech: "pos1",
						Definitions: []dictionaryapi.Definition{
							{
								Definition: "def21",
								Example:    "ex21",
								Synonyms:   []string{},
								Antonyms:   []string{},
							},
						},
					},
				},
			},
		}
	}

	getTranlationsResponse := func() ya_dictionary.TranslationResponse {
		return ya_dictionary.TranslationResponse{
			Definitions: []ya_dictionary.Definition{
				{
					Text:         "Test",
					PartOfSpeech: "pos1",
					Translations: []ya_dictionary.Translation{
						{Text: "тест", PartOfSpeech: "существительное"},
					},
				},
			},
		}
	}

	getExpected := func() DictionaryItem {
		expected := DictionaryItem{
			Word: "test",
			Meanings: []meaning{
				{
					PartOfSpeech: "pos1",
					Definition:   "def11",
					Examples:     []string{"ex11"},
					Synonyms:     []string{"syn111", "syn112"},
					Antonyms:     []string{"an111", "an112"},
				},
				{
					PartOfSpeech: "pos2",
					Definition:   "def12",
					Examples:     []string{"ex12"},
					Synonyms:     []string{"syn121", "syn122"},
					Antonyms:     []string{"an121", "an122"},
				},
				{
					PartOfSpeech: "pos1",
					Definition:   "def21",
					Examples:     []string{"ex21"},
					Synonyms:     []string{},
					Antonyms:     []string{},
				},
			},
			Translations: []translation{
				{Text: "тест", Language: "ru", PartOfSpeech: "pos1"},
			},
		}
		expected.Phonetics.Text = "phon_in1"
		expected.Phonetics.Audio = "phon_audio1"
		return expected
	}

	t.Run("full", func(t *testing.T) {
		actual := NewDictionaryItem(
			"test",
			getDictionaryResponse(),
			map[string]ya_dictionary.TranslationResponse{"ru": getTranlationsResponse()},
		)
		assert.Equal(t, getExpected(), actual)
	})

	t.Run("second_phonetics", func(t *testing.T) {
		response := getDictionaryResponse()
		response[0].Phonetics[0].Audio = nil
		expected := getExpected()
		expected.Phonetics.Text = "phon_in2"
		expected.Phonetics.Audio = "phon_audio2"
		actual := NewDictionaryItem(
			"test", response,
			map[string]ya_dictionary.TranslationResponse{"ru": getTranlationsResponse()},
		)
		assert.Equal(t, expected, actual)
	})
	t.Run("empty", func(t *testing.T) {
		actual := NewDictionaryItem("test", make([]dictionaryapi.WordResponse, 0), nil)
		expected := DictionaryItem{Word: "test"}
		assert.Equal(t, expected, actual)
	})
}
