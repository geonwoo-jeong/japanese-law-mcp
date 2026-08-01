// Package legalquerycandidateeval は、SOT-ENG-038 の候補評価準備成果物を閉じて検証する。
package legalquerycandidateeval

const (
	// SchemaVersionV2 は内容固定済み候補評価成果物の schema 版である。
	SchemaVersionV2 = 2

	ArtifactKindPointer           = "legal_query_candidate_evaluation_pointer"
	ArtifactKindCandidateContent  = "legal_query_candidate_content"
	ArtifactKindReviewAttestation = "legal_query_candidate_review_attestation"
	ArtifactKindEvaluationRequest = "legal_query_candidate_evaluation_request"
	ArtifactKindEvaluationResult  = "legal_query_candidate_evaluation_result"

	EvaluationOutcomePassed = "passed"
	EvaluationOutcomeFailed = "failed"

	ReviewScopeArchitecture = "architecture"
	ReviewScopeTestability  = "testability"
	ReviewRubricVersion     = "sot-review-rubric-v1"
	ReviewDecisionApproved  = "approved"
)

// PointerDocument は、次に評価する一 request だけを指す。
type PointerDocument struct {
	ArtifactKind  string `json:"artifactKind"`
	SchemaVersion int    `json:"schemaVersion"`
	EvaluationID  string `json:"evaluationId"`
}

// CandidateContentManifest は、holdout を開く前に候補の意味内容を固定する。
type CandidateContentManifest struct {
	ArtifactKind       string                `json:"artifactKind"`
	SchemaVersion      int                   `json:"schemaVersion"`
	CandidateContentID string                `json:"candidateContentId"`
	ProfileSet         ProfileSetIdentity    `json:"profileSet"`
	ProfileArtifacts   []ProfileArtifact     `json:"profileArtifacts"`
	LexiconArtifacts   []LexiconArtifact     `json:"lexiconArtifacts"`
	Composition        CompositionDescriptor `json:"composition"`
	SemanticSourceSet  SemanticSourceSet     `json:"semanticSourceSet"`
}

// ProfileSetIdentity は候補 profile set の identity を保持する。
type ProfileSetIdentity struct {
	ProfileSetID      string `json:"profileSetId"`
	ProfileSetVersion string `json:"profileSetVersion"`
	RankingVersion    string `json:"rankingVersion"`
}

// ProfileArtifact は一 profile の metadata と cue identity を保持する。
type ProfileArtifact struct {
	ProfileID               string `json:"profileId"`
	ProfileVersion          string `json:"profileVersion"`
	MetadataSchemaVersion   int    `json:"metadataSchemaVersion"`
	MetadataCanonicalSHA256 string `json:"metadataCanonicalSha256"`
	CueSetVersion           string `json:"cueSetVersion"`
	CueArtifactSHA256       string `json:"cueArtifactSha256"`
}

// LexiconArtifact は一辞書成果物の閉じた file 集合を保持する。
type LexiconArtifact struct {
	LexiconID       string       `json:"lexiconId"`
	LexiconVersion  string       `json:"lexiconVersion"`
	Files           []FileDigest `json:"files"`
	AggregateSHA256 string       `json:"aggregateSha256"`
}

// FileDigest は repository-relative file と原 byte digest を保持する。
type FileDigest struct {
	Path      string `json:"path"`
	RawSHA256 string `json:"rawSha256"`
}

// CompositionDescriptor は候補を組み立てる固定 component 順を保持する。
type CompositionDescriptor struct {
	DescriptorSchemaVersion int                    `json:"descriptorSchemaVersion"`
	ProfileSetID            string                 `json:"profileSetId"`
	ProfileSetVersion       string                 `json:"profileSetVersion"`
	RankingVersion          string                 `json:"rankingVersion"`
	CompositionVersion      string                 `json:"compositionVersion"`
	Components              []CompositionComponent `json:"components"`
	DescriptorSHA256        string                 `json:"descriptorSha256"`
}

// CompositionComponent は一つの意味 component と package root を結び付ける。
type CompositionComponent struct {
	Role            string `json:"role"`
	ComponentID     string `json:"componentId"`
	SemanticVersion string `json:"semanticVersion"`
	PackageRoot     string `json:"packageRoot"`
}

// SemanticSourceSet は固定 build context の意味 source closure を保持する。
type SemanticSourceSet struct {
	MainModulePath     string             `json:"mainModulePath"`
	GoLanguageVersion  string             `json:"goLanguageVersion"`
	GoToolchainVersion string             `json:"goToolchainVersion"`
	GoDebugSettings    []GoDebugSetting   `json:"goDebugSettings"`
	GOOS               string             `json:"goos"`
	GOARCH             string             `json:"goarch"`
	GOAMD64            string             `json:"goamd64"`
	GOEXPERIMENT       string             `json:"goexperiment"`
	CGOEnabled         int                `json:"cgoEnabled"`
	BuildTags          []string           `json:"buildTags"`
	Files              []FileDigest       `json:"files"`
	ModuleDependencies []ModuleDependency `json:"moduleDependencies"`
	SourceSetSHA256    string             `json:"sourceSetSha256"`
}

