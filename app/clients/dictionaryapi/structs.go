package dictionaryapi

// WordResponse holds information for an API response
type WordResponse struct {
	Word      string     `json:"word"`
	Phonetic  string     `json:"phonetic"`
	Phonetics []Phonetic `json:"phonetics"`
	Origin    string     `json:"origin"`
	Meanings  []Meaning  `json:"meanings"`
}

// Phonetic holds information for a phonetic pronunciation
type Phonetic struct {
	Text  string  `json:"text"`
	Audio *string `json:"Audio"`
}

// Meaning holds information for a meaning
type Meaning struct {
	PartOfSpeech string       `json:"partOfSpeech"`
	Definitions  []Definition `json:"definitions"`
}

// Definition holds information for a definition
type Definition struct {
	Definition string   `json:"definition"`
	Example    string   `json:"example"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}
