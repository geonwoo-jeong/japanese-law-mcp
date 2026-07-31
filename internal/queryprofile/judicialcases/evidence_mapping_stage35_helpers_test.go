package judicialcases

import (
	"context"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func mustJudicialEvidenceProfile(t *testing.T) *Profile {
	t.Helper()

	profile := mustRelationV2Profile(t)
	next, err := newJudicialEvidenceProfile(profile)
	if err != nil {
		t.Fatalf("judicial 3.5 profile を準備できません: %v", err)
	}
	return next
}

func preprocessJudicialEvidenceQuery(
	t *testing.T,
	profile *Profile,
	query string,
	ref *model.SourceResourceRef,
	verificationID string,
) legalquery.PreprocessResult {
	t.Helper()

	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("%s: preprocessor を作成できません: %v", verificationID, err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   ref,
	})
	if err != nil {
		t.Fatalf("%s: request を作成できません: %v", verificationID, err)
	}
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("%s: Preprocess() のエラー = %v", verificationID, err)
	}
	return result
}

func generateJudicialEvidenceQuery(
	t *testing.T,
	profile *Profile,
	query string,
	ref *model.SourceResourceRef,
	verificationID string,
) legalquery.CandidateGeneration {
	t.Helper()

	preprocessed := preprocessJudicialEvidenceQuery(
		t,
		profile,
		query,
		ref,
		verificationID,
	)
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("%s: candidate input を作成できません: %v", verificationID, err)
	}
	return generateJudicialEvidenceInput(
		t,
		profile,
		input,
		verificationID,
	)
}

func generateJudicialEvidenceInput(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
	verificationID string,
) legalquery.CandidateGeneration {
	t.Helper()

	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("%s: candidate scope を作成できません: %v", verificationID, err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("%s: Generate() のエラー = %v", verificationID, err)
	}
	return generation
}

func mustSingleJudicialEvidenceCandidate(
	t *testing.T,
	generation legalquery.CandidateGeneration,
	verificationID string,
) legalquery.LegalQueryCandidate {
	t.Helper()

	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"%s: candidates = %#v、期待件数は 1",
			verificationID,
			candidates,
		)
	}
	return candidates[0]
}

func judicialEvidenceSearchQueries(
	t *testing.T,
	candidate legalquery.LegalQueryCandidate,
	verificationID string,
) []string {
	t.Helper()

	result := make([]string, 0, len(candidate.Steps()))
	for _, step := range candidate.Steps() {
		input, ok := step.LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if !ok {
			t.Fatalf(
				"%s: 裁判例検索以外の step = %#v",
				verificationID,
				step,
			)
		}
		result = append(result, input.Query())
	}
	return result
}

func generationHasJudicialRead(
	generation legalquery.CandidateGeneration,
) bool {
	for _, candidate := range generation.Candidates() {
		if slices.ContainsFunc(
			candidate.Steps(),
			func(step legalquery.LegalQueryCandidateStep) bool {
				return step.InputKind() ==
					legalquery.InputKindJudicialDecisionRead
			},
		) {
			return true
		}
	}
	return false
}

func addJudicialEvidenceCue(
	t *testing.T,
	base legalquery.PreprocessResult,
	surface string,
	profileID string,
	cueID string,
	verificationID string,
) legalquery.CandidateGenerationInput {
	t.Helper()

	start := strings.Index(base.Query(), surface)
	if start < 0 {
		t.Fatalf("%s: cue surface %q が query にありません", verificationID, surface)
	}
	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: start,
		EndByte:   start + len(surface),
	})
	if err != nil {
		t.Fatalf("%s: cue span を作成できません: %v", verificationID, err)
	}
	mention, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      span,
		Surface:   surface,
		ProfileID: profileID,
		CueID:     cueID,
		MatchKind: legalquery.PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("%s: cue mention を作成できません: %v", verificationID, err)
	}
	cues := append(base.CueMentions(), mention)
	sort.Slice(cues, func(left int, right int) bool {
		leftSpan := cues[left].Span()
		rightSpan := cues[right].Span()
		if leftSpan.StartByte() != rightSpan.StartByte() {
			return leftSpan.StartByte() < rightSpan.StartByte()
		}
		if leftSpan.EndByte() != rightSpan.EndByte() {
			return leftSpan.EndByte() < rightSpan.EndByte()
		}
		if cues[left].ProfileID() != cues[right].ProfileID() {
			return cues[left].ProfileID() < cues[right].ProfileID()
		}
		return cues[left].CueID() < cues[right].CueID()
	})
	ref, hasRef := base.Ref()
	var refPointer *model.SourceResourceRef
	if hasRef {
		refPointer = &ref
	}
	rebuilt, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                base.Query(),
			ComparisonKey:        base.ComparisonKey(),
			Ref:                  refPointer,
			LawNameMentions:      base.LawNameMentions(),
			LegalConceptMentions: base.LegalConceptMentions(),
			CueMentions:          cues,
			IdentifierMentions:   base.IdentifierMentions(),
			DateMentions:         base.DateMentions(),
			ArticleMentions:      base.ArticleMentions(),
			ParagraphMentions:    base.ParagraphMentions(),
			CaseNumberMentions:   base.CaseNumberMentions(),
			QueryTermMentions:    base.QueryTermMentions(),
			CueTaskRelations:     base.CueTaskRelations(),
		},
	)
	if err != nil {
		t.Fatalf("%s: preprocess fixture を再構築できません: %v", verificationID, err)
	}
	input, err := legalquery.NewCandidateGenerationInput(rebuilt)
	if err != nil {
		t.Fatalf("%s: candidate input を再構築できません: %v", verificationID, err)
	}
	return input
}
