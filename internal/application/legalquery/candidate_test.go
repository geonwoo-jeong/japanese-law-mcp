package legalquery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLegalQueryCandidateAcceptsSevenExactStepMappings(t *testing.T) {
	t.Parallel()

	for name, fixture := range validStepFixtures(t) {
		name := name
		fixture := fixture
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			step, err := NewLegalQueryCandidateStep(fixture.values)
			if err != nil {
				t.Fatalf("SOT-MODEL-022: step を作成できません: %v", err)
			}
			if step.StepID() != fixture.values.StepID ||
				step.Task() != fixture.values.Task ||
				step.Resource() != fixture.values.Resource ||
				step.CapabilityID() != fixture.values.CapabilityID ||
				step.CapabilityMajorVersion() != 1 ||
				step.InputKind() != fixture.values.InputKind {
				t.Fatalf("SOT-MODEL-022: step = %#v", step)
			}
			if step.LogicalInput().InputKind() != fixture.values.InputKind {
				t.Fatalf(
					"SOT-MODEL-022: logical input kind = %q",
					step.LogicalInput().InputKind(),
				)
			}
		})
	}
}

func TestLegalQueryCandidateStepRejectsEveryMappingMismatch(t *testing.T) {
	t.Parallel()

	base := validStepFixtures(t)["法令検索"].values
	tests := map[string]LegalQueryCandidateStepValues{
		"不正な step ID":    withStepID(base, "Step 1"),
		"異なる task":       withTask(base, TaskRead),
		"異なる resource":   withResource(base, ResourceLawProvision),
		"異なる capability": withCapabilityID(base, "law.document.read"),
		"異なる major":      withCapabilityMajor(base, 2),
		"異なる inputKind":  withInputKind(base, InputKindLawRead),
		"inputKind と logicalInput の不一致": withLogicalInput(
			base,
			mustLawUpdateIntent(t),
		),
		"logicalInput の欠落": withLogicalInput(base, nil),
	}
	var nilLawSearch *LawSearchIntentV1
	tests["型付き nil logicalInput"] = withLogicalInput(base, nilLawSearch)

	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryCandidateStep(values); err == nil {
				t.Fatal("SOT-MODEL-022: 許可されていない step 対応を受理しました")
			}
		})
	}
}

func TestLegalQueryCandidateIsImmutableAndKeepsCanonicalEvidence(t *testing.T) {
	t.Parallel()

	step := mustValidStep(t, "法令検索")
	evidence := []EvidenceCode{
		EvidenceOfficialAlias,
		EvidenceLegalConcept,
		EvidenceMorphologicalContext,
	}
	steps := []LegalQueryCandidateStep{step}
	requiredPacks := []string{"tax"}
	conceptSources := []LegalConceptSource{mustConceptSource(t)}
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    "candidate-1",
		SemanticScore:  720,
		Confidence:     ConfidenceHigh,
		EvidenceCodes:  evidence,
		ConceptSources: conceptSources,
		RequiredPacks:  requiredPacks,
		Steps:          steps,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: candidate を作成できません: %v", err)
	}

	evidence[0] = EvidenceGeneralTerm
	steps[0] = LegalQueryCandidateStep{}
	requiredPacks[0] = "labor"
	conceptSources[0] = LegalConceptSource{}

	if candidate.CandidateID() != "candidate-1" ||
		candidate.SemanticScore() != 720 ||
		candidate.Confidence() != ConfidenceHigh {
		t.Fatalf("SOT-MODEL-022: candidate scalar = %#v", candidate)
	}
	gotEvidence := candidate.EvidenceCodes()
	if len(gotEvidence) != 3 || gotEvidence[0] != EvidenceOfficialAlias {
		t.Fatalf("SOT-MODEL-022: EvidenceCodes() = %#v", gotEvidence)
	}
	if got := candidate.ConceptSources(); len(got) != 1 ||
		got[0].ConceptID() != "permanent-residence" {
		t.Fatalf("SOT-MODEL-022: ConceptSources() = %#v", candidate.ConceptSources())
	}
	if got := candidate.RequiredPacks(); len(got) != 1 || got[0] != "tax" {
		t.Fatalf("SOT-MODEL-022: RequiredPacks() = %#v", candidate.RequiredPacks())
	}
	gotSteps := candidate.Steps()
	if len(gotSteps) != 1 || gotSteps[0].StepID() != "step-1" {
		t.Fatalf("SOT-MODEL-022: Steps() = %#v", gotSteps)
	}

	gotEvidence[0] = EvidenceGeneralTerm
	gotSteps[0] = LegalQueryCandidateStep{}
	gotConceptSources := candidate.ConceptSources()
	gotConceptSources[0] = LegalConceptSource{}
	gotRequiredPacks := candidate.RequiredPacks()
	gotRequiredPacks[0] = "labor"
	if candidate.EvidenceCodes()[0] != EvidenceOfficialAlias ||
		candidate.Steps()[0].StepID() != "step-1" ||
		candidate.ConceptSources()[0].ConceptID() != "permanent-residence" ||
		candidate.RequiredPacks()[0] != "tax" {
		t.Fatal("SOT-MODEL-022: getter から candidate を変更できました")
	}
}

