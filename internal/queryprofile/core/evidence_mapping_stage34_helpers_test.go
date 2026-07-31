package core

import (
	"context"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func mustCoreEvidenceProfile(t *testing.T) *Profile {
	t.Helper()

	lawNames, concepts := mustEmbeddedLexicons(t)
	base, err := Load(
		relationV2ProfileJSON,
		relationV2CuesJSON,
		lawNames,
		concepts,
	)
	if err != nil {
		t.Fatalf(
			"%s: relation-v2 fixture を読み込めません: %v",
			coreEvidenceMappingPrivateLifetimeID,
			err,
		)
	}
	return base
}

func generateCoreEvidenceQuery(
	t *testing.T,
	profile *Profile,
	query string,
	ref *model.SourceResourceRef,
	verificationID string,
) legalquery.CandidateGeneration {
	t.Helper()

	preprocessed := preprocessCoreEvidenceQuery(
		t,
		profile,
		query,
		ref,
		verificationID,
	)
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf(
			"%s: candidate generation input を作成できません: %v",
			verificationID,
			err,
		)
	}
	return generateCoreEvidenceInput(t, profile, input, verificationID)
}

func preprocessCoreEvidenceQuery(
	t *testing.T,
	profile *Profile,
	query string,
	ref *model.SourceResourceRef,
	verificationID string,
) legalquery.PreprocessResult {
	t.Helper()

	preprocessor, err := querypreprocess.NewEmbedded(
		profile.CueVocabulary(),
	)
	if err != nil {
		t.Fatalf(
			"%s: preprocessor を作成できません: %v",
			verificationID,
			err,
		)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   ref,
	})
	if err != nil {
		t.Fatalf(
			"%s: request を作成できません: %v",
			verificationID,
			err,
		)
	}
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf(
			"%s: Preprocess() のエラー = %v",
			verificationID,
			err,
		)
	}
	return result
}

func generateCoreEvidenceInput(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
	verificationID string,
) legalquery.CandidateGeneration {
	t.Helper()

	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf(
			"%s: candidate scope を作成できません: %v",
			verificationID,
			err,
		)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf(
			"%s: Generate() のエラー = %v",
			verificationID,
			err,
		)
	}
	return generation
}

func assertSingleContentCandidate(
	t *testing.T,
	generation legalquery.CandidateGeneration,
	wantTerms [][]string,
	verificationID string,
) {
	t.Helper()

	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"%s: candidates = %#v、期待件数は 1",
			verificationID,
			candidates,
		)
	}
	steps := candidates[0].Steps()
	if len(steps) != len(wantTerms) {
		t.Fatalf(
			"%s: steps = %#v、期待件数は %d",
			verificationID,
			steps,
			len(wantTerms),
		)
	}
	actualTerms := make([][]string, 0, len(steps))
	for index, step := range steps {
		input, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
		if !ok ||
			len(input.AnyTerms()) != 0 ||
			len(input.ExcludeTerms()) != 0 {
			t.Fatalf(
				"%s: steps[%d] の本文検索入力 = %#v",
				verificationID,
				index,
				step.LogicalInput(),
			)
		}
		actualTerms = append(actualTerms, input.AllTerms())
	}
	if !slices.EqualFunc(
		actualTerms,
		wantTerms,
		slices.Equal,
	) {
		t.Fatalf(
			"%s: 本文検索語 = %#v、期待値は %#v",
			verificationID,
			actualTerms,
			wantTerms,
		)
	}
}

func rebuildCoreEvidenceInput(
	t *testing.T,
	base legalquery.PreprocessResult,
	laws []legalquery.LawNameMention,
	concepts []legalquery.LegalConceptMention,
	terms []legalquery.QueryTermMention,
	verificationID string,
) legalquery.CandidateGenerationInput {
	t.Helper()

	ref, hasRef := base.Ref()
	var refPointer *model.SourceResourceRef
	if hasRef {
		refPointer = &ref
	}
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                base.Query(),
			ComparisonKey:        base.ComparisonKey(),
			Ref:                  refPointer,
			LawNameMentions:      laws,
			LegalConceptMentions: concepts,
			CueMentions:          base.CueMentions(),
			IdentifierMentions:   base.IdentifierMentions(),
			DateMentions:         base.DateMentions(),
			ArticleMentions:      base.ArticleMentions(),
			ParagraphMentions:    base.ParagraphMentions(),
			CaseNumberMentions:   base.CaseNumberMentions(),
			QueryTermMentions:    terms,
			CueTaskRelations:     base.CueTaskRelations(),
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: preprocess fixture を再構築できません: %v",
			verificationID,
			err,
		)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf(
			"%s: candidate input fixture を作成できません: %v",
			verificationID,
			err,
		)
	}
	return input
}

func generationContainsInputKind(
	generation legalquery.CandidateGeneration,
	kind legalquery.LogicalInputKind,
) bool {
	for _, candidate := range generation.Candidates() {
		for _, step := range candidate.Steps() {
			if step.InputKind() == kind {
				return true
			}
		}
	}
	return false
}

