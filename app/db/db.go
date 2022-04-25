package db

import (
	"tg-dictionary/app/clients/dictionaryapi"
	"tg-dictionary/app/clients/ya_dictionary"
	"time"
)

type UserID int64

// Storage defines method provided by database interfaces
type Storage interface {
	// Get dictionary item by word, nil if does not exists
	Get(string) (*DictionaryItem, error)
	// Save dictionary item to DB
	Save(DictionaryItem) error
	// SaveForUser dictionary item to user dictionary
	SaveForUser(DictionaryItem, UserID) error
}

type UserDictionaryItem struct {
	Word    string
	User    UserID
	Created time.Time
}

// DictionaryItem hold data for a single dictionary item
type DictionaryItem struct {
	Word      string
	Phonetics struct {
		Text  string
		Audio string
	}
	Meanings     []meaning
	Translations []translation
}

type meaning struct {
	PartOfSpeech string
	Definition   string
	Examples     []string
	Synonyms     []string
	Antonyms     []string
}

type translation struct {
	Text         string
	Audio        string
	Language     string
	PartOfSpeech string
}

func NewDictionaryItem(
	word string,
	dictionaryResponse []dictionaryapi.WordResponse,
	translations map[string]ya_dictionary.TranslationResponse,
) DictionaryItem {
	item := DictionaryItem{Word: word}
	var phoneticsText, phoneticAudio string
	for _, ri := range dictionaryResponse {
		if phoneticsText == "" || phoneticAudio == "" {
			if word != ri.Word {
				continue
			}
			if phoneticsText == "" {
				phoneticsText = ri.Phonetic
			}
			for _, p := range ri.Phonetics {
				if phoneticsText == "" {
					phoneticsText = p.Text
				}
				if phoneticAudio == "" && p.Audio != nil {
					phoneticsText = p.Text
					phoneticAudio = *p.Audio
				}
			}
		}
		for _, m := range ri.Meanings {
			for _, d := range m.Definitions {
				m := meaning{
					PartOfSpeech: m.PartOfSpeech,
					Definition:   d.Definition,
					Antonyms:     d.Antonyms,
					Synonyms:     d.Synonyms,
				}
				if d.Example != "" {
					m.Examples = []string{d.Example}
				}
				item.Meanings = append(item.Meanings, m)
			}
		}
		item.Phonetics.Text = phoneticsText
		item.Phonetics.Audio = phoneticAudio
	}

	if translations != nil {
		for lang, tranlationResponse := range translations {
			for _, d := range tranlationResponse.Definitions {
				for _, t := range d.Translations {
					item.Translations = append(item.Translations, translation{
						Text:         t.Text,
						Language:     lang,
						PartOfSpeech: d.PartOfSpeech,
					})
				}
			}
		}
	}
	return item
}
