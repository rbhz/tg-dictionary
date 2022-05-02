package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryGet(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		storage := NewInMemoryStorage()
		item := DictionaryItem{Word: "test"}
		require.NoError(t, storage.Save(item))
		resItem, err := storage.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, item, resItem)
	})

	t.Run("not found", func(t *testing.T) {
		storage := NewInMemoryStorage()
		_, err := storage.Get("word")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestInMemorySave(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := NewInMemoryStorage()
		item := DictionaryItem{Word: "test"}
		storage.Save(item)
		resItem, err := storage.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, item, resItem)
	})
}

func TestInMemorySaveUserItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := NewInMemoryStorage()
		item := UserDictionaryItem{
			User:    UserID(1),
			Word:    "test",
			Created: time.Now(),
		}
		require.NoError(t, storage.SaveUserItem(item))
		resItem, err := storage.GetUserItem(UserID(1), "test")
		assert.NoError(t, err)
		assert.Equal(t, item, resItem)
	})
}

func TestInMemoryGetUserItem(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		storage := NewInMemoryStorage()
		item := UserDictionaryItem{
			User:    UserID(1),
			Word:    "test",
			Created: time.Now(),
		}
		require.NoError(t, storage.SaveUserItem(item))
		resItem, err := storage.GetUserItem(item.User, "test")
		assert.NoError(t, err)
		assert.Equal(t, item, resItem)
	})

	t.Run("not found", func(t *testing.T) {
		storage := NewInMemoryStorage()
		_, err := storage.GetUserItem(UserID(1), "test")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestInMemoryGetUserDictionary(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		storage := NewInMemoryStorage()
		item1 := DictionaryItem{Word: "test"}
		userItem1 := UserDictionaryItem{Word: item1.Word, User: UserID(1), Created: time.Now()}
		require.NoError(t, storage.Save(item1))
		require.NoError(t, storage.SaveUserItem(userItem1))

		item2 := DictionaryItem{Word: "test2"}
		userItem2 := UserDictionaryItem{Word: item2.Word, User: UserID(1), Created: time.Now()}
		require.NoError(t, storage.Save(item2))
		require.NoError(t, storage.SaveUserItem(userItem2))

		item3 := DictionaryItem{Word: "test3"} // other user
		userItem3 := UserDictionaryItem{Word: item3.Word, User: UserID(2), Created: time.Now()}
		require.NoError(t, storage.Save(item3))
		require.NoError(t, storage.SaveUserItem(userItem3))

		res, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, map[UserDictionaryItem]DictionaryItem{userItem1: item1, userItem2: item2}, res)

	})
	t.Run("empty", func(t *testing.T) {
		storage := NewInMemoryStorage()
		res, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, map[UserDictionaryItem]DictionaryItem{}, res)
	})
}
