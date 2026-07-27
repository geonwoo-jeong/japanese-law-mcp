package legalquerycorpus

import "fmt"

type manifestEntryV1DTO struct {
	CaseID *string `json:"caseId"`
	SHA256 *string `json:"sha256"`
}

type manifestSetV1DTO struct {
	CaseCount *int                  `json:"caseCount"`
	Cases     *[]manifestEntryV1DTO `json:"cases"`
}

type manifestSetsV1DTO struct {
	Development *manifestSetV1DTO `json:"development"`
	Holdout     *manifestSetV1DTO `json:"holdout"`
	Execution   *manifestSetV1DTO `json:"execution"`
}

type manifestV1DTO struct {
	ArtifactKind                 *string            `json:"artifactKind"`
	SchemaVersion                *int               `json:"schemaVersion"`
	CorpusVersion                *string            `json:"corpusVersion"`
	Seed                         *int               `json:"seed"`
	HoldoutDigest                *string            `json:"holdoutDigest"`
	RequiredCategoryIDs          *[]string          `json:"requiredCategoryIds"`
	RequiredExecutionScenarioIDs *[]string          `json:"requiredExecutionScenarioIds"`
	Sets                         *manifestSetsV1DTO `json:"sets"`
}

func decodeManifestV1(data []byte) (Manifest, error) {
	var dto manifestV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return Manifest{}, err
	}
	return dto.manifest()
}

func (d manifestV1DTO) manifest() (Manifest, error) {
	if d.ArtifactKind == nil ||
		d.SchemaVersion == nil ||
		d.CorpusVersion == nil ||
		d.Seed == nil ||
		d.HoldoutDigest == nil ||
		d.RequiredCategoryIDs == nil ||
		d.RequiredExecutionScenarioIDs == nil ||
		d.Sets == nil {
		return Manifest{}, fmt.Errorf("manifest の必須項目が不足しています")
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
		ArtifactKind:                 ArtifactKind(*d.ArtifactKind),
		SchemaVersion:                *d.SchemaVersion,
		CorpusVersion:                *d.CorpusVersion,
		Seed:                         *d.Seed,
		HoldoutDigest:                *d.HoldoutDigest,
		RequiredCategoryIDs:          *d.RequiredCategoryIDs,
		RequiredExecutionScenarioIDs: *d.RequiredExecutionScenarioIDs,
		Development:                  development,
		Holdout:                      holdout,
		Execution:                    execution,
	})
}

func convertManifestSetV1(
	kind ManifestSetKind,
	dto *manifestSetV1DTO,
) (ManifestSet, error) {
	if dto == nil || dto.CaseCount == nil || dto.Cases == nil {
		return ManifestSet{}, fmt.Errorf("manifest set の必須項目が不足しています")
	}
	entries := make([]ManifestEntry, 0, len(*dto.Cases))
	for _, entryDTO := range *dto.Cases {
		if entryDTO.CaseID == nil || entryDTO.SHA256 == nil {
			return ManifestSet{}, fmt.Errorf("manifest set の entry に必須項目がありません")
		}
		entry, err := NewManifestEntry(ManifestEntryValues{
			CaseID: *entryDTO.CaseID,
			SHA256: *entryDTO.SHA256,
		})
		if err != nil {
			return ManifestSet{}, fmt.Errorf("manifest set の entry が有効ではありません: %w", err)
		}
		entries = append(entries, entry)
	}
	return NewManifestSet(ManifestSetValues{
		Kind:      kind,
		CaseCount: *dto.CaseCount,
		Cases:     entries,
	})
}
