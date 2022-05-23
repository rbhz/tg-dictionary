package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rbhz/tg-dictionary/app/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserDictionary(t *testing.T) {
	const path = "/api/v1/dictionary"
	t.Run("success", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.Save(db.DictionaryItem{Word: "test"}))
		storage.SaveUserItem(db.UserDictionaryItem{User: db.UserID(testUserID), Word: "test"})

		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		expected := `[{"Word":{"Word":"test","Phonetics":{"Text":"","Audio":""},"Meanings":null,"Translations":null},` +
			`"UserItem":{"Word":"test","User":1,"Created":"0001-01-01T00:00:00Z","LastQuiz":null}}]`
		assert.Equal(t, expected, string(body))
	})
	t.Run("empty", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, `[]`, string(body))
	})
	t.Run("storage error", func(t *testing.T) {
		storage := ErrorStorage{db.NewInMemoryStorage()}
		ts, cancel := getTestServer(storage)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
	})
	t.Run("unauthorized", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))
	})
}

func TestGetWord(t *testing.T) {
	const path = "/api/v1/dictionary/word"
	t.Run("success", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.Save(db.DictionaryItem{Word: "test"}))

		req, err := http.NewRequest(http.MethodGet, ts.URL+path+"/test", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		expected := `{"Word":"test","Phonetics":{"Text":"","Audio":""},"Meanings":null,"Translations":null}`
		assert.Equal(t, expected, string(body))
	})
	t.Run("missing", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path+"/test", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, r.StatusCode)
	})
	t.Run("storage error", func(t *testing.T) {
		storage := ErrorStorage{db.NewInMemoryStorage()}
		ts, cancel := getTestServer(storage)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path+"/test", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
	})
	t.Run("unauthorized", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path+"/test", nil)
		require.NoError(t, err)
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))
	})
}

func TestUpdateWord(t *testing.T) {
	const path = "/api/v1/dictionary/word"
	const wordJson = `{
	"Word":"test",
	"Meanings": [
		{"PartOfSpeech" :"noun", "Definition" : "test word", "Examples": ["test example"]}
	],
	"Translations": [
		{"Text": "тест", "Language": "ru", "PartOfSpeech": "noun"}
	]
}`
	t.Run("success", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID), IsAdmin: true}))
		require.NoError(t, storage.Save(db.DictionaryItem{Word: "test"}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(wordJson))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)
		word, err := storage.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, "test", word.Word)
		assert.Equal(t, 1, len(word.Meanings))
		assert.Equal(t, "noun", word.Meanings[0].PartOfSpeech)
		assert.Equal(t, "test word", word.Meanings[0].Definition)
		assert.Equal(t, []string{"test example"}, word.Meanings[0].Examples)
		assert.Equal(t, 1, len(word.Translations))
		assert.Equal(t, "noun", word.Translations[0].PartOfSpeech)
		assert.Equal(t, "ru", word.Translations[0].Language)
		assert.Equal(t, "тест", word.Translations[0].Text)
	})
	t.Run("new word", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID), IsAdmin: true}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(wordJson))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)
		word, err := storage.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, "test", word.Word)
		assert.Equal(t, 1, len(word.Meanings))
		assert.Equal(t, "noun", word.Meanings[0].PartOfSpeech)
		assert.Equal(t, "test word", word.Meanings[0].Definition)
		assert.Equal(t, []string{"test example"}, word.Meanings[0].Examples)
		assert.Equal(t, 1, len(word.Translations))
		assert.Equal(t, "noun", word.Translations[0].PartOfSpeech)
		assert.Equal(t, "ru", word.Translations[0].Language)
		assert.Equal(t, "тест", word.Translations[0].Text)
	})
	t.Run("invalid json", func(t *testing.T) {
		storage := ErrorStorage{db.NewInMemoryStorage()}
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID), IsAdmin: true}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(`NOT JSON`))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, r.StatusCode)
	})
	t.Run("missing data", func(t *testing.T) {
		storage := ErrorStorage{db.NewInMemoryStorage()}
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID), IsAdmin: true}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(`{"Word": "test"}`))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, r.StatusCode)
	})
	t.Run("storage error", func(t *testing.T) {
		storage := ErrorStorage{db.NewInMemoryStorage()}
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID), IsAdmin: true}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(wordJson))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
	})
	t.Run("not admin user", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		require.NoError(t, storage.SaveUser(db.User{ID: db.UserID(testUserID)}))
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(wordJson))
		require.NoError(t, err)
		req.Header.Set("Authorization", getTestJWT())
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, r.StatusCode)

		_, err = storage.Get("test")
		assert.ErrorIs(t, err, db.ErrNotFound)
	})
	t.Run("unauthorized", func(t *testing.T) {
		storage := db.NewInMemoryStorage()
		ts, cancel := getTestServer(storage)
		defer cancel()
		req, err := http.NewRequest(http.MethodPost, ts.URL+path+"/test", strings.NewReader(wordJson))
		require.NoError(t, err)
		r, err := http.DefaultClient.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))
		_, err = storage.Get("test")
		assert.ErrorIs(t, err, db.ErrNotFound)
	})
}
