package db

import (
	"encoding/base64"
	"errors"
	"github.com/rbhz/tg-dictionary/app/clients/dictionaryapi"
	"github.com/rbhz/tg-dictionary/app/clients/ya_dictionary"
	"time"

	"github.com/google/uuid"
)

type UserID int64

var ErrNotFound error = errors.New("not found")

func GenerateID() string {
	id := [16]byte(uuid.New())
	return base64.RawURLEncoding.EncodeToString(id[:])
}

// Storage defines method provided by database interfaces
type Storage interface {
	// Get dictionary item by word, nil if does not exists
	Get(string) (DictionaryItem, error)
	// Save dictionary item to DB
	Save(DictionaryItem) error

	// GetUserDictionary returns item from user dictionary
	GetUserItem(UserID, string) (UserDictionaryItem, error)
	// SaveUserItem saves UserDictionaryItem
	SaveUserItem(UserDictionaryItem) error
	// GetUserDictionary returns map of user dictionary items
	GetUserDictionary(UserID) (map[UserDictionaryItem]DictionaryItem, error)
	// // SaveQuiz saves quiz to DB

	SaveQuiz(Quiz) error
	// GetQuiz returns quiz by ID
	GetQuiz(string) (Quiz, error)
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

// UserDictionaryItem hold data for a single user dictionary item
type UserDictionaryItem struct {
	Word     string
	User     UserID
	Created  time.Time
	LastQuiz *time.Time
}

// QuizResult holds data for a quiz result
type QuizResult struct {
	Choice  int
	Correct bool
}

// QuizItem holds data for a single quiz item
type QuizItem struct {
	Word         string
	Translations []string
	Correct      bool
}

// Quiz holds data for a single quiz`
type Quiz struct {
	ID       string
	User     UserID
	Word     string
	Language string
	Choices  []QuizItem
	Created  time.Time
	Result   *QuizResult
}

func (q *Quiz) SetResult(choice int, s Storage) error {
	if choice < 0 || choice >= len(q.Choices) {
		return errors.New("invalid choice")
	}
	if q.Result != nil {
		return errors.New("result already set")
	}

	q.Result = &QuizResult{
		Choice:  choice,
		Correct: q.Choices[choice].Correct,
	}
	if q.Result.Correct {
		item, err := s.GetUserItem(q.User, q.Word)
		if err != nil {
			return err
		}
		now := time.Now()
		item.LastQuiz = &now
		s.SaveUserItem(item)
	}
	s.SaveQuiz(*q)
	return nil
}

func NewQuiz(user UserID, word string, lang string, items []QuizItem) Quiz {
	return Quiz{
		ID:       GenerateID(),
		User:     user,
		Word:     word,
		Language: lang,
		Choices:  items,
		Created:  time.Now(),
	}
}