func TestLegalQueryCandidateRequiresSourcedLegalConcept(t *testing.T) {
	t.Parallel()

	source, err := NewLegalConceptSource(LegalConceptSourceValues{
		ConceptID:   "permanent-residence",
		Title:       "出入国在留管理庁「永住許可に関するガイドライン」",
		URL:         "https://www.moj.go.jp/isa/applications/resources/nyukan_nyukan50.html",
		ConfirmedOn: mustDate(t, "2026-07-27"),
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: LegalConceptSource を作成できません: %v", err)
	}
	if source.ConceptID() != "permanent-residence" ||
		source.Title() == "" ||
		source.URL() == "" ||
		source.ConfirmedOn().String() != "2026-07-27" {
		t.Fatalf("SOT-MODEL-022: source = %#v", source)
	}

	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    "candidate-concept",
		SemanticScore:  510,
		Confidence:     ConfidenceMedium,
		EvidenceCodes:  []EvidenceCode{EvidenceLegalConcept},
		ConceptSources: []LegalConceptSource{source},
		Steps:          []LegalQueryCandidateStep{mustValidStep(t, "法令本文検索")},
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: concept candidate を作成できません: %v", err)
	}
	if len(candidate.ConceptSources()) != 1 {
		t.Fatalf("SOT-MODEL-022: ConceptSources() = %#v", candidate.ConceptSources())
	}

	missingSource := validCandidateValues(t)
	missingSource.EvidenceCodes = []EvidenceCode{EvidenceLegalConcept}
	missingSource.ConceptSources = nil
	if _, err := NewLegalQueryCandidate(missingSource); err == nil {
		t.Fatal("SOT-MODEL-022: 出典のない legal_concept を受理しました")
	}

	unrelatedSource := validCandidateValues(t)
	unrelatedSource.ConceptSources = []LegalConceptSource{source}
	if _, err := NewLegalQueryCandidate(unrelatedSource); err == nil {
		t.Fatal("SOT-MODEL-022: legal_concept でない候補が出典を持てました")
	}
}

