package db

import "sync"

// InMemoryStorage is a storage implementation using in-memory maps
type InMemoryStorage struct {
	dictionary        map[string]DictionaryItem
	users             map[UserID]User
	quizzes           map[string]Quiz
	usersDictionaries map[UserID]map[string]UserDictionaryItem
	mx                sync.RWMutex
}

// Get returns dictionary item by word
func (d *InMemoryStorage) Get(word string) (DictionaryItem, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	item, ok := d.dictionary[word]
	if !ok {
		return DictionaryItem{}, ErrNotFound
	}
	return item, nil
}

// Save saves dictionary item
func (d *InMemoryStorage) Save(item DictionaryItem) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.dictionary[item.Word] = item
	return nil
}

// GetUser returns user by ID
func (d *InMemoryStorage) GetUser(id UserID) (User, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	item, ok := d.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return item, nil
}

// SaveUser saves user
func (d *InMemoryStorage) SaveUser(item User) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.users[item.ID] = item
	return nil
}

// GetUserItem returns item from user dictionary
func (d *InMemoryStorage) GetUserItem(user UserID, word string) (UserDictionaryItem, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	item, ok := d.usersDictionaries[user][word]
	if !ok {
		return UserDictionaryItem{}, ErrNotFound
	}
	return item, nil
}

// SaveUserItem saves UserDictionaryItem
func (d *InMemoryStorage) SaveUserItem(item UserDictionaryItem) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	userDict, ok := d.usersDictionaries[item.User]
	if !ok {
		userDict = make(map[string]UserDictionaryItem)
		d.usersDictionaries[item.User] = userDict
	}
	userDict[item.Word] = item
	return nil
}

// GetUserDictionary returns map of user dictionary items
func (d *InMemoryStorage) GetUserDictionary(user UserID) (map[UserDictionaryItem]DictionaryItem, error) {
	result := make(map[UserDictionaryItem]DictionaryItem)
	d.mx.RLock()
	defer d.mx.RUnlock()
	for _, item := range d.usersDictionaries[user] {
		result[item] = d.dictionary[item.Word]
	}
	return result, nil
}

// SaveQuiz saves quiz
func (d *InMemoryStorage) SaveQuiz(q Quiz) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.quizzes[q.ID] = q
	return nil
}

// GetQuiz returns quiz by ID
func (d *InMemoryStorage) GetQuiz(id string) (Quiz, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	q, ok := d.quizzes[id]
	if !ok {
		return Quiz{}, ErrNotFound
	}
	return q, nil
}

// NewInMemoryStorage creates new empty in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		users:             make(map[UserID]User),
		dictionary:        make(map[string]DictionaryItem),
		quizzes:           make(map[string]Quiz),
		usersDictionaries: make(map[UserID]map[string]UserDictionaryItem),
	}
}
