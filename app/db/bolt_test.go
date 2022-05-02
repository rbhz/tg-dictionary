package db

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func getBoltDB(t *testing.T) (*bolt.DB, func()) {
	tmpFile, err := ioutil.TempFile("", "bolt_test")
	require.NoError(t, err)
	boltDB, err := bolt.Open(tmpFile.Name(), 0600, nil)
	require.NoError(t, err)
	return boltDB, func() {
		os.Remove(tmpFile.Name())
		boltDB.Close()
	}
}

func getStorage(t *testing.T) (*BoltStorage, func()) {
	boltDB, cleanup := getBoltDB(t)
	storage, err := NewBoltStorage(boltDB)
	require.NoError(t, err)
	return storage, cleanup
}

func getItem() DictionaryItem {
	meanings := []meaning{
		{
			PartOfSpeech: "pof1",
			Definition:   "def1",
			Examples:     []string{"ex11", "ex12"},
			Synonyms:     []string{"sn11", "sn12"},
			Antonyms:     []string{"an11", "an12"},
		},
		{
			PartOfSpeech: "pof2",
			Definition:   "def2",
			Examples:     []string{"ex21", "ex22"},
			Synonyms:     []string{"sn21", "sn22"},
			Antonyms:     []string{"an21", "an22"},
		},
	}

	item := DictionaryItem{
		Word:     "test",
		Meanings: meanings,
	}
	item.Phonetics.Text = "pText"
	item.Phonetics.Audio = "pAudio"
	return item
}

func TestNewBoltStorage(t *testing.T) {
	buckets := []string{
		bucketDictionary,
		bucketUsersDictionaries,
	}
	t.Run("first", func(t *testing.T) {
		boltDB, cleanup := getBoltDB(t)
		defer cleanup()
		storage, err := NewBoltStorage(boltDB)
		require.NoError(t, err)
		storage.db.View(func(tx *bolt.Tx) error {
			for _, b := range buckets {
				assert.NotNil(t, tx.Bucket([]byte(b)))
				assert.Equal(t, 0, tx.Bucket([]byte(b)).Stats().KeyN)
			}
			return nil
		})
	})
	t.Run("already exists", func(t *testing.T) {
		boltDB, cleanup := getBoltDB(t)
		defer cleanup()
		err := boltDB.Update(func(tx *bolt.Tx) error {
			for _, b := range buckets {
				if _, err := tx.CreateBucket([]byte(b)); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)

		storage, err := NewBoltStorage(boltDB)
		require.NoError(t, err)
		storage.db.View(func(tx *bolt.Tx) error {
			for _, b := range buckets {
				assert.NotNil(t, tx.Bucket([]byte(b)))
				assert.Equal(t, 0, tx.Bucket([]byte(b)).Stats().KeyN)
			}
			return nil
		})
	})
}

func TestBoltGet(t *testing.T) {
	word := "test"
	t.Run("ok", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		savedItem := getItem()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketDictionary))
			jdata, jerr := json.Marshal(savedItem)
			require.NoError(t, jerr)
			return bucket.Put([]byte(word), jdata)
		})
		require.NoError(t, err)

		item, err := storage.Get(word)
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, savedItem, item)
	})
	t.Run("non existing", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		_, err := storage.Get(word)
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid json", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketDictionary))
			return bucket.Put([]byte(word), []byte("NON_JSON_DATA"))
		})
		require.NoError(t, err)

		_, err = storage.Get(word)
		assert.Error(t, err)
	})
}

func TestBoltSave(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		item := getItem()
		jdata, jerr := json.Marshal(item)
		require.NoError(t, jerr)
		err := storage.Save(item)
		assert.NoError(t, err)
		storage.db.View(func(tx *bolt.Tx) error {
			wordData := tx.Bucket([]byte(bucketDictionary)).Get([]byte(item.Word))
			assert.Equal(t, wordData, jdata)
			return nil
		})
	})
}