func TestLegalQueryCandidateEnforcesEvidenceStepsAndPackInvariants(t *testing.T) {
	t.Parallel()

	tests := map[string]func(LegalQueryCandidateValues) LegalQueryCandidateValues{
		"不正な candidate ID": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.CandidateID = "candidate_1"
			return values
		},
		"上限超過の candidate ID": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.CandidateID = strings.Repeat("a", 65)
			return values
		},
		"不正な confidence": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.Confidence = Confidence("certain")
			return values
		},
		"根拠なし": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.EvidenceCodes = nil
			return values
		},
		"未知の根拠": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.EvidenceCodes = []EvidenceCode{"unknown"}
			return values
		},
		"重複する根拠": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.EvidenceCodes = []EvidenceCode{
				EvidenceOfficialAlias,
				EvidenceOfficialAlias,
			}
			return values
		},
		"根拠の順序違反": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.EvidenceCodes = []EvidenceCode{
				EvidenceGeneralTerm,
				EvidenceOfficialAlias,
			}
			return values
		},
		"step なし": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.Steps = nil
			return values
		},
		"五つの step": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.Steps = []LegalQueryCandidateStep{
				mustValidStep(t, "法令検索"),
				mustValidStep(t, "法令本文検索"),
				mustValidStep(t, "法令読取り"),
				mustValidStep(t, "条文読取り"),
				mustValidStep(t, "更新一覧"),
			}
			return values
		},
		"重複する step ID": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.Steps = []LegalQueryCandidateStep{
				mustValidStep(t, "法令検索"),
				mustValidStep(t, "法令検索"),
			}
			return values
		},
		"pack の重複": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.Steps = []LegalQueryCandidateStep{mustValidStep(t, "裁判例検索")}
			values.RequiredPacks = []string{"judicial-cases", "judicial-cases"}
			return values
		},
		"pack ID の形式違反": func(values LegalQueryCandidateValues) LegalQueryCandidateValues {
			values.RequiredPacks = []string{"Judicial_Cases"}
			return values
		},
	}

	for name, mutate := range tests {
		name := name
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryCandidate(mutate(validCandidateValues(t))); err == nil {
				t.Fatal("SOT-MODEL-022: 不正な candidate を受理しました")
			}
		})
	}

	judicialValues := validCandidateValues(t)
	judicialValues.Steps = []LegalQueryCandidateStep{mustValidStep(t, "裁判例検索")}
	judicialValues.RequiredPacks = []string{"judicial-cases"}
	if _, err := NewLegalQueryCandidate(judicialValues); err != nil {
		t.Fatalf("SOT-MODEL-022: judicial candidate を拒否しました: %v", err)
	}

	domainPackValues := validCandidateValues(t)
	domainPackValues.RequiredPacks = []string{"tax"}
	if _, err := NewLegalQueryCandidate(domainPackValues); err != nil {
		t.Fatalf("SOT-ARCH-019: law step を再利用する pack candidate を拒否しました: %v", err)
	}
}

func TestCandidateGettersFailFastOnBrokenInternalInvariant(t *testing.T) {
	t.Parallel()

	step := mustValidStep(t, "法令検索")
	var typedNil *LawSearchIntentV1
	step.logicalInput = typedNil
	assertPanics(t, func() {
		_ = step.LogicalInput()
	})

	candidate := LegalQueryCandidate{
		candidateID:   "candidate-broken",
		confidence:    ConfidenceLow,
		evidenceCodes: []EvidenceCode{EvidenceGeneralTerm},
		requiredPacks: []string{},
		steps:         []LegalQueryCandidateStep{step},
	}
	assertPanics(t, func() {
		_ = candidate.Steps()
	})
}

func TestLegalConceptSourceRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := LegalConceptSourceValues{
		ConceptID:   "permanent-residence",
		Title:       "公的資料",
		URL:         "https://example.go.jp/reference",
		ConfirmedOn: mustDate(t, "2026-07-27"),
	}
	tests := map[string]LegalConceptSourceValues{
		"conceptId 欠落": withConceptID(valid, ""),
		"title 欠落":     withConceptTitle(valid, ""),
		"HTTP URL":     withConceptURL(valid, "http://example.go.jp/reference"),
		"利用者情報付き URL": withConceptURL(
			valid,
			"https://user@example.go.jp/reference",
		),
		"確認日欠落": withConfirmedOn(valid, model.Date{}),
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalConceptSource(values); err == nil {
				t.Fatal("SOT-MODEL-022: 不正な concept source を受理しました")
			}
		})
	}
}

func TestCandidateModelsRejectDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	for _, target := range []any{
		&LegalConceptSource{},
		&LegalQueryCandidateStep{},
		&LegalQueryCandidate{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-022: %T を JSON から直接復元できました", target)
		}
	}
}

type stepFixture struct {
	values LegalQueryCandidateStepValues
}

func validStepFixtures(t *testing.T) map[string]stepFixture {
	t.Helper()
	lawRef := newLegalQuerySourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"law-1",
	)
	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	return map[string]stepFixture{
		"法令検索": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-1",
				Task:                   TaskSearch,
				Resource:               ResourceLaw,
				CapabilityID:           "law.search",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindLawSearch,
				LogicalInput:           mustLawSearchIntent(t),
			},
		},
		"法令本文検索": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-2",
				Task:                   TaskSearch,
				Resource:               ResourceLawProvision,
				CapabilityID:           "law.content.search",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindLawContentSearch,
				LogicalInput:           mustLawContentSearchIntent(t),
			},
		},
		"法令読取り": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-3",
				Task:                   TaskRead,
				Resource:               ResourceLaw,
				CapabilityID:           "law.document.read",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindLawRead,
				LogicalInput:           mustLawReadIntent(t, lawRef),
			},
		},
		"条文読取り": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-4",
				Task:                   TaskRead,
				Resource:               ResourceLawProvision,
				CapabilityID:           "law.article.read",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindLawArticleRead,
				LogicalInput:           mustLawArticleReadIntent(t, lawRef),
			},
		},
		"更新一覧": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-5",
				Task:                   TaskListUpdates,
				Resource:               ResourceLaw,
				CapabilityID:           "law.update.list",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindLawUpdates,
				LogicalInput:           mustLawUpdateIntent(t),
			},
		},
		"裁判例検索": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-6",
				Task:                   TaskSearch,
				Resource:               ResourceJudicialDecision,
				CapabilityID:           "judicial-decision.search",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindJudicialDecisionSearch,
				LogicalInput:           mustJudicialSearchIntent(t),
			},
		},
		"裁判例読取り": {
			values: LegalQueryCandidateStepValues{
				StepID:                 "step-7",
				Task:                   TaskRead,
				Resource:               ResourceJudicialDecision,
				CapabilityID:           "judicial-decision.read",
				CapabilityMajorVersion: 1,
				InputKind:              InputKindJudicialDecisionRead,
				LogicalInput:           mustJudicialReadIntent(t, judicialRef),
			},
		},
	}
}

func validCandidateValues(t *testing.T) LegalQueryCandidateValues {
	t.Helper()
	return LegalQueryCandidateValues{
		CandidateID:    "candidate-1",
		SemanticScore:  600,
		Confidence:     ConfidenceMedium,
		EvidenceCodes:  []EvidenceCode{EvidenceOfficialAlias},
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  []string{},
		Steps:          []LegalQueryCandidateStep{mustValidStep(t, "法令検索")},
	}
}

func mustValidStep(t *testing.T, name string) LegalQueryCandidateStep {
	t.Helper()
	step, err := NewLegalQueryCandidateStep(validStepFixtures(t)[name].values)
	if err != nil {
		t.Fatalf("試験用 step を作成できません: %v", err)
	}
	return step
}