// GoDebugSetting は root go.mod が明示する一 GODEBUG 設定を保持する。
type GoDebugSetting struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ModuleDependency は実際に import する外部 module archive identity を保持する。
type ModuleDependency struct {
	ModulePath               string `json:"modulePath"`
	Version                  string `json:"version"`
	ModuleZipSum             string `json:"moduleZipSum"`
	ModuleZipRawSHA256       string `json:"moduleZipRawSha256"`
	ModuleZipByteLength      int    `json:"moduleZipByteLength"`
	ModuleZipEntryCount      int    `json:"moduleZipEntryCount"`
	ModuleExpandedByteLength int    `json:"moduleExpandedByteLength"`
	ModuleGoModSum           string `json:"moduleGoModSum"`
	ModuleGoModRawSHA256     string `json:"moduleGoModRawSha256"`
}

// ReviewAttestation は候補内容へ結合した独立 review assertion を保持する。
type ReviewAttestation struct {
	ArtifactKind                   string           `json:"artifactKind"`
	SchemaVersion                  int              `json:"schemaVersion"`
	AttestationID                  string           `json:"attestationId"`
	CandidateContentID             string           `json:"candidateContentId"`
	CandidateContentManifestSHA256 string           `json:"candidateContentManifestSha256"`
	ReviewScope                    string           `json:"reviewScope"`
	RubricVersion                  string           `json:"rubricVersion"`
	RubricSHA256                   string           `json:"rubricSha256"`
	ReviewerAuthorityID            string           `json:"reviewerAuthorityId"`
	ReviewedSOTs                   []SOTReference   `json:"reviewedSOTs"`
	ReviewedSOTSetSHA256           string           `json:"reviewedSOTSetSha256"`
	CriterionScores                []CriterionScore `json:"criterionScores"`
	Score100                       int              `json:"score100"`
	BlockerCount                   int              `json:"blockerCount"`
	MajorCount                     int              `json:"majorCount"`
	MinorCount                     int              `json:"minorCount"`
	Decision                       string           `json:"decision"`
}

// SOTReference は review 対象 SOT と原 byte digest を結び付ける。
type SOTReference struct {
	SOTID             string `json:"sotId"`
	SOTDocumentSHA256 string `json:"sotDocumentSha256"`
}

// CriterionScore は rubric の一 criterion に対する固定 anchor score を保持する。
type CriterionScore struct {
	CriterionID string `json:"criterionId"`
	Score20     int    `json:"score20"`
}

// EvaluationRequest は一回だけ評価する候補 tuple を保持する。
type EvaluationRequest struct {
	ArtifactKind                   string                       `json:"artifactKind"`
	SchemaVersion                  int                          `json:"schemaVersion"`
	EvaluationID                   string                       `json:"evaluationId"`
	EvaluatorVersion               string                       `json:"evaluatorVersion"`
	CorpusVersion                  string                       `json:"corpusVersion"`
	CorpusManifestSHA256           string                       `json:"corpusManifestSha256"`
	HoldoutDigest                  string                       `json:"holdoutDigest"`
	HoldoutLeakageGroupDigests     []string                     `json:"holdoutLeakageGroupDigests"`
	CandidateContentID             string                       `json:"candidateContentId"`
	CandidateContentManifestSHA256 string                       `json:"candidateContentManifestSha256"`
	ReviewRubricVersion            string                       `json:"reviewRubricVersion"`
	ReviewRubricSHA256             string                       `json:"reviewRubricSha256"`
	RequiredReviewSOTs             []SOTReference               `json:"requiredReviewSOTs"`
	RequiredReviewSOTSetSHA256     string                       `json:"requiredReviewSOTSetSha256"`
	ReviewAttestations             []ReviewAttestationReference `json:"reviewAttestations"`
	BaselineVersion                string                       `json:"baselineVersion"`
}

// ReviewAttestationReference は request から一 review file への content binding である。
type ReviewAttestationReference struct {
	ReviewScope       string `json:"reviewScope"`
	AttestationID     string `json:"attestationId"`
	AttestationSHA256 string `json:"attestationSha256"`
}

// EvaluationResult は、評価 request と完成した report の原 byte を一意に結ぶ。
type EvaluationResult struct {
	ArtifactKind  string `json:"artifactKind"`
	SchemaVersion int    `json:"schemaVersion"`
	EvaluationID  string `json:"evaluationId"`
	RequestSHA256 string `json:"requestSha256"`
	Outcome       string `json:"outcome"`
	ReportSHA256  string `json:"reportSha256"`
}