func mustCoreEvidenceLawRef(
	t *testing.T,
	providerID string,
	sourceID string,
	verificationID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   "129AC0000000089",
	})
	if err != nil {
		t.Fatalf("%s: ref key を作成できません: %v", verificationID, err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("%s: ref を作成できません: %v", verificationID, err)
	}
	return ref
}

func singleRefReadCandidate(
	t *testing.T,
	generation legalquery.CandidateGeneration,
	verificationID string,
) (legalquery.LegalQueryCandidate, int) {
	t.Helper()

	candidates := generation.Candidates()
	members := generation.CompositionMembers()
	if len(candidates) != 1 ||
		len(candidates[0].Steps()) != 1 ||
		candidates[0].Steps()[0].InputKind() !=
			legalquery.InputKindLawRead ||
		len(members) != 1 ||
		len(members[0].StepOrigins()) != 1 {
		t.Fatalf(
			"%s: ref read candidate/member = %#v/%#v",
			verificationID,
			candidates,
			members,
		)
	}
	return candidates[0], members[0].StepOrigins()[0].SourceStartByte()
}

func syntheticSharedTerminalConceptInput(
	t *testing.T,
	profile *Profile,
	count int,
	typoIndex int,
	verificationID string,
) legalquery.CandidateGenerationInput {
	t.Helper()

	const (
		query   = "仮語、帰化を教えてください"
		surface = "仮語"
	)
	base := preprocessCoreEvidenceQuery(
		t,
		profile,
		query,
		nil,
		verificationID,
	)
	startByte := strings.Index(query, surface)
	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   startByte + len(surface),
	})
	if err != nil {
		t.Fatalf("%s: concept span を作成できません: %v", verificationID, err)
	}
	definitions := coreEvidenceConceptFixtures(
		t,
		profile,
		count,
		surface,
		verificationID,
	)
	concepts := make([]legalquery.LegalConceptMention, 0, count)
	for index, definition := range definitions {
		matchKind := legalquery.PreprocessMatchRegisteredTerm
		if index == typoIndex {
			matchKind = legalquery.PreprocessMatchUniqueTypoCorrection
		}
		mention, mentionErr := legalquery.NewLegalConceptMention(
			legalquery.LegalConceptMentionValues{
				Span:      span,
				Surface:   surface,
				ConceptID: definition.entry.ConceptID,
				Canonical: definition.entry.Canonical,
				MatchKind: matchKind,
			},
		)
		if mentionErr != nil {
			t.Fatalf(
				"%s: concept fixture を作成できません: %v",
				verificationID,
				mentionErr,
			)
		}
		concepts = append(concepts, mention)
	}
	sort.Slice(concepts, func(left int, right int) bool {
		return concepts[left].ConceptID() < concepts[right].ConceptID()
	})

	terms := make([]legalquery.QueryTermMention, 0)
	for _, term := range base.QueryTermMentions() {
		if term.Span() == span {
			continue
		}
		terms = append(terms, term)
	}
	return rebuildCoreEvidenceInput(
		t,
		base,
		base.LawNameMentions(),
		concepts,
		terms,
		verificationID,
	)
}

func coreEvidenceConceptFixtures(
	t *testing.T,
	profile *Profile,
	count int,
	surface string,
	verificationID string,
) []conceptDefinition {
	t.Helper()

	ids := make([]string, 0, len(profile.concepts))
	for conceptID := range profile.concepts {
		ids = append(ids, conceptID)
	}
	sort.Strings(ids)
	result := make([]conceptDefinition, 0, count)
	seenTerms := make(map[string]struct{})
	for _, conceptID := range ids {
		definition := profile.concepts[conceptID]
		if definition.entry.SelectionPolicy !=
			legalconceptlexicon.SelectionPolicySingleCandidate {
			continue
		}
		var term string
		coreCount := 0
		for _, candidate := range definition.entry.Candidates {
			if !isCoreConceptCandidate(candidate) {
				continue
			}
			coreCount++
			term = candidate.OfficialTermFor(surface)
		}
		if coreCount != 1 || term == "" {
			continue
		}
		if _, exists := seenTerms[term]; exists {
			continue
		}
		seenTerms[term] = struct{}{}
		result = append(result, definition)
		if len(result) == count {
			return result
		}
	}
	t.Fatalf(
		"%s: 相異なる検索語を持つ concept fixture が %d 件ありません",
		verificationID,
		count,
	)
	return nil
}

func scoreRange(candidates []legalquery.LegalQueryCandidate) int {
	if len(candidates) == 0 {
		return 0
	}
	minimum := candidates[0].SemanticScore()
	maximum := minimum
	for _, candidate := range candidates[1:] {
		minimum = min(minimum, candidate.SemanticScore())
		maximum = max(maximum, candidate.SemanticScore())
	}
	return maximum - minimum
}
