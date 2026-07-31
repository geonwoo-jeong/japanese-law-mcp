package legalquerycandidateeval

import (
	"bytes"
	"context"
	"testing"
)

const (
	verificationClosedArtifacts   = "candidate-evaluation-closed-artifacts"
	verificationReviewAttestation = "candidate-evaluation-review-attestation"
)

func TestCanonicalIdentityは各文書の意味Tupleへ結合する(t *testing.T) {
	t.Parallel()

	manifest := validCandidateManifest(t)
	originalID := manifest.CandidateContentID
	manifest.ProfileSet.RankingVersion = "ranking-v2"
	changedID, err := CanonicalCandidateContentID(manifest)
	if err != nil {
		t.Fatalf("candidateContentId を計算できません: %v", err)
	}
	if changedID == originalID {
		t.Fatal("rankingVersion の変更で candidateContentId が変わりません")
	}

	attestation := validReviewAttestation(t, manifestWithID(t), ReviewScopeArchitecture, "authority-a")
	attestationID := attestation.AttestationID
	attestation.MinorCount++
	changedAttestationID, err := CanonicalReviewAttestationID(attestation)
	if err != nil {
		t.Fatalf("attestationId を計算できません: %v", err)
	}
	if changedAttestationID == attestationID {
		t.Fatal("minorCount の変更で attestationId が変わりません")
	}

	request := validEvaluationRequest(t, manifestWithID(t))
	evaluationID := request.EvaluationID
	request.BaselineVersion = "default-3"
	changedEvaluationID, err := CanonicalEvaluationID(request)
	if err != nil {
		t.Fatalf("evaluationId を計算できません: %v", err)
	}
	if changedEvaluationID == evaluationID {
		t.Fatal("baselineVersion の変更で evaluationId が変わりません")
	}
}

func TestDecodeCandidateContentはclosedかつcanonicalな文書だけを受理する(t *testing.T) {
	t.Parallel()

	manifest := manifestWithID(t)
	raw := mustCanonicalJSON(t, manifest)
	decoded, err := DecodeCandidateContentManifest(raw)
	if err != nil {
		t.Fatalf("%s: 正しい manifest を拒否しました: %v", verificationClosedArtifacts, err)
	}
	if decoded.CandidateContentID != manifest.CandidateContentID {
		t.Fatalf("candidateContentId = %q", decoded.CandidateContentID)
	}

	nonCanonical := append([]byte{' '}, raw...)
	if _, err := DecodeCandidateContentManifest(nonCanonical); err == nil {
		t.Fatal("非 canonical byte を受理しました")
	}
	unknown := bytes.Replace(raw, []byte("\"schemaVersion\":2"), []byte("\"schemaVersion\":2,\"unknown\":1"), 1)
	if _, err := DecodeCandidateContentManifest(unknown); err == nil {
		t.Fatal("未知 field を受理しました")
	}
	duplicate := bytes.Replace(raw, []byte("\"schemaVersion\":2"), []byte("\"schemaVersion\":2,\"schemaVersion\":2"), 1)
	if _, err := DecodeCandidateContentManifest(duplicate); err == nil {
		t.Fatal("重複 key を受理しました")
	}
}

func TestReviewAttestationはrubricとscope別得点を検証する(t *testing.T) {
	t.Parallel()

	manifest := manifestWithID(t)
	attestation := validReviewAttestation(t, manifest, ReviewScopeArchitecture, "authority-a")
	if _, err := DecodeReviewAttestation(mustCanonicalJSON(t, attestation)); err != nil {
		t.Fatalf("%s: 正しい attestation を拒否しました: %v", verificationReviewAttestation, err)
	}

	attestation.CriterionScores[0].Score20 = 10
	attestation.Score100 = 90
	attestation.AttestationID = mustReviewID(t, attestation)
	if _, err := DecodeReviewAttestation(mustCanonicalJSON(t, attestation)); err == nil {
		t.Fatal("16 未満の criterion を approved として受理しました")
	}
}

