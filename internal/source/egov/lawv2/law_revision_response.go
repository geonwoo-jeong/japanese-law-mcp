package lawv2

import "encoding/json"

type lawRevisionResponse struct {
	lawID            string
	lawNumber        string
	promulgationDate string
	revisions        []lawRevisionRecord
}

type lawRevisionRecord struct {
	revisionID                string
	lawType                   string
	title                     string
	titleKana                 string
	abbreviation              string
	category                  string
	sourceUpdatedAt           string
	amendmentPromulgationDate string
	effectiveDate             string
	effectiveDateNote         string
	scheduledEffectiveDate    string
	amendmentLawID            string
	amendmentLawTitle         string
	amendmentLawTitleKana     string
	amendmentLawNumber        string
	amendmentType             string
	repealStatus              string
	repealRecordedDate        string
	remainInForce             *bool
	mission                   string
	currentStatus             string
}

type rawLawRevisionResponse struct {
	LawInfo   json.RawMessage `json:"law_info"`
	Revisions json.RawMessage `json:"revisions"`
}

type rawLawRevisionLawInfo struct {
	LawID            json.RawMessage `json:"law_id"`
	LawNumber        json.RawMessage `json:"law_num"`
	PromulgationDate json.RawMessage `json:"promulgation_date"`
}

type rawLawRevisionRecord struct {
	RevisionID                json.RawMessage `json:"law_revision_id"`
	LawType                   json.RawMessage `json:"law_type"`
	Title                     json.RawMessage `json:"law_title"`
	TitleKana                 json.RawMessage `json:"law_title_kana"`
	Abbreviation              json.RawMessage `json:"abbrev"`
	Category                  json.RawMessage `json:"category"`
	SourceUpdatedAt           json.RawMessage `json:"updated"`
	AmendmentPromulgationDate json.RawMessage `json:"amendment_promulgate_date"`
	EffectiveDate             json.RawMessage `json:"amendment_enforcement_date"`
	EffectiveDateNote         json.RawMessage `json:"amendment_enforcement_comment"`
	ScheduledEffectiveDate    json.RawMessage `json:"amendment_scheduled_enforcement_date"`
	AmendmentLawID            json.RawMessage `json:"amendment_law_id"`
	AmendmentLawTitle         json.RawMessage `json:"amendment_law_title"`
	AmendmentLawTitleKana     json.RawMessage `json:"amendment_law_title_kana"`
	AmendmentLawNumber        json.RawMessage `json:"amendment_law_num"`
	AmendmentType             json.RawMessage `json:"amendment_type"`
	RepealStatus              json.RawMessage `json:"repeal_status"`
	RepealRecordedDate        json.RawMessage `json:"repeal_date"`
	RemainInForce             json.RawMessage `json:"remain_in_force"`
	Mission                   json.RawMessage `json:"mission"`
	CurrentStatus             json.RawMessage `json:"current_revision_status"`
}
