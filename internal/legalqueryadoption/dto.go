package legalqueryadoption

type pointerDocument struct {
	ArtifactKind  string `json:"artifactKind"`
	SchemaVersion int    `json:"schemaVersion"`
	AdoptionID    string `json:"adoptionId"`
}

type manifestDocument struct {
	ArtifactKind       string            `json:"artifactKind"`
	SchemaVersion      int               `json:"schemaVersion"`
	AdoptionID         string            `json:"adoptionId"`
	PreviousAdoptionID *string           `json:"previousAdoptionId,omitempty"`
	ProfileSetID       string            `json:"profileSetId"`
	ProfileSetVersion  string            `json:"profileSetVersion"`
	RankingVersion     string            `json:"rankingVersion"`
	CompositionVersion string            `json:"compositionVersion"`
	EvaluatorVersion   string            `json:"evaluatorVersion"`
	Profiles           []profileDocument `json:"profiles"`
	CorpusVersion      string            `json:"corpusVersion"`
	HoldoutDigest      string            `json:"holdoutDigest"`
	BaselineVersion    string            `json:"baselineVersion"`
	BaselineSHA256     string            `json:"baselineSha256"`
	CatalogVersion     string            `json:"catalogVersion"`
	CatalogSHA256      string            `json:"catalogSha256"`
}

type profileDocument struct {
	ProfileID      string `json:"profileId"`
	ProfileVersion string `json:"profileVersion"`
	CueSetVersion  string `json:"cueSetVersion"`
}

func manifestFromDocument(document manifestDocument) Manifest {
	profiles := make([]Profile, 0, len(document.Profiles))
	for _, profile := range document.Profiles {
		profiles = append(profiles, Profile{
			profileID:      profile.ProfileID,
			profileVersion: profile.ProfileVersion,
			cueSetVersion:  profile.CueSetVersion,
		})
	}
	previous := ""
	if document.PreviousAdoptionID != nil {
		previous = *document.PreviousAdoptionID
	}
	return Manifest{
		adoptionID:         document.AdoptionID,
		previousAdoptionID: previous,
		profileSetID:       document.ProfileSetID,
		profileSetVersion:  document.ProfileSetVersion,
		rankingVersion:     document.RankingVersion,
		compositionVersion: document.CompositionVersion,
		evaluatorVersion:   document.EvaluatorVersion,
		profiles:           profiles,
		corpusVersion:      document.CorpusVersion,
		holdoutDigest:      document.HoldoutDigest,
		baselineVersion:    document.BaselineVersion,
		baselineSHA256:     document.BaselineSHA256,
		catalogVersion:     document.CatalogVersion,
		catalogSHA256:      document.CatalogSHA256,
	}
}
