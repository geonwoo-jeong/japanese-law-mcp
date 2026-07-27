package lawnamelexicon

type officialDataset struct {
	SchemaVersion int                `json:"schemaVersion"`
	DatasetID     string             `json:"datasetId"`
	Source        officialSource     `json:"source"`
	Statistics    officialStatistics `json:"statistics"`
	Entries       []officialEntry    `json:"entries"`
}

type officialSource struct {
	ProviderID  string `json:"providerId"`
	Operation   string `json:"operation"`
	URL         string `json:"url"`
	RetrievedAt string `json:"retrievedAt"`
}

type officialStatistics struct {
	LawCount   int `json:"lawCount"`
	AliasCount int `json:"aliasCount"`
}

type officialEntry struct {
	LawID      string    `json:"lawId"`
	RevisionID string    `json:"revisionId"`
	LawNumber  string    `json:"lawNumber"`
	Title      string    `json:"title"`
	TitleKana  string    `json:"titleKana,omitempty"`
	Aliases    *[]string `json:"aliases"`
}

type supplementalDataset struct {
	SchemaVersion int                 `json:"schemaVersion"`
	DatasetID     string              `json:"datasetId"`
	Entries       []supplementalEntry `json:"entries"`
}

type supplementalEntry struct {
	LawID       string `json:"lawId"`
	LawNumber   string `json:"lawNumber"`
	Title       string `json:"title"`
	Alias       string `json:"alias"`
	Kind        string `json:"kind"`
	SourceName  string `json:"sourceName"`
	SourceURL   string `json:"sourceUrl"`
	ConfirmedAt string `json:"confirmedAt"`
}
