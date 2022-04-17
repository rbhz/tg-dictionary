package mymemory

type TranslationResult struct {
	Text  string  `json:"translatedText"`
	Match float64 `json:"match"`
}

type TranslationMatch struct {
	ID          string  `json:"id"`
	Segment     string  `json:"segment"`
	Translation string  `json:"translation"`
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Quality     string  `json:"quality"`
	Reference   *string `json:"reference"`
	UsageCount  int     `json:"usage-count"`
	Subject     string  `json:"subject"`
	Match       float64 `json:"match"`
}

type TranslationResponse struct {
	Result          TranslationResult  `json:"responseData"`
	Matches         []TranslationMatch `json:"matches"`
	QuotaFinished   bool               `json:"quotaFinished"`
	ResponseDetails string             `json:"responseDetails"`
	ResponseStatus  int                `json:"responseStatus"`
	ResponderID     string             `json:"responderId"`
	ExceptionCode   *string            `json:"exception_code"`
}