func TestBoltGetUserItem(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		userID := UserID(1)
		item := UserDictionaryItem{
			Word: "test",
			User: userID,
		}
		err := storage.db.Update(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketUsersDictionaries))
			bucket, err := allBucket.CreateBucket([]byte("1"))
			if err != nil {
				return err
			}
			jdata, jerr := json.Marshal(item)
			require.NoError(t, jerr)
			return bucket.Put([]byte(item.Word), jdata)
		})
		require.NoError(t, err)

		userItem, err := storage.GetUserItem(userID, item.Word)
		require.NoError(t, err)
		require.NotNil(t, userItem)
		assert.Equal(t, item, userItem)
	})
	t.Run("non existing", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		_, err := storage.GetUserItem(UserID(1), "test")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("non existing with sub bucket", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketUsersDictionaries))
			_, err := allBucket.CreateBucket([]byte("1"))
			return err
		})
		require.NoError(t, err)

		_, err = storage.GetUserItem(UserID(1), "test")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid json", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketUsersDictionaries))
			bucket, err := allBucket.CreateBucket([]byte("1"))
			if err != nil {
				return err
			}
			return bucket.Put([]byte("test"), []byte("NON_JSON_DATA"))
		})
		require.NoError(t, err)

		_, err = storage.GetUserItem(UserID(1), "test")
		assert.Error(t, err)
	})
}
func TestBoltSaveUserItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		userID := UserID(1)
		item := UserDictionaryItem{
			Word: "test",
			User: userID,
		}
		err := storage.SaveUserItem(item)
		assert.NoError(t, err)
		storage.db.View(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketUsersDictionaries))
			bucket := allBucket.Bucket([]byte("1"))
			require.NotNil(t, bucket)
			jdata, jerr := json.Marshal(item)
			require.NoError(t, jerr)
			assert.Equal(t, jdata, bucket.Get([]byte(item.Word)))
			return nil
		})
	})
	t.Run("rewrite", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		userID := UserID(1)
		item := UserDictionaryItem{
			Word: "test",
			User: userID,
		}
		require.NoError(t, storage.SaveUserItem(item))
		lastQuiz := time.Now().Truncate(time.Nanosecond)
		item.LastQuiz = &lastQuiz
		assert.NoError(t, storage.SaveUserItem(item))
		storage.db.View(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketUsersDictionaries))
			bucket := allBucket.Bucket([]byte("1"))
			require.NotNil(t, bucket)
			jdata, jerr := json.Marshal(item)
			require.NoError(t, jerr)
			assert.Equal(t, jdata, bucket.Get([]byte(item.Word)))
			return nil
		})
	})
}

func TestBoltGetUserDictionary(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		item1 := DictionaryItem{Word: "test"}
		userItem1 := UserDictionaryItem{Word: item1.Word, User: UserID(1)}
		require.NoError(t, storage.Save(item1))
		require.NoError(t, storage.SaveUserItem(userItem1))

		item2 := DictionaryItem{Word: "test2"}
		userItem2 := UserDictionaryItem{Word: item2.Word, User: UserID(1)}
		require.NoError(t, storage.Save(item2))
		require.NoError(t, storage.SaveUserItem(userItem2))

		item3 := DictionaryItem{Word: "test3"} // other user
		userItem3 := UserDictionaryItem{Word: item3.Word, User: UserID(2)}
		require.NoError(t, storage.Save(item3))
		require.NoError(t, storage.SaveUserItem(userItem3))

		res, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, map[UserDictionaryItem]DictionaryItem{userItem1: item1, userItem2: item2}, res)
	})
	t.Run("empty", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		expected := make(map[UserDictionaryItem]DictionaryItem)
		actual, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)

	})
}
func TestBoltSaveQuiz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		quiz := Quiz{
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
		err := storage.SaveQuiz(quiz)
		assert.NoError(t, err)
		storage.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketQuizzes))
			jdata, jerr := json.Marshal(quiz)
			require.NoError(t, jerr)
			assert.Equal(t, jdata, bucket.Get([]byte(quiz.ID)))
			return nil
		})
	})
}
func TestBoltGetQuiz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		quiz := Quiz{
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
		err := storage.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketQuizzes))
			jdata, jerr := json.Marshal(quiz)
			require.NoError(t, jerr)
			return bucket.Put([]byte(quiz.ID), jdata)
		})
		require.NoError(t, err)
		assert.NoError(t, err)

		dbQuiz, err := storage.GetQuiz(quiz.ID)
		assert.NoError(t, err)
		assert.Equal(t, quiz, dbQuiz)
	})
	t.Run("not found", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		_, err := storage.GetQuiz("1")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			allBucket := tx.Bucket([]byte(bucketQuizzes))
			bucket, err := allBucket.CreateBucket([]byte("1"))
			if err != nil {
				return err
			}
			return bucket.Put([]byte("test"), []byte("NON_JSON_DATA"))
		})
		require.NoError(t, err)

		_, err = storage.GetQuiz("1")
		assert.Error(t, err)
	})
}
