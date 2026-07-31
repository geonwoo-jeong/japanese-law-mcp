package legalquerycorpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// integrityCheckedCorpus は、manifest と原 byte の整合を確認した三集合を保持する。
type integrityCheckedCorpus struct {
	manifest    Manifest
	development []SemanticCase
	holdout     []SemanticCase
	execution   []ExecutionCase
}

type manifestIntegritySet struct {
	kind ManifestSetKind
	set  ManifestSet
}

type fixtureDigestEntry struct {
	caseID string
	sha256 string
}

// validateManifestIntegrity は、成果物と集合横断要件を順に検証する。
func validateManifestIntegrity(
	ctx context.Context,
	filesystem *corpusFilesystem,
	schema corpusSchemaV1,
	manifest Manifest,
) (integrityCheckedCorpus, error) {
	checked, err := validateManifestArtifacts(ctx, filesystem, schema, manifest)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	return validateIntegrityRequirements(checked)
}

func validateIntegrityRequirements(
	checked integrityCheckedCorpus,
) (integrityCheckedCorpus, error) {
	checked, err := validateIntegrityDevelopmentAssertions(checked)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	checked, err = validateIntegrityHoldoutRequirements(checked)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	checked, err = validateIntegrityExecutionReferences(checked)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	return validateIntegrityExecutionScenarioRequirements(checked)
}

// validateManifestArtifacts は、manifest 宣言と実在 fixture を byte 単位で照合する。
func validateManifestArtifacts(
	ctx context.Context,
	filesystem *corpusFilesystem,
	schema corpusSchemaV1,
	manifest Manifest,
) (integrityCheckedCorpus, error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return integrityCheckedCorpus{}, err
	}
	if filesystem == nil || schema.resolved == nil {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"manifest integrity の入力が初期化されていません",
		)
	}
	if err := manifest.Validate(); err != nil {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"manifest integrity の manifest が有効ではありません: %w",
			err,
		)
	}
	if schema.version != manifest.SchemaVersion() {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"manifest と schema の version が一致しません",
		)
	}
	if manifest.CorpusVersion() != filesystem.corpusVersion {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"manifest の corpusVersion が directory と一致しません",
		)
	}
	sets := manifestIntegritySets(manifest)
	if err := validateManifestFileSets(filesystem, sets); err != nil {
		return integrityCheckedCorpus{}, err
	}
	if err := validateManifestGlobalUniqueness(sets); err != nil {
		return integrityCheckedCorpus{}, err
	}

	reader := filesystem.newFixtureReader()
	development, _, err := loadSemanticIntegritySet(
		ctx,
		reader,
		schema,
		manifest.SchemaVersion(),
		manifest.Development(),
	)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	holdout, holdoutDigests, err := loadSemanticIntegritySet(
		ctx,
		reader,
		schema,
		manifest.SchemaVersion(),
		manifest.Holdout(),
	)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	execution, err := loadExecutionIntegritySet(
		ctx,
		reader,
		schema,
		manifest.SchemaVersion(),
		manifest.Execution(),
	)
	if err != nil {
		return integrityCheckedCorpus{}, err
	}
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return integrityCheckedCorpus{}, err
	}
	if manifest.HoldoutDigest() != computeFixtureDigest(holdoutDigests) {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"manifest の holdoutDigest が実在 fixture と一致しません",
		)
	}
	if err := validateManifestHoldoutLeakageGroupDigests(
		manifest,
		holdout,
	); err != nil {
		return integrityCheckedCorpus{}, err
	}
	if err := validateSemanticSetSeparation(development, holdout); err != nil {
		return integrityCheckedCorpus{}, err
	}
	clonedManifest, err := cloneCorpusManifest(manifest)
	if err != nil {
		return integrityCheckedCorpus{}, fmt.Errorf(
			"検証済み manifest を複製できません",
		)
	}
	return integrityCheckedCorpus{
		manifest:    clonedManifest,
		development: development,
		holdout:     holdout,
		execution:   execution,
	}, nil
}