func TestEvaluationRequestは固定SOT集合とreview順を検証する(t *testing.T) {
	t.Parallel()

	manifest := manifestWithID(t)
	request := validEvaluationRequest(t, manifest)
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, request)); err != nil {
		t.Fatalf("正しい request を拒否しました: %v", err)
	}

	request.ReviewAttestations[0], request.ReviewAttestations[1] =
		request.ReviewAttestations[1], request.ReviewAttestations[0]
	request.EvaluationID = mustEvaluationID(t, request)
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, request)); err == nil {
		t.Fatal("review scope の逆順を受理しました")
	}
}

func TestSchemaV2は内部参照だけを許す(t *testing.T) {
	t.Parallel()

	if _, err := ParseSchemaV2([]byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","$ref":"https://example.invalid/schema"}`)); err == nil {
		t.Fatal("外部 $ref を受理しました")
	}
	schema, err := ParseSchemaV2(testSchemaV2())
	if err != nil {
		t.Fatalf("内部だけの schema を拒否しました: %v", err)
	}
	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV2,
		EvaluationID:  "evaluation-sha256-" + repeatHex('a'),
	}
	if err := schema.Validate(context.Background(), mustCanonicalJSON(t, pointer)); err != nil {
		t.Fatalf("pointer schema 検証に失敗しました: %v", err)
	}
}

func TestCandidateContentは閉じた境界条件を拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*CandidateContentManifest)
	}{
		{
			name: "profile-order",
			mutate: func(value *CandidateContentManifest) {
				value.ProfileArtifacts[0], value.ProfileArtifacts[1] = value.ProfileArtifacts[1], value.ProfileArtifacts[0]
			},
		},
		{
			name: "lexicon-aggregate",
			mutate: func(value *CandidateContentManifest) {
				value.LexiconArtifacts[0].AggregateSHA256 = repeatHex('0')
			},
		},
		{
			name: "composition-binding",
			mutate: func(value *CandidateContentManifest) {
				value.Composition.ProfileSetID = "different"
				value.Composition.DescriptorSHA256 = mustCompositionDigest(t, value.Composition)
			},
		},
		{
			name: "build-context",
			mutate: func(value *CandidateContentManifest) {
				value.SemanticSourceSet.GOOS = "darwin"
				value.SemanticSourceSet.SourceSetSHA256 = mustSourceSetDigest(t, value.SemanticSourceSet)
			},
		},
		{
			name: "source-path",
			mutate: func(value *CandidateContentManifest) {
				value.SemanticSourceSet.Files[0].Path = "../outside.go"
				value.SemanticSourceSet.SourceSetSHA256 = mustSourceSetDigest(t, value.SemanticSourceSet)
			},
		},
		{
			name: "module-sum",
			mutate: func(value *CandidateContentManifest) {
				value.SemanticSourceSet.ModuleDependencies = []ModuleDependency{validModuleDependency()}
				value.SemanticSourceSet.ModuleDependencies[0].ModuleZipSum = "invalid"
				value.SemanticSourceSet.SourceSetSHA256 = mustSourceSetDigest(t, value.SemanticSourceSet)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := manifestWithID(t)
			test.mutate(&document)
			document.CandidateContentID = mustCandidateID(t, document)
			if _, err := DecodeCandidateContentManifest(mustCanonicalJSON(t, document)); err == nil {
				t.Fatal("不正な candidate content を受理しました")
			}
		})
	}
}

func TestCandidateContentは外部ModuleIdentityを保持できる(t *testing.T) {
	t.Parallel()

	document := manifestWithID(t)
	document.SemanticSourceSet.ModuleDependencies = []ModuleDependency{validModuleDependency()}
	document.SemanticSourceSet.SourceSetSHA256 = mustSourceSetDigest(t, document.SemanticSourceSet)
	document.CandidateContentID = mustCandidateID(t, document)
	if _, err := DecodeCandidateContentManifest(mustCanonicalJSON(t, document)); err != nil {
		t.Fatalf("正しい module dependency を拒否しました: %v", err)
	}
}

