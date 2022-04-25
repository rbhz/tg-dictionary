package ya_dictionary

type TranslationResponse struct {
	Definitions []Definition `json:"def"`
}

type Definition struct {
	Text         string        `json:"text"`
	PartOfSpeech string        `json:"pos"`
	Translations []Translation `json:"tr"`
}

type Translation struct {
	Text         string     `json:"text"`
	PartOfSpeech string     `json:"pos"`
	Examples     []Example  `json:"ex"`
	Synonyms     []textItem `json:"syn"`
	Meanings     []textItem `json:"mean"`
}

type Example struct {
	Text         string     `json:"text"`
	Translations []textItem `json:"tr"`
}

type textItem struct {
	Text string `json:"text"`
}
