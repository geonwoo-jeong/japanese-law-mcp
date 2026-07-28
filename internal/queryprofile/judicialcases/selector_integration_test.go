package judicialcases

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestSelectorは裁判例検索をpack状態だけで実行可否へ分ける(t *testing.T) {
	t.Parallel()

	const query = "司法パック無効時に医療過誤の裁判例を検索してください。"
	disabled := planQuery(t, query, nil, false)
	if disabled.Decision() != legalquery.PlanDecisionCapabilityUnavailable ||
		!slices.Equal(
			disabled.ReasonCodes(),
			[]legalquery.ReasonCode{legalquery.ReasonCodeRequiredPackDisabled},
		) {
		t.Fatalf(
			"pack 無効時の plan = decision:%q reasons:%#v",
			disabled.Decision(),
			disabled.ReasonCodes(),
		)
	}
	assertSingleJudicialSelection(t, disabled, legalquery.SelectionAvailabilityPackDisabled)

	enabled := planQuery(t, query, nil, true)
	if enabled.Decision() != legalquery.PlanDecisionSingle ||
		!slices.Equal(
			enabled.ReasonCodes(),
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		) {
		t.Fatalf(
			"pack 有効時の plan = decision:%q reasons:%#v",
			enabled.Decision(),
			enabled.ReasonCodes(),
		)
	}
	assertSingleJudicialSelection(t, enabled, legalquery.SelectionAvailabilityAvailable)

	candidate := enabled.RankedCandidates()[0]
	if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceMorphologicalContext,
	}) {
		t.Fatalf("検索候補の evidence = %#v", candidate.EvidenceCodes())
	}
	steps := candidate.Steps()
	if len(steps) != 1 {
		t.Fatalf("検索候補の steps = %#v", steps)
	}
	input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || input.Query() != "医療過誤" {
		t.Fatalf("検索 logical input = %#v", steps[0].LogicalInput())
	}
}

func TestSelectorは有効な裁判例refだけをread候補にする(t *testing.T) {
	t.Parallel()

	ref := newTestRef(t, "judicial-decision", "95878/detail3")
	plan := planQuery(
		t,
		"指定参照の最高裁判例を取得してください。",
		&ref,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf("read plan decision = %q, reasons=%#v", plan.Decision(), plan.ReasonCodes())
	}
	assertSingleJudicialSelection(t, plan, legalquery.SelectionAvailabilityAvailable)

	candidate := plan.RankedCandidates()[0]
	if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
	}) {
		t.Fatalf("read candidate evidence = %#v", candidate.EvidenceCodes())
	}
	steps := candidate.Steps()
	if len(steps) != 1 {
		t.Fatalf("read candidate steps = %#v", steps)
	}
	input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionReadIntentV1)
	if !ok || input.Ref() != ref {
		t.Fatalf("read logical input = %#v", steps[0].LogicalInput())
	}
}

func planQuery(
	t *testing.T,
	query string,
	ref *model.SourceResourceRef,
	packEnabled bool,
) legalquery.LegalQueryPlan {
	t.Helper()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("profile を読み込めません: %v", err)
	}
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("前処理器を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   ref,
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("前処理できません: %v", err)
	}
	profileSet, err := legalquery.NewQueryProfileSet(
		[]legalquery.QueryProfile{profile},
	)
	if err != nil {
		t.Fatalf("profile set を構築できません: %v", err)
	}
	result, err := profileSet.Collect(preprocessed)
	if err != nil {
		t.Fatalf("profile contribution を集約できません: %v", err)
	}
	enabled := []string(nil)
	if packEnabled {
		enabled = []string{"judicial-cases"}
	}
	packState, err := legalquery.NewStaticPackState(
		[]string{"judicial-cases"},
		enabled,
	)
	if err != nil {
		t.Fatalf("pack state を構築できません: %v", err)
	}
	plan, err := legalquery.SelectLegalQueryPlan(legalquery.SelectorInput{
		ProfileSetResult: result,
		PackState:        packState,
		LimitPerAttempt:  request.LimitPerAttempt(),
	})
	if err != nil {
		t.Fatalf("selector のエラー = %v", err)
	}
	return plan
}

func assertSingleJudicialSelection(
	t *testing.T,
	plan legalquery.LegalQueryPlan,
	availability legalquery.SelectionAvailability,
) {
	t.Helper()

	ranked := plan.RankedCandidates()
	selected := plan.Selected()
	if len(ranked) != 1 || len(selected) != 1 {
		t.Fatalf("ranked=%#v selected=%#v", ranked, selected)
	}
	if selected[0].CandidateID() != ranked[0].CandidateID() ||
		selected[0].Availability() != availability ||
		!slices.Equal(selected[0].RequiredPacks(), []string{"judicial-cases"}) ||
		!slices.Equal(ranked[0].RequiredPacks(), []string{"judicial-cases"}) {
		t.Fatalf("selection=%#v candidate=%#v", selected[0], ranked[0])
	}
}

func newTestRef(
	t *testing.T,
	resourceType string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("ref key を構築できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref を構築できません: %v", err)
	}
	return ref
}