func manifestIntegritySets(manifest Manifest) []manifestIntegritySet {
	return []manifestIntegritySet{
		{kind: ManifestSetDevelopment, set: manifest.Development()},
		{kind: ManifestSetHoldout, set: manifest.Holdout()},
		{kind: ManifestSetExecution, set: manifest.Execution()},
	}
}

func validateManifestFileSets(
	filesystem *corpusFilesystem,
	sets []manifestIntegritySet,
) error {
	for _, candidate := range sets {
		names, err := filesystem.fixtureFileNames(candidate.kind)
		if err != nil {
			return err
		}
		entries := candidate.set.Cases()
		if len(names) != len(entries) {
			return fmt.Errorf(
				"manifest の fixture 件数が実在 file 集合と一致しません",
			)
		}
		for index, entry := range entries {
			if names[index] != entry.CaseID()+".json" {
				return fmt.Errorf(
					"manifest の fixture 名と実在 file 名が一致しません",
				)
			}
		}
	}
	return nil
}

func validateManifestGlobalUniqueness(sets []manifestIntegritySet) error {
	caseIDs := make(map[string]struct{})
	checksums := make(map[string]struct{})
	for _, candidate := range sets {
		for _, entry := range candidate.set.Cases() {
			if _, exists := caseIDs[entry.CaseID()]; exists {
				return fmt.Errorf(
					"manifest の caseId は全集合で一意でなければなりません",
				)
			}
			if _, exists := checksums[entry.SHA256()]; exists {
				return fmt.Errorf(
					"manifest の checksum は全集合で一意でなければなりません",
				)
			}
			caseIDs[entry.CaseID()] = struct{}{}
			checksums[entry.SHA256()] = struct{}{}
		}
	}
	return nil
}

func computeFixtureDigest(entries []fixtureDigestEntry) string {
	var input strings.Builder
	for _, entry := range entries {
		input.WriteString(entry.caseID)
		input.WriteByte(' ')
		input.WriteString(entry.sha256)
		input.WriteByte('\n')
	}
	return sha256Hex([]byte(input.String()))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func computeHoldoutLeakageGroupDigests(cases []SemanticCase) []string {
	unique := make(map[string]struct{}, len(cases))
	for _, semanticCase := range cases {
		input := "legal-query-leakage-group-v1\x00" +
			semanticCase.LeakageGroupID()
		unique[sha256Hex([]byte(input))] = struct{}{}
	}
	digests := make([]string, 0, len(unique))
	for digest := range unique {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	return digests
}

func validateManifestHoldoutLeakageGroupDigests(
	manifest Manifest,
	holdout []SemanticCase,
) error {
	if manifest.SchemaVersion() == corpusSchemaVersionV1 {
		return nil
	}
	if !equalStringSequence(
		manifest.HoldoutLeakageGroupDigests(),
		computeHoldoutLeakageGroupDigests(holdout),
	) {
		return fmt.Errorf(
			"manifest の holdout leakage digest 集合が実在 fixture と一致しません",
		)
	}
	return nil
}

func validateIntegrityDevelopmentAssertions(
	checked integrityCheckedCorpus,
) (integrityCheckedCorpus, error) {
	if checked.manifest.SchemaVersion() == corpusSchemaVersionV1 {
		return checked, nil
	}
	present := make(map[string]struct{})
	for _, semanticCase := range checked.development {
		for _, assertionID := range semanticCase.DevelopmentAssertionIDs() {
			present[assertionID] = struct{}{}
		}
	}
	for _, required := range checked.manifest.RequiredDevelopmentAssertionIDs() {
		if _, exists := present[required]; !exists {
			return integrityCheckedCorpus{}, fmt.Errorf(
				"development assertion の必須 ID が不足しています",
			)
		}
	}
	return checked, nil
}