func TestDecodeCandidateContentは固定Schemaの正規形も検証する(t *testing.T) {
	t.Parallel()

	document := manifestWithID(t)
	document.SemanticSourceSet.GoLanguageVersion = "go1.25"
	document.SemanticSourceSet.SourceSetSHA256 = mustSourceSetDigest(t, document.SemanticSourceSet)
	document.CandidateContentID = mustCandidateID(t, document)
	if _, err := DecodeCandidateContentManifest(mustCanonicalJSON(t, document)); err == nil {
		t.Fatal("schema v2 が許さない goLanguageVersion を受理しました")
	}
}

func TestReviewとRequestは内容Digestの不一致を拒否する(t *testing.T) {
	t.Parallel()

	manifest := manifestWithID(t)
	attestation := validReviewAttestation(t, manifest, ReviewScopeArchitecture, "authority-a")
	attestation.ReviewedSOTSetSHA256 = repeatHex('0')
	attestation.AttestationID = mustReviewID(t, attestation)
	if _, err := DecodeReviewAttestation(mustCanonicalJSON(t, attestation)); err == nil {
		t.Fatal("reviewed SOT 集合 digest の不一致を受理しました")
	}

	request := validEvaluationRequest(t, manifest)
	request.HoldoutLeakageGroupDigests[0], request.HoldoutLeakageGroupDigests[1] =
		request.HoldoutLeakageGroupDigests[1], request.HoldoutLeakageGroupDigests[0]
	request.EvaluationID = mustEvaluationID(t, request)
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, request)); err == nil {
		t.Fatal("holdout leakage digest の逆順を受理しました")
	}
}

func validModuleDependency() ModuleDependency {
	return ModuleDependency{
		ModulePath:               "example.com/dependency",
		Version:                  "v1.2.3",
		ModuleZipSum:             "h1:" + string(bytes.Repeat([]byte{'A'}, 43)) + "=",
		ModuleZipRawSHA256:       repeatHex('d'),
		ModuleZipByteLength:      1024,
		ModuleZipEntryCount:      4,
		ModuleExpandedByteLength: 2048,
		ModuleGoModSum:           "h1:" + string(bytes.Repeat([]byte{'B'}, 43)) + "=",
		ModuleGoModRawSHA256:     repeatHex('e'),
	}
}

func validCandidateManifest(t *testing.T) CandidateContentManifest {
	t.Helper()

	lexicons := []LexiconArtifact{
		{
			LexiconID:      "lawNames",
			LexiconVersion: "law-names-v1",
			Files: []FileDigest{
				{Path: "internal/lawnamelexicon/data/egov-current.json", RawSHA256: repeatHex('1')},
				{Path: "internal/lawnamelexicon/data/supplemental.json", RawSHA256: repeatHex('2')},
			},
		},
		{
			LexiconID:      "legalConcepts",
			LexiconVersion: "legal-concepts-v1",
			Files: []FileDigest{
				{Path: "internal/legalconceptlexicon/data/current.json", RawSHA256: repeatHex('3')},
			},
		},
	}
	for index := range lexicons {
		lexicons[index].AggregateSHA256 = LexiconAggregateSHA256(lexicons[index].Files)
	}
	composition := CompositionDescriptor{
		DescriptorSchemaVersion: 1,
		ProfileSetID:            "candidate-default",
		ProfileSetVersion:       "profile-set-v1",
		RankingVersion:          "ranking-v1",
		CompositionVersion:      "composition-v1",
		Components: []CompositionComponent{
			{Role: "preprocessor", ComponentID: "query-preprocessor", SemanticVersion: "preprocessor-v1", PackageRoot: "internal/querypreprocess"},
			{Role: "profile", ComponentID: "core", SemanticVersion: "core-v1", PackageRoot: "internal/queryprofile/core"},
			{Role: "profile", ComponentID: "judicial-cases", SemanticVersion: "judicial-v1", PackageRoot: "internal/queryprofile/judicialcases"},
			{Role: "composer", ComponentID: "candidate-composer", SemanticVersion: "composer-v1", PackageRoot: "internal/application/legalquery"},
			{Role: "selector", ComponentID: "legal-query-selector", SemanticVersion: "selector-v1", PackageRoot: "internal/application/legalquery"},
		},
	}
	composition.DescriptorSHA256 = mustCompositionDigest(t, composition)
	sourceSet := SemanticSourceSet{
		MainModulePath:     "github.com/geonwoo-jeong/japanese-law-mcp",
		GoLanguageVersion:  "1.25.0",
		GoToolchainVersion: "go1.25.0",
		GoDebugSettings:    []GoDebugSetting{},
		GOOS:               "linux",
		GOARCH:             "amd64",
		GOAMD64:            "v1",
		GOEXPERIMENT:       "",
		CGOEnabled:         0,
		BuildTags:          []string{},
		Files:              []FileDigest{{Path: "internal/querypreprocess/preprocess.go", RawSHA256: repeatHex('4')}},
		ModuleDependencies: []ModuleDependency{},
	}
	sourceSet.SourceSetSHA256 = mustSourceSetDigest(t, sourceSet)
	return CandidateContentManifest{
		ArtifactKind:  ArtifactKindCandidateContent,
		SchemaVersion: SchemaVersionV2,
		ProfileSet: ProfileSetIdentity{
			ProfileSetID:      "candidate-default",
			ProfileSetVersion: "profile-set-v1",
			RankingVersion:    "ranking-v1",
		},
		ProfileArtifacts: []ProfileArtifact{
			{ProfileID: "core", ProfileVersion: "core-v1", MetadataSchemaVersion: 1, MetadataCanonicalSHA256: repeatHex('5'), CueSetVersion: "core-cues-v1", CueArtifactSHA256: repeatHex('6')},
			{ProfileID: "judicial-cases", ProfileVersion: "judicial-v1", MetadataSchemaVersion: 1, MetadataCanonicalSHA256: repeatHex('7'), CueSetVersion: "judicial-cues-v1", CueArtifactSHA256: repeatHex('8')},
		},
		LexiconArtifacts:  lexicons,
		Composition:       composition,
		SemanticSourceSet: sourceSet,
	}
}

