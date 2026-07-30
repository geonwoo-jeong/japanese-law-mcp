package legalconceptlexicon

type dataset struct {
	SchemaVersion  int            `json:"schemaVersion"`
	LexiconVersion string         `json:"lexiconVersion"`
	GeneratedAt    string         `json:"generatedAt"`
	Entries        []datasetEntry `json:"entries"`
}

type datasetEntry struct {
	ConceptID       string             `json:"conceptId"`
	Canonical       string             `json:"canonical"`
	Terms           []string           `json:"terms"`
	ComparisonTerms []string           `json:"comparisonTerms"`
	SourceName      string             `json:"sourceName"`
	SourceURL       string             `json:"sourceUrl"`
	ConfirmedAt     string             `json:"confirmedAt"`
	MappingNote     string             `json:"mappingNote"`
	ConflictGroupID string             `json:"conflictGroupId,omitempty"`
	SelectionPolicy SelectionPolicy    `json:"selectionPolicy"`
	Candidates      []datasetCandidate `json:"candidates"`
}

type datasetCandidate struct {
	Task                  string                        `json:"task"`
	Resource              string                        `json:"resource"`
	InputKind             string                        `json:"inputKind"`
	OfficialTerm          string                        `json:"officialTerm"`
	TermOfficialOverrides []datasetTermOfficialOverride `json:"termOfficialOverrides,omitempty"`
	RequiredPacks         *[]string                     `json:"requiredPacks"`
}

type datasetTermOfficialOverride struct {
	Term         string `json:"term"`
	OfficialTerm string `json:"officialTerm"`
}
