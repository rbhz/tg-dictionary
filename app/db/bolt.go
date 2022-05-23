package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

const (
	bucketDictionary        = "Dictionary"
	bucketUsers             = "Users"
	bucketUsersDictionaries = "UsersDictionaries"
	bucketQuizzes           = "Quizzes"
)

// BoltStorage implements storage interface for BoltDB
type BoltStorage struct {
	db *bolt.DB
}

// Get dictionary item from database
func (b *BoltStorage) Get(word string) (DictionaryItem, error) {
	var res *DictionaryItem
	if err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		res, err = b.getItem(word, tx)
		return err
	}); err != nil {
		return DictionaryItem{}, err
	}
	if res == nil {
		return DictionaryItem{}, ErrNotFound
	}
	return *res, nil
}

func (b *BoltStorage) getItem(word string, tx *bolt.Tx) (*DictionaryItem, error) {
	bucket := tx.Bucket([]byte(bucketDictionary))
	jdata := bucket.Get([]byte(word))
	if len(jdata) == 0 {
		return nil, nil
	}
	var res *DictionaryItem
	if err := json.Unmarshal(jdata, &res); err != nil {
		return nil, fmt.Errorf("unmarshal dictionary item: %w", err)
	}
	return res, nil
}

// Save dictionary item to database
func (b *BoltStorage) Save(item DictionaryItem) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDictionary))
		jdata, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal word: %w", err)
		}
		if err := bucket.Put([]byte(item.Word), jdata); err != nil {
			return fmt.Errorf("put word: %w", err)
		}
		return nil
	})
}

func (b *BoltStorage) GetUser(user UserID) (User, error) {
	var res User
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsers))
		jdata := bucket.Get([]byte(strconv.FormatInt(int64(user), 10)))
		if len(jdata) == 0 {
			return ErrNotFound
		}
		if err := json.Unmarshal(jdata, &res); err != nil {
			return fmt.Errorf("unmarshal user: %w", err)
		}
		return nil
	}); err != nil {
		return User{}, err
	}
	return res, nil
}

func (b *BoltStorage) SaveUser(user User) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsers))
		jdata, err := json.Marshal(user)

		if err != nil {
			return fmt.Errorf("marshal user: %w", err)
		}
		if err := bucket.Put([]byte(strconv.FormatInt(int64(user.ID), 10)), jdata); err != nil {
			return fmt.Errorf("put user: %w", err)
		}
		return nil
	})
}

func (b *BoltStorage) GetUserItem(user UserID, word string) (UserDictionaryItem, error) {
	var res UserDictionaryItem
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsersDictionaries))
		userBucket := bucket.Bucket([]byte(strconv.FormatInt(int64(user), 10)))
		if userBucket == nil {
			return ErrNotFound
		}
		jdata := userBucket.Get([]byte(word))
		if len(jdata) == 0 {
			return ErrNotFound
		}
		if err := json.Unmarshal(jdata, &res); err != nil {
			return fmt.Errorf("unmarshal user dictionary item: %w", err)
		}
		return nil
	}); err != nil {
		return UserDictionaryItem{}, err
	}
	return res, nil
}

func (b *BoltStorage) SaveUserItem(item UserDictionaryItem) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsersDictionaries))
		userBucket, err := bucket.CreateBucketIfNotExists([]byte(strconv.FormatInt(int64(item.User), 10)))
		if err != nil {
			return fmt.Errorf("create user bucket: %w", err)
		}
		jdata, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal user dictionary item: %w", err)
		}
		userBucket.Put([]byte(item.Word), jdata)
		return nil
	})
}

func (b *BoltStorage) GetUserDictionary(user UserID) (map[UserDictionaryItem]DictionaryItem, error) {
	res := make(map[UserDictionaryItem]DictionaryItem)
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsersDictionaries))
		userBucket := bucket.Bucket([]byte(strconv.FormatInt(int64(user), 10)))
		if userBucket == nil {
			return nil
		}
		c := userBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var item UserDictionaryItem
			if err := json.Unmarshal(v, &item); err != nil {
				return fmt.Errorf("unmarshal user dictionary item: %w", err)
			}
			if item.Word == "" {
				return errors.New("word is empty")
			}
			dicItem, err := b.getItem(item.Word, tx)
			if err != nil {
				return fmt.Errorf("get dictionary item: %w", err)
			}
			res[item] = *dicItem
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// SaveQuiz saves quiz to database
func (b *BoltStorage) SaveQuiz(quiz Quiz) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQuizzes))
		jdata, jerr := json.Marshal(quiz)
		if jerr != nil {
			return fmt.Errorf("marshal quiz: %w", jerr)
		}
		bucket.Put([]byte(quiz.ID), jdata)
		return nil
	})
}

// GetQuiz returns quiz by ID`
func (b *BoltStorage) GetQuiz(id string) (Quiz, error) {
	var res Quiz
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQuizzes))
		jdata := bucket.Get([]byte(id))
		if len(jdata) == 0 {
			return ErrNotFound
		}
		if err := json.Unmarshal(jdata, &res); err != nil {
			return fmt.Errorf("unmarshal quiz: %w", err)
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, nil
}

// NewBoltStorage creates BoltStorage instance and initialize buckets
func NewBoltStorage(db *bolt.DB) (*BoltStorage, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{bucketUsers, bucketUsersDictionaries, bucketDictionary, bucketQuizzes} {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &BoltStorage{db: db}, nil
}
