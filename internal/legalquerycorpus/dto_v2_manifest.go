package legalquerycorpus

import "fmt"

type manifestV2DTO struct {
	ArtifactKind                    *string            `json:"artifactKind"`
	SchemaVersion                   *int               `json:"schemaVersion"`
	CorpusVersion                   *string            `json:"corpusVersion"`
	Seed                            *int               `json:"seed"`
	HoldoutDigest                   *string            `json:"holdoutDigest"`
	HoldoutLeakageGroupDigests      *[]string          `json:"holdoutLeakageGroupDigests"`
	RequiredCategoryIDs             *[]string          `json:"requiredCategoryIds"`
	RequiredExecutionScenarioIDs    *[]string          `json:"requiredExecutionScenarioIds"`
	RequiredDevelopmentAssertionIDs *[]string          `json:"requiredDevelopmentAssertionIds"`
	Sets                            *manifestSetsV1DTO `json:"sets"`
}

func decodeManifestV2(data []byte) (Manifest, error) {
	var dto manifestV2DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return Manifest{}, err
	}
	return dto.manifest()
}

func (d manifestV2DTO) manifest() (Manifest, error) {
	if d.ArtifactKind == nil ||
		d.SchemaVersion == nil ||
		d.CorpusVersion == nil ||
		d.Seed == nil ||
		d.HoldoutDigest == nil ||
		d.HoldoutLeakageGroupDigests == nil ||
		d.RequiredCategoryIDs == nil ||
		d.RequiredExecutionScenarioIDs == nil ||
		d.RequiredDevelopmentAssertionIDs == nil ||
		d.Sets == nil {
		return Manifest{}, fmt.Errorf("manifest v2 の必須項目が不足しています")
	}
	development, err := convertManifestSetV1(
		ManifestSetDevelopment,
		d.Sets.Development,
	)
	if err != nil {
		return Manifest{}, err
	}
	holdout, err := convertManifestSetV1(ManifestSetHoldout, d.Sets.Holdout)
	if err != nil {
		return Manifest{}, err
	}
	execution, err := convertManifestSetV1(ManifestSetExecution, d.Sets.Execution)
	if err != nil {
		return Manifest{}, err
	}
	return NewManifest(ManifestValues{
		ArtifactKind:                    ArtifactKind(*d.ArtifactKind),
		SchemaVersion:                   *d.SchemaVersion,
		CorpusVersion:                   *d.CorpusVersion,
		Seed:                            *d.Seed,
		HoldoutDigest:                   *d.HoldoutDigest,
		HoldoutLeakageGroupDigests:      *d.HoldoutLeakageGroupDigests,
		RequiredCategoryIDs:             *d.RequiredCategoryIDs,
		RequiredExecutionScenarioIDs:    *d.RequiredExecutionScenarioIDs,
		RequiredDevelopmentAssertionIDs: *d.RequiredDevelopmentAssertionIDs,
		Development:                     development,
		Holdout:                         holdout,
		Execution:                       execution,
	})
}
