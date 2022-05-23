package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rbhz/tg-dictionary/app/db"
	"github.com/rs/zerolog/log"
)

type UserDictionaryItem struct {
	Word     db.DictionaryItem
	UserItem db.UserDictionaryItem
}

// dictionaryService implements methods for dictionary API
type dictionaryService struct {
	storage db.Storage
}

// GetUserDictionary returns user dictionary
func (d dictionaryService) GetUserDictionary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(CtxUserIDKey).(db.UserID)
	if !ok {
		log.Error().Interface("user", r.Context().Value(CtxUserIDKey)).Msg("invalid user id in context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dictionary, err := d.storage.GetUserDictionary(userID)
	if err != nil {
		log.Error().Err(err).Int64("user", int64(userID)).Msg("failed to get user dictionary")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dict := make([]UserDictionaryItem, 0, len(dictionary))
	for userItem, item := range dictionary {
		dict = append(dict, UserDictionaryItem{Word: item, UserItem: userItem})
	}
	response, jerr := json.Marshal(dict)
	if jerr != nil {
		log.Error().Err(jerr).Int64("user", int64(userID)).Msg("failed to marshal user dictionary")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

// GetWord returns single word data
func (d dictionaryService) GetWord(w http.ResponseWriter, r *http.Request) {
	word := chi.URLParam(r, "word")
	wordData, err := d.storage.Get(word)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("word not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	response, jerr := json.Marshal(wordData)
	if jerr != nil {
		log.Error().Err(jerr).Str("word", word).Msg("failed to marshal dictionary item")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// UpdateWord updates word data
func (d dictionaryService) UpdateWord(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(CtxUserIDKey)
	userID := uid.(db.UserID)
	user, err := d.storage.GetUser(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !user.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
		return
	}
	var word db.DictionaryItem
	if err := json.NewDecoder(r.Body).Decode(&word); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid JSON"))
		return
	}
	if len(word.Meanings) == 0 && len(word.Translations) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("word must have at least one meaning or translation"))
		return
	}
	if err := d.storage.Save(word); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
