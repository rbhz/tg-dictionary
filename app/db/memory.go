package db

import "sync"

type InMemoryStorage struct {
	dictionary        map[string]DictionaryItem
	quizzes           map[string]Quiz
	UsersDictionaries map[UserID]map[string]UserDictionaryItem
	mx                sync.RWMutex
}

func (d *InMemoryStorage) Get(word string) (DictionaryItem, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	item, ok := d.dictionary[word]
	if !ok {
		return DictionaryItem{}, ErrNotFound
	}
	return item, nil
}

func (d *InMemoryStorage) Save(item DictionaryItem) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.dictionary[item.Word] = item
	return nil
}

func (d *InMemoryStorage) GetUserItem(user UserID, word string) (UserDictionaryItem, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	item, ok := d.UsersDictionaries[user][word]
	if !ok {
		return UserDictionaryItem{}, ErrNotFound
	}
	return item, nil
}

func (d *InMemoryStorage) SaveUserItem(item UserDictionaryItem) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	userDict, ok := d.UsersDictionaries[item.User]
	if !ok {
		userDict = make(map[string]UserDictionaryItem)
		d.UsersDictionaries[item.User] = userDict
	}
	userDict[item.Word] = item
	return nil
}

func (d *InMemoryStorage) GetUserDictionary(user UserID) (map[UserDictionaryItem]DictionaryItem, error) {
	result := make(map[UserDictionaryItem]DictionaryItem)
	d.mx.RLock()
	defer d.mx.RUnlock()
	for _, item := range d.UsersDictionaries[user] {
		result[item] = d.dictionary[item.Word]
	}
	return result, nil
}

func (d *InMemoryStorage) SaveQuiz(q Quiz) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.quizzes[q.ID] = q
	return nil
}

func (d *InMemoryStorage) GetQuiz(id string) (Quiz, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	q, ok := d.quizzes[id]
	if !ok {
		return Quiz{}, ErrNotFound
	}
	return q, nil
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		dictionary:        make(map[string]DictionaryItem),
		quizzes:           make(map[string]Quiz),
		UsersDictionaries: make(map[UserID]map[string]UserDictionaryItem),
	}
}
