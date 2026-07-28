package legalquerycorpus

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	repositoryCorpusV2Seed           = 20260727
	repositoryCorpusV2Development    = 31
	repositoryCorpusV2Holdout        = 240
	repositoryCorpusV2Execution      = 7
	repositoryCorpusV2HoldoutDigest  = "25b06db5e29ada2922e970a8c569ee9d2e73ce22fae4ffe3ce5880043d3543b8"
	repositoryCorpusV2ManifestSHA256 = "c922e194bab4d8eebc677302dedf8fb5f9cb39292fe9a2b55be0f8acc9a5ebbf"
)

func TestRepositoryCorpusV2の期待値は入力根拠を持つ(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
	corpus, err := Load(
		context.Background(),
		repositoryRoot,
		"testdata/legalquery/corpus-v2",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024, SOT-ENG-026: corpus-v2 Load() error = %v", err)
	}
	if corpus.Manifest().CorpusVersion() != "corpus-v2" {
		t.Fatalf(
			"SOT-ENG-026: corpusVersion = %q",
			corpus.Manifest().CorpusVersion(),
		)
	}
	manifest := corpus.Manifest()
	if manifest.Seed() != repositoryCorpusV2Seed ||
		manifest.HoldoutDigest() != repositoryCorpusV2HoldoutDigest {
		t.Fatalf(
			"SOT-ENG-024: corpus-v2 の固定値 = (%d, %q)",
			manifest.Seed(),
			manifest.HoldoutDigest(),
		)
	}
	if manifest.Development().CaseCount() != repositoryCorpusV2Development ||
		manifest.Holdout().CaseCount() != repositoryCorpusV2Holdout ||
		manifest.Execution().CaseCount() != repositoryCorpusV2Execution ||
		len(corpus.Development()) != repositoryCorpusV2Development ||
		len(corpus.Holdout()) != repositoryCorpusV2Holdout ||
		len(corpus.Execution()) != repositoryCorpusV2Execution {
		t.Fatalf(
			"SOT-ENG-024: corpus-v2 の集合件数 = (%d, %d, %d)",
			len(corpus.Development()),
			len(corpus.Holdout()),
			len(corpus.Execution()),
		)
	}
	assertRepositoryCorpusManifestDigest(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v2/manifest.json",
		repositoryCorpusV2ManifestSHA256,
	)
	assertRepositoryCorpusReproducible(
		t,
		repositoryRoot,
		"testdata/legalquery/corpus-v2",
		corpus,
	)
	assertRepositoryCorpusImmutable(t, corpus)

	for _, fixture := range corpus.Development() {
		assertExpectedDatesAreGrounded(t, fixture)
		assertOfficialIdentifierEvidenceIsGrounded(t, fixture)
	}
	for _, fixture := range corpus.Holdout() {
		assertExpectedDatesAreGrounded(t, fixture)
		assertOfficialIdentifierEvidenceIsGrounded(t, fixture)
	}
}

func assertExpectedDatesAreGrounded(t *testing.T, fixture SemanticCase) {
	t.Helper()

	plan, ok := fixture.Expected().(ExpectedPlan)
	if !ok {
		return
	}
	for _, meaning := range plan.Meanings() {
		for _, step := range meaning.Steps() {
			for _, value := range expectedStepDates(step.LogicalInput()) {
				if queryContainsDate(fixture.Request().Query(), value) {
					continue
				}
				t.Errorf(
					"SOT-MODEL-022: %s の期待日付 %s は query に根拠がありません",
					fixture.CaseID(),
					value,
				)
			}
		}
	}
}

func expectedStepDates(input legalquery.LogicalInput) []string {
	switch value := input.(type) {
	case legalquery.LawSearchIntentV1:
		return optionalExpectedDate(value.AsOf())
	case legalquery.LawContentSearchIntentV1:
		return optionalExpectedDate(value.AsOf())
	case legalquery.LawReadIntentV1:
		return optionalExpectedDate(value.AsOf())
	case legalquery.LawArticleReadIntentV1:
		return optionalExpectedDate(value.AsOf())
	case legalquery.LawUpdateListIntentV1:
		return []string{value.Date().String()}
	case legalquery.JudicialDecisionSearchIntentV1,
		legalquery.JudicialDecisionReadIntentV1:
		return nil
	default:
		panic("SOT-MODEL-022: 未対応の logical input に日付根拠検証を適用できません")
	}
}

func optionalExpectedDate(value model.Date, exists bool) []string {
	if !exists {
		return nil
	}
	return []string{value.String()}
}

func queryContainsDate(query string, value string) bool {
	if strings.Contains(query, value) {
		return true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return false
	}
	japanese := fmt.Sprintf(
		"%d年%d月%d日",
		parsed.Year(),
		parsed.Month(),
		parsed.Day(),
	)
	return strings.Contains(query, japanese)
}

