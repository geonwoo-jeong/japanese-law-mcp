package core

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreProjectionFixture struct {
	profile      *Profile
	preprocessor legalquery.QueryPreprocessor
}

func newCoreProjectionFixture(t *testing.T) coreProjectionFixture {
	t.Helper()

	profile := mustCoreProjectionProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf(
			"%s: preprocessor を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return coreProjectionFixture{
		profile:      profile,
		preprocessor: preprocessor,
	}
}

func (f coreProjectionFixture) preprocess(
	t *testing.T,
	query string,
) legalquery.PreprocessResult {
	t.Helper()

	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf(
			"%s: request を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	preprocessed, err := f.preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf(
			"%s: Preprocess() のエラー = %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return preprocessed
}

func (f coreProjectionFixture) input(
	t *testing.T,
	query string,
) legalquery.CandidateGenerationInput {
	t.Helper()
	return mustCoreProjectionInput(t, f.preprocess(t, query))
}

func (f coreProjectionFixture) inputAndCues(
	t *testing.T,
	query string,
) (legalquery.CandidateGenerationInput, resolvedCues) {
	t.Helper()
	input := f.input(t, query)
	return input, resolveCoreProjectionCues(t, f.profile, input)
}

func mustCoreProjectionProfile(t *testing.T) *Profile {
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
			coreLawNameProjectionContractID,
			err,
		)
	}
	profile, err := newCueTaskRelationV2Profile(base)
	if err != nil {
		t.Fatalf(
			"%s: core evidence profile を準備できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return profile
}

func mustCoreProjectionInput(
	t *testing.T,
	preprocessed legalquery.PreprocessResult,
) legalquery.CandidateGenerationInput {
	t.Helper()

	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf(
			"%s: candidate input を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return input
}

func resolveCoreProjectionCues(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
) resolvedCues {
	t.Helper()

	cues, err := profile.resolveCues(input.CueMentions())
	if err == nil {
		cues, err = profile.resolveRelationV2Cues(input, cues)
	}
	if err != nil {
		t.Fatalf(
			"%s: cue を解決できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return cues
}

func lawSpanBySurface(
	t *testing.T,
	input legalquery.CandidateGenerationInput,
	surface string,
) legalquery.QuerySpan {
	t.Helper()

	var result legalquery.QuerySpan
	found := false
	for _, mention := range input.LawNameMentions() {
		if mention.Surface() != surface {
			continue
		}
		if found && !sameQuerySpan(result, mention.Span()) {
			t.Fatalf(
				"%s: 表記 %q が複数 span にあります",
				coreLawNameProjectionContractID,
				surface,
			)
		}
		result = mention.Span()
		found = true
	}
	if !found {
		t.Fatalf(
			"%s: 法令名 %q が見つかりません: %#v",
			coreLawNameProjectionContractID,
			surface,
			input.LawNameMentions(),
		)
	}
	return result
}

func mustCoreProjectionSpan(
	t *testing.T,
	startByte int,
	endByte int,
) legalquery.QuerySpan {
	t.Helper()

	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf(
			"%s: span を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return span
}

func mustCoreProjectionContext(
	t *testing.T,
	lawNameSpan legalquery.QuerySpan,
	individualTopic *legalquery.QuerySpan,
	sharedTerminalTopic *coreValidatedContentTopic,
) coreLawNameProjectionContext {
	t.Helper()

	projectionContext, err := newCoreLawNameProjectionContext(
		lawNameSpan,
		individualTopic,
		sharedTerminalTopic,
	)
	if err != nil {
		t.Fatalf(
			"%s: 投影 context を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return projectionContext
}

func mustCoreValidatedTopic(
	t *testing.T,
	span legalquery.QuerySpan,
	evidence []profileevidence.EvidenceValues,
) coreValidatedContentTopic {
	t.Helper()

	validated, err := newCoreValidatedContentTopic(
		span,
		evidence,
		coreLawProvisionBindingExplicitResource,
	)
	if err != nil {
		t.Fatalf(
			"%s: 検証済み主題を作成できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return validated
}

func coreProjectionTaskResourceEvidence(
	includeResource bool,
) []profileevidence.EvidenceValues {
	result := []profileevidence.EvidenceValues{
		{
			FactID: "cue-task",
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitTask,
		},
	}
	if includeResource {
		result = append(result, profileevidence.EvidenceValues{
			FactID: "cue-resource",
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitResource,
		})
	}
	return result
}

func mustCoreProjectedOption(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	projectionContext coreLawNameProjectionContext,
) coreTopicOption {
	t.Helper()

	option, exists, err := profile.projectCoreLawName(
		input,
		cues,
		projectionContext,
	)
	if err != nil || !exists {
		t.Fatalf(
			"%s: 投影結果 = (exists:%t, err:%v)",
			coreLawNameProjectionContractID,
			exists,
			err,
		)
	}
	return option
}

func assertCoreProjectionRejected(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	projectionContext coreLawNameProjectionContext,
) {
	t.Helper()

	option, exists, err := profile.projectCoreLawName(
		input,
		cues,
		projectionContext,
	)
	if err != nil || exists {
		t.Fatalf(
			"%s: 投影を拒否しませんでした: exists=%t err=%v option=%#v",
			coreLawNameProjectionContractID,
			exists,
			err,
			option,
		)
	}
}

func assertCoreProjectionOption(
	t *testing.T,
	option coreTopicOption,
	wantTerm string,
	wantCode legalquery.EvidenceCode,
	wantBindings int,
) {
	t.Helper()

	if !slices.Equal(option.input.AllTerms(), []string{wantTerm}) ||
		len(option.input.AnyTerms()) != 0 ||
		len(option.input.ExcludeTerms()) != 0 {
		t.Fatalf(
			"%s: 本文検索語 = %#v",
			coreLawNameProjectionContractID,
			option.input,
		)
	}
	bindings := 0
	positive := 0
	clusterSpans := 0
	otherCode := legalquery.EvidenceGeneralTerm
	if wantCode == legalquery.EvidenceGeneralTerm {
		otherCode = legalquery.EvidenceOfficialAlias
	}
	for _, evidence := range option.evidence {
		if evidence.Code == otherCode {
			t.Fatalf(
				"%s: 後続経路の根拠 %q が混在しました",
				coreLawNameProjectionContractID,
				otherCode,
			)
		}
		if evidence.Code != wantCode {
			continue
		}
		bindings++
		if evidence.IndependentPositive {
			positive++
		}
		if evidence.ClusterSpan {
			clusterSpans++
		}
	}
	if bindings != wantBindings || positive != 1 || clusterSpans != 1 {
		t.Fatalf(
			"%s: anchor bindings=%d positive=%d clusterSpans=%d evidence=%#v",
			coreLawNameProjectionContractID,
			bindings,
			positive,
			clusterSpans,
			option.evidence,
		)
	}
}

func replaceCoreProjectionLawMentions(
	t *testing.T,
	base legalquery.PreprocessResult,
	laws []legalquery.LawNameMention,
) legalquery.CandidateGenerationInput {
	t.Helper()

	ref, hasRef := base.Ref()
	values := legalquery.PreprocessResultValues{
		Query:                base.Query(),
		ComparisonKey:        base.ComparisonKey(),
		LawNameMentions:      slices.Clone(laws),
		LegalConceptMentions: base.LegalConceptMentions(),
		CueMentions:          base.CueMentions(),
		IdentifierMentions:   base.IdentifierMentions(),
		DateMentions:         base.DateMentions(),
		ArticleMentions:      base.ArticleMentions(),
		ParagraphMentions:    base.ParagraphMentions(),
		CaseNumberMentions:   base.CaseNumberMentions(),
		QueryTermMentions:    base.QueryTermMentions(),
		CueTaskRelations:     base.CueTaskRelations(),
	}
	if hasRef {
		values.Ref = &ref
	}
	preprocessed, err := legalquery.NewPreprocessResult(values)
	if err != nil {
		t.Fatalf(
			"%s: preprocess fixture を再構築できません: %v",
			coreLawNameProjectionContractID,
			err,
		)
	}
	return mustCoreProjectionInput(t, preprocessed)
}