func manifestWithID(t *testing.T) CandidateContentManifest {
	t.Helper()
	manifest := validCandidateManifest(t)
	manifest.CandidateContentID = mustCandidateID(t, manifest)
	return manifest
}

func validReviewAttestation(
	t *testing.T,
	manifest CandidateContentManifest,
	scope string,
	authority string,
) ReviewAttestation {
	t.Helper()
	sots := validSOTReferences()
	criteria := ArchitectureCriterionIDs()
	if scope == ReviewScopeTestability {
		criteria = TestabilityCriterionIDs()
	}
	scores := make([]CriterionScore, 0, len(criteria))
	for _, criterionID := range criteria {
		scores = append(scores, CriterionScore{CriterionID: criterionID, Score20: 20})
	}
	attestation := ReviewAttestation{
		ArtifactKind:                   ArtifactKindReviewAttestation,
		SchemaVersion:                  SchemaVersionV2,
		CandidateContentID:             manifest.CandidateContentID,
		CandidateContentManifestSHA256: RawSHA256(mustCanonicalJSON(t, manifest)),
		ReviewScope:                    scope,
		RubricVersion:                  ReviewRubricVersion,
		RubricSHA256:                   ReviewRubricSHA256(),
		ReviewerAuthorityID:            authority,
		ReviewedSOTs:                   sots,
		ReviewedSOTSetSHA256:           SOTSetSHA256(sots),
		CriterionScores:                scores,
		Score100:                       100,
		BlockerCount:                   0,
		MajorCount:                     0,
		MinorCount:                     0,
		Decision:                       ReviewDecisionApproved,
	}
	attestation.AttestationID = mustReviewID(t, attestation)
	return attestation
}

