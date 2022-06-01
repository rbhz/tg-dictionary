package db

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisGet(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("word:test").SetVal("{\"word\":\"test\"}")

		word, err := storage.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, DictionaryItem{Word: "test"}, word)
	})
	t.Run("not_found", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("word:test").RedisNil()

		_, err := storage.Get("test")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("word:test").SetVal("NOT_JSON")

		_, err := storage.Get("test")
		assert.Error(t, err)
	})
}

func TestRedisSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		expected := `{"Word":"test","Phonetics":{"Text":"","Audio":""},"Meanings":null,"Translations":null}`
		mock.ExpectSet("word:test", expected, 0).SetVal("OK")

		err := storage.Save(DictionaryItem{Word: "test"})
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		expected := `{"Word":"test","Phonetics":{"Text":"","Audio":""},"Meanings":null,"Translations":null}`
		mock.ExpectSet("word:test", expected, 0).SetErr(errors.New("FAIL"))

		err := storage.Save(DictionaryItem{Word: "test"})
		assert.Error(t, err)
	})
}

func TestGetUser(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("user:1").SetVal("{\"ID\":1,\"Username\":\"test\",\"Language\":\"en\"}")

		user, err := storage.GetUser(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, User{ID: 1, Username: "test", Language: "en"}, user)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("user:1").RedisNil()

		_, err := storage.GetUser(UserID(1))
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("user:1").SetVal("NOT_JSON")

		_, err := storage.GetUser(UserID(1))
		assert.Error(t, err)
	})
}

func TestRedisSaveUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		user := User{ID: 1, Username: "test", Language: "en"}
		expected, err := json.Marshal(user)
		require.NoError(t, err)
		mock.ExpectSet("user:1", string(expected), 0).SetVal("OK")

		err = storage.SaveUser(user)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		user := User{ID: 1, Username: "test", Language: "en"}
		expected, err := json.Marshal(user)
		require.NoError(t, err)
		mock.ExpectSet("user:1", string(expected), 0).SetErr(errors.New("FAIL"))

		err = storage.SaveUser(user)
		assert.Error(t, err)
	})
}

func TestRedisSaveUserItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item := UserDictionaryItem{Word: "word", User: UserID(1), Created: time.Now()}
		expected, err := json.Marshal(item)
		require.NoError(t, err)
		mock.ExpectHSet("user_item:1", "word", string(expected)).SetVal(1)

		err = storage.SaveUserItem(item)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item := UserDictionaryItem{Word: "word", User: UserID(1), Created: time.Now()}
		expected, err := json.Marshal(item)
		require.NoError(t, err)
		mock.ExpectHSet("user_item:1", "word", string(expected)).SetErr(errors.New("FAIL"))

		err = storage.SaveUserItem(item)
		assert.Error(t, err)
	})
}

func TestRedisGetUserItem(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item := UserDictionaryItem{Word: "word", User: UserID(1), Created: time.Now().Truncate(time.Nanosecond)}
		expected, err := json.Marshal(item)
		require.NoError(t, err)
		mock.ExpectHGet("user_item:1", "word").SetVal(string(expected))

		result, err := storage.GetUserItem(item.User, item.Word)
		assert.NoError(t, err)
		assert.Equal(t, item, result)
	})
	t.Run("missing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item := UserDictionaryItem{Word: "word", User: UserID(1), Created: time.Now()}
		mock.ExpectHGet("user_item:1", "word").RedisNil()

		_, err := storage.GetUserItem(item.User, item.Word)
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item := UserDictionaryItem{Word: "word", User: UserID(1), Created: time.Now()}
		mock.ExpectHGet("user_item:1", "word").SetVal("NOT_JSON")

		_, err := storage.GetUserItem(item.User, item.Word)
		assert.Error(t, err)
	})
}

func TestRedisGetUserDictionary(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		item1 := DictionaryItem{Word: "test1"}
		item1Json, err := json.Marshal(item1)
		require.NoError(t, err)
		userItem1 := UserDictionaryItem{Word: item1.Word, User: UserID(1)}
		userItem1JSON, err := json.Marshal(userItem1)
		require.NoError(t, err)

		item2 := DictionaryItem{Word: "test2"}
		item2Json, err := json.Marshal(item2)
		require.NoError(t, err)
		userItem2 := UserDictionaryItem{Word: item2.Word, User: UserID(1)}
		userItem2JSON, err := json.Marshal(userItem2)
		require.NoError(t, err)

		mock.ExpectHGetAll("user_item:1").SetVal(map[string]string{
			item1.Word: string(userItem1JSON),
			item2.Word: string(userItem2JSON),
		})
		mock.Regexp().
			ExpectMGet(`word:test\d`, `word:test\d`).SetVal([]interface{}{string(item1Json), string(item2Json)})

		res, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Equal(t, map[UserDictionaryItem]DictionaryItem{userItem1: item1, userItem2: item2}, res)

	})
	t.Run("empty", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectHGetAll("user_item:1").RedisNil()

		items, err := storage.GetUserDictionary(UserID(1))
		assert.NoError(t, err)
		assert.Len(t, items, 0)
	})
}

func TestGetQuiz(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		quiz := Quiz{ID: "test", User: UserID(1), Language: "en"}
		expected, err := json.Marshal(quiz)
		require.NoError(t, err)
		mock.ExpectGet("quiz:test").SetVal(string(expected))

		res, err := storage.GetQuiz("test")
		assert.NoError(t, err)
		assert.Equal(t, quiz, res)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("quiz:test").RedisNil()

		_, err := storage.GetQuiz("test")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		mock.ExpectGet("quiz:test").SetVal("INVALID_JSON")

		_, err := storage.GetQuiz("test")
		assert.Error(t, err)
	})
}

func TestRedisSaveQuiz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		quiz := Quiz{ID: "test", User: UserID(1), Language: "en"}
		expected, err := json.Marshal(quiz)
		require.NoError(t, err)
		mock.ExpectSet("quiz:test", string(expected), 0).SetVal("OK")

		err = storage.SaveQuiz(quiz)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		storage := RedisStorage{db: db}
		quiz := Quiz{ID: "test", User: UserID(1), Language: "en"}
		expected, err := json.Marshal(quiz)
		require.NoError(t, err)
		mock.ExpectSet("quiz:test", string(expected), 0).SetErr(errors.New("FAIL"))

		err = storage.SaveQuiz(quiz)
		assert.Error(t, err)
	})
}