func mustLawSearchIntent(t *testing.T) LawSearchIntentV1 {
	t.Helper()
	intent, err := NewLawSearchIntentV1(LawSearchIntentV1Values{Query: "行政手続法"})
	if err != nil {
		t.Fatalf("試験用 LawSearchIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustLawContentSearchIntent(t *testing.T) LawContentSearchIntentV1 {
	t.Helper()
	intent, err := NewLawContentSearchIntentV1(
		LawContentSearchIntentV1Values{AllTerms: []string{"永住許可"}},
	)
	if err != nil {
		t.Fatalf("試験用 LawContentSearchIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustLawReadIntent(t *testing.T, ref model.SourceResourceRef) LawReadIntentV1 {
	t.Helper()
	intent, err := NewLawReadIntentV1(LawReadIntentV1Values{Ref: &ref})
	if err != nil {
		t.Fatalf("試験用 LawReadIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustLawArticleReadIntent(
	t *testing.T,
	ref model.SourceResourceRef,
) LawArticleReadIntentV1 {
	t.Helper()
	intent, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
		Ref:      &ref,
		Location: mustLawArticleLocation(t),
	})
	if err != nil {
		t.Fatalf("試験用 LawArticleReadIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustLawUpdateIntent(t *testing.T) LawUpdateListIntentV1 {
	t.Helper()
	intent, err := NewLawUpdateListIntentV1(
		LawUpdateListIntentV1Values{Date: mustDate(t, "2026-07-27")},
	)
	if err != nil {
		t.Fatalf("試験用 LawUpdateListIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustJudicialSearchIntent(t *testing.T) JudicialDecisionSearchIntentV1 {
	t.Helper()
	intent, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "永住許可"},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionSearchIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustJudicialReadIntent(
	t *testing.T,
	ref model.SourceResourceRef,
) JudicialDecisionReadIntentV1 {
	t.Helper()
	intent, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: ref},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionReadIntentV1 を作成できません: %v", err)
	}
	return intent
}

func mustConceptSource(t *testing.T) LegalConceptSource {
	t.Helper()
	source, err := NewLegalConceptSource(LegalConceptSourceValues{
		ConceptID:   "permanent-residence",
		Title:       "公的資料",
		URL:         "https://example.go.jp/reference",
		ConfirmedOn: mustDate(t, "2026-07-27"),
	})
	if err != nil {
		t.Fatalf("試験用 LegalConceptSource を作成できません: %v", err)
	}
	return source
}

func withStepID(
	values LegalQueryCandidateStepValues,
	value string,
) LegalQueryCandidateStepValues {
	values.StepID = value
	return values
}

func withTask(
	values LegalQueryCandidateStepValues,
	value Task,
) LegalQueryCandidateStepValues {
	values.Task = value
	return values
}

func withResource(
	values LegalQueryCandidateStepValues,
	value Resource,
) LegalQueryCandidateStepValues {
	values.Resource = value
	return values
}

func withCapabilityID(
	values LegalQueryCandidateStepValues,
	value string,
) LegalQueryCandidateStepValues {
	values.CapabilityID = value
	return values
}

func withCapabilityMajor(
	values LegalQueryCandidateStepValues,
	value int,
) LegalQueryCandidateStepValues {
	values.CapabilityMajorVersion = value
	return values
}

func withInputKind(
	values LegalQueryCandidateStepValues,
	value LogicalInputKind,
) LegalQueryCandidateStepValues {
	values.InputKind = value
	return values
}

func withLogicalInput(
	values LegalQueryCandidateStepValues,
	value LogicalInput,
) LegalQueryCandidateStepValues {
	values.LogicalInput = value
	return values
}

func withConceptID(
	values LegalConceptSourceValues,
	value string,
) LegalConceptSourceValues {
	values.ConceptID = value
	return values
}

func withConceptTitle(
	values LegalConceptSourceValues,
	value string,
) LegalConceptSourceValues {
	values.Title = value
	return values
}

func withConceptURL(
	values LegalConceptSourceValues,
	value string,
) LegalConceptSourceValues {
	values.URL = value
	return values
}

func withConfirmedOn(
	values LegalConceptSourceValues,
	value model.Date,
) LegalConceptSourceValues {
	values.ConfirmedOn = value
	return values
}

func assertPanics(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("内部不変条件の破損を通知せず処理を継続しました")
		}
	}()
	operation()
}
