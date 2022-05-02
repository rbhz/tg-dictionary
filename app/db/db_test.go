package db

import (
	"testing"
	"tg-dictionary/app/clients/dictionaryapi"
	"tg-dictionary/app/clients/ya_dictionary"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewQuiz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := Quiz{
			User:     UserID(1),
			Word:     "test",
			Language: "ru",
			Choices: []QuizItem{
				{
					Word:         "t1",
					Translations: []string{"t1", "t2"},
					Correct:      false,
				},
				{
					Word:         "t2",
					Translations: []string{"t1", "t2"},
					Correct:      false,
				},
				{
					Word:         "test",
					Translations: []string{"t1", "t2"},
					Correct:      true,
				},
			},
		}
		actual := NewQuiz(UserID(1), expected.Word, expected.Language, expected.Choices)
		assert.NotEmpty(t, actual.ID)
		assert.NotEmpty(t, actual.Created)
		expected.ID = actual.ID
		expected.Created = actual.Created
		assert.Equal(t, expected, actual)
	})
}

func TestQuizSetResult(t *testing.T) {
	getQuiz := func() Quiz {
		return Quiz{
			ID:       "1",
			User:     UserID(1),
			Word:     "test",
			Language: "ru",
			Choices: []QuizItem{
				{
					Word:         "t1",
					Translations: []string{"t1", "t2"},
					Correct:      false,
				},
				{
					Word:         "t2",
					Translations: []string{"t1", "t2"},
					Correct:      false,
				},
				{
					Word:         "test",
					Translations: []string{"t1", "t2"},
					Correct:      true,
				},
			},
		}
	}
	t.Run("success correct", func(t *testing.T) {
		storage := NewInMemoryStorage()
		quiz := getQuiz()
		require.NoError(t, storage.SaveQuiz(quiz))
		require.NoError(t, storage.SaveUserItem(UserDictionaryItem{
			Word:    quiz.Word,
			User:    quiz.User,
			Created: time.Now()}))

		assert.NoError(t, quiz.SetResult(2, storage))
		require.NotNil(t, quiz.Result)
		assert.True(t, quiz.Result.Correct)
		assert.Equal(t, 2, quiz.Result.Choice)

		dbQuiz, err := storage.GetQuiz(quiz.ID)
		require.NoError(t, err)
		assert.Equal(t, quiz, dbQuiz)
	})
	t.Run("success wrong", func(t *testing.T) {
		storage := NewInMemoryStorage()
		quiz := getQuiz()
		require.NoError(t, storage.SaveQuiz(quiz))
		require.NoError(t, storage.SaveUserItem(UserDictionaryItem{
			Word:    quiz.Word,
			User:    quiz.User,
			Created: time.Now()}))

		assert.NoError(t, quiz.SetResult(1, storage))
		require.NotNil(t, quiz.Result)
		assert.False(t, quiz.Result.Correct)
		assert.Equal(t, 1, quiz.Result.Choice)

		dbQuiz, err := storage.GetQuiz(quiz.ID)
		require.NoError(t, err)
		assert.Equal(t, quiz, dbQuiz)
	})
	t.Run("invalid choice", func(t *testing.T) {
		for _, choice := range []int{-1, 3} {
			storage := NewInMemoryStorage()
			quiz := getQuiz()
			require.NoError(t, storage.SaveQuiz(quiz))
			require.NoError(t, storage.SaveUserItem(UserDictionaryItem{
				Word:    quiz.Word,
				User:    quiz.User,
				Created: time.Now()}))
			assert.Error(t, quiz.SetResult(choice, storage))
		}
	})
	t.Run("already set", func(t *testing.T) {
		storage := NewInMemoryStorage()
		quiz := getQuiz()
		require.NoError(t, storage.SaveQuiz(quiz))
		require.NoError(t, storage.SaveUserItem(UserDictionaryItem{
			Word:    quiz.Word,
			User:    quiz.User,
			Created: time.Now()}))
		assert.NoError(t, quiz.SetResult(2, storage))
		assert.Error(t, quiz.SetResult(2, storage))
	})
}