func validEvaluationRequest(t *testing.T, manifest CandidateContentManifest) EvaluationRequest {
	t.Helper()
	architecture := validReviewAttestation(t, manifest, ReviewScopeArchitecture, "authority-a")
	testability := validReviewAttestation(t, manifest, ReviewScopeTestability, "authority-b")
	sots := validSOTReferences()
	request := EvaluationRequest{
		ArtifactKind:                   ArtifactKindEvaluationRequest,
		SchemaVersion:                  SchemaVersionV2,
		EvaluatorVersion:               "legal-query-evaluator-v1",
		CorpusVersion:                  "corpus-v10",
		CorpusManifestSHA256:           repeatHex('9'),
		HoldoutDigest:                  repeatHex('a'),
		HoldoutLeakageGroupDigests:     []string{repeatHex('b'), repeatHex('c')},
		CandidateContentID:             manifest.CandidateContentID,
		CandidateContentManifestSHA256: RawSHA256(mustCanonicalJSON(t, manifest)),
		ReviewRubricVersion:            ReviewRubricVersion,
		ReviewRubricSHA256:             ReviewRubricSHA256(),
		RequiredReviewSOTs:             sots,
		RequiredReviewSOTSetSHA256:     SOTSetSHA256(sots),
		ReviewAttestations: []ReviewAttestationReference{
			{ReviewScope: ReviewScopeArchitecture, AttestationID: architecture.AttestationID, AttestationSHA256: RawSHA256(mustCanonicalJSON(t, architecture))},
			{ReviewScope: ReviewScopeTestability, AttestationID: testability.AttestationID, AttestationSHA256: RawSHA256(mustCanonicalJSON(t, testability))},
		},
		BaselineVersion: "default-2",
	}
	request.EvaluationID = mustEvaluationID(t, request)
	return request
}

func validSOTReferences() []SOTReference {
	ids := RequiredReviewSOTIDs()
	references := make([]SOTReference, 0, len(ids))
	for index, id := range ids {
		digit := "0123456789abcdef"[index%16]
		references = append(references, SOTReference{SOTID: id, SOTDocumentSHA256: repeatHex(digit)})
	}
	return references
}

func mustCandidateID(t *testing.T, document CandidateContentManifest) string {
	t.Helper()
	value, err := CanonicalCandidateContentID(document)
	if err != nil {
		t.Fatalf("candidateContentId を計算できません: %v", err)
	}
	return value
}

func mustReviewID(t *testing.T, document ReviewAttestation) string {
	t.Helper()
	value, err := CanonicalReviewAttestationID(document)
	if err != nil {
		t.Fatalf("attestationId を計算できません: %v", err)
	}
	return value
}

func mustEvaluationID(t *testing.T, document EvaluationRequest) string {
	t.Helper()
	value, err := CanonicalEvaluationID(document)
	if err != nil {
		t.Fatalf("evaluationId を計算できません: %v", err)
	}
	return value
}

func mustCompositionDigest(t *testing.T, document CompositionDescriptor) string {
	t.Helper()
	value, err := CanonicalCompositionSHA256(document)
	if err != nil {
		t.Fatalf("composition digest を計算できません: %v", err)
	}
	return value
}

func mustSourceSetDigest(t *testing.T, document SemanticSourceSet) string {
	t.Helper()
	value, err := CanonicalSourceSetSHA256(document)
	if err != nil {
		t.Fatalf("source set digest を計算できません: %v", err)
	}
	return value
}

func mustCanonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := MarshalCanonicalJSON(value)
	if err != nil {
		t.Fatalf("canonical JSON を作れません: %v", err)
	}
	return raw
}

func repeatHex(value byte) string {
	return string(bytes.Repeat([]byte{value}, 64))
}

func testSchemaV2() []byte {
	return []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","oneOf":[{"$ref":"#/$defs/pointer"},{"$ref":"#/$defs/content"},{"$ref":"#/$defs/attestation"},{"$ref":"#/$defs/request"}],"$defs":{"pointer":{"type":"object","properties":{"artifactKind":{"const":"legal_query_candidate_evaluation_pointer"}},"required":["artifactKind"]},"content":{"type":"object","properties":{"artifactKind":{"const":"legal_query_candidate_content"}},"required":["artifactKind"]},"attestation":{"type":"object","properties":{"artifactKind":{"const":"legal_query_candidate_review_attestation"}},"required":["artifactKind"]},"request":{"type":"object","properties":{"artifactKind":{"const":"legal_query_candidate_evaluation_request"}},"required":["artifactKind"]}}}`)
}
