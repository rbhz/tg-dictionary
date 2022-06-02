package yandexdictionary

// TranslationResponse Describes API response
type TranslationResponse struct {
	Definitions []Definition `json:"def"`
}

// Definition holds definition data in response
type Definition struct {
	Text         string        `json:"text"`
	PartOfSpeech string        `json:"pos"`
	Translations []Translation `json:"tr"`
}

// Translation holds translation data in response
type Translation struct {
	Text         string     `json:"text"`
	PartOfSpeech string     `json:"pos"`
	Examples     []Example  `json:"ex"`
	Synonyms     []textItem `json:"syn"`
	Meanings     []textItem `json:"mean"`
}

// Example holds example data in response
type Example struct {
	Text         string     `json:"text"`
	Translations []textItem `json:"tr"`
}

type textItem struct {
	Text string `json:"text"`
}
