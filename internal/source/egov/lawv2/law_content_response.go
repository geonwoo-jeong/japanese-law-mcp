package lawv2

import "encoding/json"

type lawContentSearchResponse struct {
	totalCount    int
	sentenceCount int
	items         []lawContentSearchLaw
}

type lawContentSearchLaw struct {
	law       lawSearchLaw
	sentences []lawContentSentence
}

type lawContentSentence struct {
	position string
	text     string
}

type rawLawContentSearchResponse struct {
	TotalCount    json.RawMessage `json:"total_count"`
	SentenceCount json.RawMessage `json:"sentence_count"`
	NextOffset    json.RawMessage `json:"next_offset"`
	Items         json.RawMessage `json:"items"`
}

type rawLawContentSearchLaw struct {
	LawInfo      json.RawMessage `json:"law_info"`
	RevisionInfo json.RawMessage `json:"revision_info"`
	Sentences    json.RawMessage `json:"sentences"`
}

type rawLawContentSentence struct {
	Position json.RawMessage `json:"position"`
	Text     json.RawMessage `json:"text"`
}
