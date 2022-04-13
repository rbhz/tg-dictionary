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

func TestGet(t *testing.T) {
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
		assert.Equal(t, savedItem, *item)
	})
	t.Run("non existing", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		item, err := storage.Get(word)
		require.NoError(t, err)
		assert.Nil(t, item)
	})
	t.Run("invalid json", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		err := storage.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketDictionary))
			return bucket.Put([]byte(word), []byte("NON_JSON_DATA"))
		})
		require.NoError(t, err)

		item, err := storage.Get(word)
		assert.Error(t, err)
		assert.Nil(t, item)
	})
}

func TestSave(t *testing.T) {
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

func TestSaveForUser(t *testing.T) {
	user := UserID(1)
	t.Run("new", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		item := getItem()
		err := storage.SaveForUser(item, user)
		require.NoError(t, err)
		var ui UserDictionaryItem
		storage.db.View(func(tx *bolt.Tx) error {
			jdata := tx.Bucket([]byte(bucketUsersDictionaries)).Bucket([]byte("1")).Get([]byte(item.Word))
			jerr := json.Unmarshal(jdata, &ui)
			require.NoError(t, jerr)
			assert.Equal(t, ui.Word, item.Word)
			assert.Equal(t, ui.User, user)
			assert.Less(t, ui.Created, time.Now())
			assert.Greater(t, ui.Created, time.Now().Add(-1*time.Minute))
			return nil
		})
	})

	t.Run("new", func(t *testing.T) {
		storage, cleanup := getStorage(t)
		defer cleanup()
		item := getItem()
		existing := UserDictionaryItem{Word: item.Word, User: user, Created: time.Now().Truncate(time.Nanosecond)}
		existingJSON, jerr := json.Marshal(existing)
		require.NoError(t, jerr)
		storage.db.Update(func(tx *bolt.Tx) error {
			usrBucket, err := tx.Bucket([]byte(bucketUsersDictionaries)).CreateBucket([]byte("1"))
			require.NoError(t, err)
			usrBucket.Put([]byte(item.Word), existingJSON)
			require.NoError(t, err)
			return nil
		})
		err := storage.SaveForUser(item, user)
		require.NoError(t, err)
		var ui UserDictionaryItem
		storage.db.View(func(tx *bolt.Tx) error {
			jdata := tx.Bucket([]byte(bucketUsersDictionaries)).Bucket([]byte("1")).Get([]byte(item.Word))
			jerr := json.Unmarshal(jdata, &ui)
			require.NoError(t, jerr)
			assert.Equal(t, existing.Word, ui.Word)
			assert.Equal(t, existing.User, ui.User)
			assert.Equal(t, existing.Created, ui.Created)
			return nil
		})
	})

}
