package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	prefixWord     = "word:"
	prefixUser     = "user:"
	prefixUserItem = "user_item:"
	prefixQuiz     = "quiz:"
)

type RedisStorage struct {
	db *redis.Client
}

// Get word from redis
func (s *RedisStorage) Get(word string) (DictionaryItem, error) {
	data, err := s.db.Get(context.Background(), prefixWord+word).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return DictionaryItem{}, ErrNotFound
		}
		return DictionaryItem{}, fmt.Errorf("fetching word: %w", err)
	}
	buf := bytes.NewBufferString(data)
	var item DictionaryItem
	if jerr := json.NewDecoder(buf).Decode(&item); jerr != nil {
		return item, fmt.Errorf("unmarshal word: %w", jerr)
	}
	return item, nil
}

// Save word to redis
func (s *RedisStorage) Save(item DictionaryItem) error {
	key := prefixWord + item.Word
	jdata, jerr := json.Marshal(item)
	if jerr != nil {
		return fmt.Errorf("marshal word: %w", jerr)
	}
	_, err := s.db.Set(context.Background(), key, string(jdata), 0).Result()
	if err != nil {
		return fmt.Errorf("saving word: %w", err)
	}
	return nil
}

// GetUser from redis
func (s *RedisStorage) GetUser(id UserID) (User, error) {
	data, err := s.db.Get(context.Background(), prefixUser+strconv.FormatInt(int64(id), 10)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("fetching user: %w", err)
	}
	buf := bytes.NewBufferString(data)
	var user User
	if jerr := json.NewDecoder(buf).Decode(&user); jerr != nil {
		return user, fmt.Errorf("unmarshal word: %w", jerr)
	}
	return user, nil

}

// SaveUser to redis
func (s *RedisStorage) SaveUser(user User) error {
	key := prefixUser + strconv.FormatInt(int64(user.ID), 10)
	jdata, jerr := json.Marshal(user)
	if jerr != nil {
		return fmt.Errorf("marshal user: %w", jerr)
	}
	_, err := s.db.Set(context.Background(), key, string(jdata), 0).Result()
	if err != nil {
		return fmt.Errorf("saving user: %w", err)
	}
	return nil
}

func (s *RedisStorage) GetUserItem(user UserID, word string) (UserDictionaryItem, error) {
	key := prefixUserItem + strconv.FormatInt(int64(user), 10)
	data, err := s.db.HGet(context.Background(), key, word).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return UserDictionaryItem{}, ErrNotFound
		}
		return UserDictionaryItem{}, fmt.Errorf("fetching user item: %w", err)
	}
	buf := bytes.NewBufferString(data)
	var item UserDictionaryItem
	if jerr := json.NewDecoder(buf).Decode(&item); jerr != nil {
		return item, fmt.Errorf("unmarshal user item: %w", jerr)
	}
	return item, nil
}

func (s *RedisStorage) SaveUserItem(item UserDictionaryItem) error {
	key := prefixUserItem + strconv.FormatInt(int64(item.User), 10)
	jdata, jerr := json.Marshal(item)
	if jerr != nil {
		return fmt.Errorf("marshal user item: %w", jerr)
	}
	_, err := s.db.HSet(context.Background(), key, item.Word, string(jdata)).Result()
	if err != nil {
		return fmt.Errorf("saving user item: %w", err)
	}
	return nil
}

// GetUserDictionary from redis
func (s *RedisStorage) GetUserDictionary(user UserID) (map[UserDictionaryItem]DictionaryItem, error) {
	key := prefixUserItem + strconv.FormatInt(int64(user), 10)
	userItems, err := s.db.HGetAll(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return map[UserDictionaryItem]DictionaryItem{}, nil
		}
		return nil, fmt.Errorf("fetching user items: %w", err)
	}
	dict := make(map[UserDictionaryItem]DictionaryItem, len(userItems))
	words := make([]string, 0, len(userItems))
	for word, jdata := range userItems {
		buf := bytes.NewBufferString(jdata)
		var item UserDictionaryItem
		if jerr := json.NewDecoder(buf).Decode(&item); jerr != nil {
			return nil, fmt.Errorf("unmarshal user item: %w", jerr)
		}
		dict[item] = DictionaryItem{Word: word}
		words = append(words, prefixWord+word)
	}
	wordsData := s.db.MGet(context.Background(), words...)
	if err := wordsData.Err(); err != nil {
		return nil, fmt.Errorf("fetching words: %w", err)
	}
	wordsJsons := make(map[string]DictionaryItem, len(wordsData.Val()))
	for _, wd := range wordsData.Val() {
		jdata, ok := wd.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected word data: %w", err)
		}
		word := DictionaryItem{}
		if jerr := json.NewDecoder(bytes.NewBufferString(jdata)).Decode(&word); jerr != nil {
			return nil, fmt.Errorf("unmarshal word: %w", jerr)
		}
		wordsJsons[word.Word] = word
	}
	for item := range dict {
		dict[item] = wordsJsons[item.Word]
	}
	return dict, nil
}

// GetQuiz from redis
func (s *RedisStorage) GetQuiz(id string) (Quiz, error) {
	get := s.db.Get(context.Background(), prefixQuiz+id)
	if err := get.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return Quiz{}, ErrNotFound
		}
		return Quiz{}, fmt.Errorf("fetching quiz: %w", err)
	}
	buf := bytes.NewBufferString(get.Val())
	var quiz Quiz
	if jerr := json.NewDecoder(buf).Decode(&quiz); jerr != nil {
		return quiz, fmt.Errorf("unmarshal quiz: %w", jerr)
	}
	return quiz, nil
}

// Save Quiz to redis
func (s *RedisStorage) SaveQuiz(q Quiz) error {
	key := prefixQuiz + q.ID
	jdata, jerr := json.Marshal(q)
	if jerr != nil {
		return fmt.Errorf("marshal quiz: %w", jerr)
	}
	set := s.db.Set(context.Background(), key, string(jdata), 0)
	if err := set.Err(); err != nil {
		return fmt.Errorf("saving quiz: %w", err)
	}
	return nil
}

// NewRedisStorage creates RedisStorage with given url
func NewRedisStorage(url string) (*RedisStorage, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisStorage{db: rdb}, nil
}
