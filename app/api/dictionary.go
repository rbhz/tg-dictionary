package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rbhz/tg-dictionary/app/db"
	"github.com/rs/zerolog/log"
)

// UserDictionaryItem represents user dictionary item in API response
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
	userID, ok := r.Context().Value(ctxUserIDKey).(db.UserID)
	if !ok {
		log.Error().Interface("user", r.Context().Value(ctxUserIDKey)).Msg("invalid user id in context")
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
	if _, err := w.Write(response); err != nil {
		log.Warn().Err(err).Msg("failed to write response")
	}
}

// GetWord returns single word data
func (d dictionaryService) GetWord(w http.ResponseWriter, r *http.Request) {
	word := chi.URLParam(r, "word")
	wordData, err := d.storage.Get(word)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("word not found")); err != nil {
				log.Warn().Err(err).Msg("failed to write response")
			}
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
	if _, err := w.Write(response); err != nil {
		log.Warn().Err(err).Msg("failed to write response")
	}
}

// UpdateWord updates word data
func (d dictionaryService) UpdateWord(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(ctxUserIDKey)
	userID := uid.(db.UserID)
	user, err := d.storage.GetUser(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !user.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte("forbidden")); err != nil {
			log.Warn().Err(err).Msg("failed to write response")
		}
		return
	}
	var word db.DictionaryItem
	if err := json.NewDecoder(r.Body).Decode(&word); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("invalid JSON")); err != nil {
			log.Warn().Err(err).Msg("failed to write response")
		}
		return
	}
	if len(word.Meanings) == 0 && len(word.Translations) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("word must have at least one meaning or translation")); err != nil {
			log.Warn().Err(err).Msg("failed to write response")
		}
		return
	}
	if err := d.storage.Save(word); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
