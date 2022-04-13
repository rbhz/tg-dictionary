package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	bucketUsersDictionaries = "UsersDictionaries"
	bucketDictionary        = "Dictionary"
)

// BoltStorage implements storage interface for BoltDB
type BoltStorage struct {
	db *bolt.DB
}

// Get dictionary item from database
func (b *BoltStorage) Get(word string) (*DictionaryItem, error) {
	var res *DictionaryItem
	if err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDictionary))
		jdata := bucket.Get([]byte(word))
		if len(jdata) == 0 {
			return nil
		}
		if err := json.Unmarshal(jdata, &res); err != nil {
			return fmt.Errorf("failed to unmarshal dictionary item: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// Save dictionary item to database
func (b *BoltStorage) Save(item DictionaryItem) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDictionary))
		jdata, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		if err := bucket.Put([]byte(item.Word), jdata); err != nil {
			return fmt.Errorf("failed to put event: %w", err)
		}
		return nil
	})
}

// Save dictionary item to user dictionary
func (b *BoltStorage) SaveForUser(item DictionaryItem, user UserID) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUsersDictionaries))
		userBucket, err := bucket.CreateBucketIfNotExists([]byte(strconv.FormatInt(int64(user), 10)))
		if err != nil {
			return errors.New("failed to create bucket")
		}
		if len(userBucket.Get([]byte(item.Word))) != 0 {
			// already exists
			return nil
		}
		obj := UserDictionaryItem{User: user, Word: item.Word, Created: time.Now()}
		jdata, err := json.Marshal(obj)
		if err != nil {
			return errors.New("failed to marshal users event")
		}
		userBucket.Put([]byte(item.Word), jdata)
		return nil
	})
}

// NewBoltStorage creates BoltStorage instance and initialize buckets
func NewBoltStorage(db *bolt.DB) (*BoltStorage, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{bucketUsersDictionaries, bucketDictionary} {
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