func assertOfficialIdentifierEvidenceIsGrounded(
	t *testing.T,
	fixture SemanticCase,
) {
	t.Helper()

	plan, ok := fixture.Expected().(ExpectedPlan)
	if !ok {
		return
	}
	requestRef, exists := fixture.Request().Ref()
	var productRef model.SourceResourceRef
	if exists {
		converted, err := productRefFromRaw(requestRef)
		if err != nil {
			t.Fatalf(
				"SOT-ENG-026: %s の request.ref を製品 ref に変換できません: %v",
				fixture.CaseID(),
				err,
			)
		}
		productRef = converted
	}
	for _, meaning := range plan.Meanings() {
		if !containsEvidence(
			meaning.EvidenceCodes(),
			legalquery.EvidenceOfficialIdentifier,
		) {
			continue
		}
		if meaningContainsGroundedIdentifier(
			meaning,
			fixture.Request().Query(),
			productRef,
			exists,
		) {
			continue
		}
		t.Errorf(
			"SOT-MODEL-022: %s の official_identifier は query または ref に根拠がありません",
			fixture.CaseID(),
		)
	}
}

func containsEvidence(
	values []legalquery.EvidenceCode,
	target legalquery.EvidenceCode,
) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func meaningContainsGroundedIdentifier(
	meaning ExpectedMeaning,
	query string,
	requestRef model.SourceResourceRef,
	requestRefExists bool,
) bool {
	for _, step := range meaning.Steps() {
		switch value := step.LogicalInput().(type) {
		case legalquery.LawReadIntentV1:
			if lawID, exists := value.LawID(); exists &&
				strings.Contains(query, lawID) {
				return true
			}
			if revisionID, exists := value.RevisionID(); exists &&
				strings.Contains(query, revisionID) {
				return true
			}
			if requestRefExists &&
				lawReadIntentMatchesRequestRef(value, requestRef) {
				return true
			}
		case legalquery.LawArticleReadIntentV1:
			if lawID, exists := value.LawID(); exists &&
				strings.Contains(query, lawID) {
				return true
			}
			if requestRefExists &&
				lawArticleReadIntentMatchesRequestRef(value, requestRef) {
				return true
			}
		case legalquery.JudicialDecisionReadIntentV1:
			if requestRefExists && value.Ref() == requestRef {
				return true
			}
		}
	}
	return false
}

func lawReadIntentMatchesRequestRef(
	input legalquery.LawReadIntentV1,
	requestRef model.SourceResourceRef,
) bool {
	if ref, exists := input.Ref(); exists {
		return ref == requestRef
	}
	key := requestRef.Key()
	if key.ResourceType() != string(legalquery.ResourceLaw) {
		return false
	}
	lawID, exists := input.LawID()
	if !exists || lawID != key.ResourceID() {
		return false
	}
	revisionID, exists := input.RevisionID()
	if !exists {
		return true
	}
	versionID, versioned := key.VersionID()
	return versioned && revisionID == versionID
}

func lawArticleReadIntentMatchesRequestRef(
	input legalquery.LawArticleReadIntentV1,
	requestRef model.SourceResourceRef,
) bool {
	if ref, exists := input.Ref(); exists {
		return ref == requestRef
	}
	lawID, exists := input.LawID()
	if !exists {
		return false
	}
	key := requestRef.Key()
	return key.ResourceType() == string(legalquery.ResourceLaw) &&
		lawID == key.ResourceID()
}

func TestOfficialIdentifierGroundingは異なるRefを拒否する(t *testing.T) {
	matching := newRepositoryCorpusTestRef(t, "law", "129AC0000000089")
	mismatched := newRepositoryCorpusTestRef(t, "law", "140AC0000000045")
	wrongResource := newRepositoryCorpusTestRef(
		t,
		"judicial-decision",
		"129AC0000000089",
	)
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{LawID: "129AC0000000089"},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: test law read intent error = %v", err)
	}

	if !lawReadIntentMatchesRequestRef(input, matching) {
		t.Fatal("SOT-MODEL-022: 同じ law resource ID の ref を根拠として認識しません")
	}
	if lawReadIntentMatchesRequestRef(input, mismatched) {
		t.Fatal("SOT-MODEL-022: 異なる law resource ID の ref を根拠として認識しました")
	}
	if lawReadIntentMatchesRequestRef(input, wrongResource) {
		t.Fatal("SOT-MODEL-022: 異なる resource type の ref を根拠として認識しました")
	}
}

func newRepositoryCorpusTestRef(
	t *testing.T,
	resourceType string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "test-source",
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-011: test resource key error = %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "test-provider",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-016: test resource ref error = %v", err)
	}
	return ref
}
