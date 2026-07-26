package lawv2

import "encoding/json"

type lawSearchResponse struct {
	totalCount int
	count      int
	laws       []lawSearchLaw
}

type lawSearchLaw struct {
	lawID                 string
	revisionID            string
	title                 string
	lawNumber             string
	promulgationDate      string
	revisionEffectiveDate string
}

type rawLawSearchResponse struct {
	TotalCount json.RawMessage `json:"total_count"`
	Count      json.RawMessage `json:"count"`
	NextOffset json.RawMessage `json:"next_offset"`
	Laws       json.RawMessage `json:"laws"`
}

type rawLawSearchLaw struct {
	LawInfo      json.RawMessage `json:"law_info"`
	RevisionInfo json.RawMessage `json:"revision_info"`
}

type rawLawInfo struct {
	LawID            json.RawMessage `json:"law_id"`
	LawNumber        json.RawMessage `json:"law_num"`
	PromulgationDate json.RawMessage `json:"promulgation_date"`
}

type rawRevisionInfo struct {
	RevisionID            json.RawMessage `json:"law_revision_id"`
	Title                 json.RawMessage `json:"law_title"`
	RevisionEffectiveDate json.RawMessage `json:"amendment_enforcement_date"`
}

type responseNextOffset struct {
	present bool
	isNull  bool
	value   int
}
