package api

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/rbhz/tg-dictionary/app/db"
)

const (
	testTGToken   = "123123213:1231231312"
	testJWTSecret = "tokentokentokentoken"
	testUserID    = 1
)

// emptyHandler is a dummy handler for testing.
type emptyHandler struct{}

func (h *emptyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

// ErrorStorage is a dummy storage for testing storage error handling.
type ErrorStorage struct {
	*db.InMemoryStorage
}

func (d ErrorStorage) GetUserDictionary(db.UserID) (map[db.UserDictionaryItem]db.DictionaryItem, error) {
	return nil, errors.New("test")
}

func (d ErrorStorage) Get(string) (db.DictionaryItem, error) {
	return db.DictionaryItem{}, errors.New("test")
}

func (d ErrorStorage) Save(item db.DictionaryItem) error {
	return errors.New("test")

}

// getTestServer returns a test server.
func getTestServer(storage db.Storage) (*httptest.Server, func()) {
	if storage == nil {
		storage = db.NewInMemoryStorage()
	}

	server := NewServer(storage, testTGToken, testJWTSecret)
	srv := httptest.NewServer(server.router)
	return srv, srv.Close
}

// getTestJWT returns a test JWT signed with testJWTSecret
func getTestJWT() string {
	token, _ := (&authService{telegramToken: testTGToken, jwtSecret: []byte(testJWTSecret)}).createToken(testUserID)
	return "Bearer " + token
}
