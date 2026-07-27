package legalquerycorpus

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSemanticCaseは有効なplanとrequestErrorを保持する(t *testing.T) {
	t.Parallel()

	planCase := mustSemanticCaseFromMap(t, validSemanticCase(validLawSearchStep()))
	if planCase.ArtifactKind() != ArtifactKindSemanticCase ||
		planCase.SchemaVersion() != 1 ||
		planCase.CaseID() != "development-law-search" ||
		planCase.LeakageGroupID() != "law-search" {
		t.Fatalf("SOT-ENG-026: semantic case header = %#v", planCase)
	}
	if got := planCase.CoverageIDs(); len(got) != 1 || got[0] != "intent-law-search" {
		t.Fatalf("SOT-ENG-026: coverageIds = %#v", got)
	}
	if _, exists := planCase.SafetyVariant(); exists {
		t.Fatal("SOT-ENG-026: 通常 case に safetyVariant が存在する")
	}
	if got := planCase.EnabledPacks(); len(got) != 0 {
		t.Fatalf("SOT-ENG-026: enabledPacks = %#v", got)
	}
	if planCase.Request().Query() != "行政手続法を検索" {
		t.Fatalf("SOT-ENG-026: request = %#v", planCase.Request())
	}
	if expected := planCase.Expected(); expected.Kind() != SemanticExpectedKindPlan {
		t.Fatalf("SOT-ENG-026: expected kind = %q", expected.Kind())
	}

	requestError := validSemanticCase(validLawSearchStep())
	requestError["caseId"] = "holdout-query-empty"
	requestError["coverageIds"] = []any{"input-query-empty"}
	requestError["request"] = map[string]any{"query": ""}
	requestError["expected"] = map[string]any{
		"kind":      "request_error",
		"errorCode": "invalid_argument",
		"field":     "query",
	}
	errorCase := mustSemanticCaseFromMap(t, requestError)
	if errorCase.CaseID() != "holdout-query-empty" ||
		errorCase.Expected().Kind() != SemanticExpectedKindRequestError {
		t.Fatalf("SOT-ENG-026: request error case = %#v", errorCase)
	}
}

func TestSemanticCaseはheaderとcatalogの違反を拒否する(t *testing.T) {
	t.Parallel()

	tooManyPacks := make([]string, 65)
	for index := range tooManyPacks {
		tooManyPacks[index] = fmt.Sprintf("pack-%03d", index)
	}
	tests := map[string]func(SemanticCaseValues) SemanticCaseValues{
		"artifactKind": func(values SemanticCaseValues) SemanticCaseValues {
			values.ArtifactKind = ArtifactKindCorpusManifest
			return values
		},
		"schemaVersion": func(values SemanticCaseValues) SemanticCaseValues {
			values.SchemaVersion = 2
			return values
		},
		"execution caseId": func(values SemanticCaseValues) SemanticCaseValues {
			values.CaseID = "execution-law-search"
			return values
		},
		"caseId形式": func(values SemanticCaseValues) SemanticCaseValues {
			values.CaseID = "development_Law"
			return values
		},
		"leakageGroupId": func(values SemanticCaseValues) SemanticCaseValues {
			values.LeakageGroupID = "Law Search"
			return values
		},
		"coverage空": func(values SemanticCaseValues) SemanticCaseValues {
			values.CoverageIDs = []string{}
			return values
		},
		"coverage未知": func(values SemanticCaseValues) SemanticCaseValues {
			values.CoverageIDs = []string{"unknown-coverage"}
			return values
		},
		"coverage順序": func(values SemanticCaseValues) SemanticCaseValues {
			values.CoverageIDs = []string{"intent-law-search", "concept-single"}
			return values
		},
		"coverage重複": func(values SemanticCaseValues) SemanticCaseValues {
			values.CoverageIDs = []string{"intent-law-search", "intent-law-search"}
			return values
		},
		"enabled pack形式": func(values SemanticCaseValues) SemanticCaseValues {
			values.EnabledPacks = []string{"Judicial"}
			return values
		},
		"enabled pack順序": func(values SemanticCaseValues) SemanticCaseValues {
			values.EnabledPacks = []string{"tax", "judicial-cases"}
			return values
		},
		"enabled pack重複": func(values SemanticCaseValues) SemanticCaseValues {
			values.EnabledPacks = []string{"tax", "tax"}
			return values
		},
		"enabled pack上限": func(values SemanticCaseValues) SemanticCaseValues {
			values.EnabledPacks = tooManyPacks
			return values
		},
		"request未初期化": func(values SemanticCaseValues) SemanticCaseValues {
			values.Request = Request{}
			return values
		},
		"expectedなし": func(values SemanticCaseValues) SemanticCaseValues {
			values.Expected = nil
			return values
		},
		"expected pointer": func(values SemanticCaseValues) SemanticCaseValues {
			plan := values.Expected.(ExpectedPlan)
			values.Expected = &plan
			return values
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := validSemanticCaseValues(t)
			if _, err := NewSemanticCase(mutate(values)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な semantic case を受理した")
			}
		})
	}
	if err := (SemanticCase{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: SemanticCase の zero value を受理した")
	}
}

func TestSemanticCaseはsafetyCoverageとvariantを一致させる(t *testing.T) {
	t.Parallel()

	for _, coverageID := range safetyCoverageIDs() {
		coverageID := coverageID
		t.Run(coverageID, func(t *testing.T) {
			t.Parallel()
			values := validSemanticCaseValues(t)
			values.CoverageIDs = []string{coverageID}
			if _, err := NewSemanticCase(values); err == nil {
				t.Fatal("SOT-ENG-026: safetyVariant 欠落を受理した")
			}
			variant := SafetyVariantOrdinary
			values.SafetyVariant = &variant
			semanticCase, err := NewSemanticCase(values)
			if err != nil {
				t.Fatalf("SOT-ENG-026: safety case error = %v", err)
			}
			if got, exists := semanticCase.SafetyVariant(); !exists || got != variant {
				t.Fatalf("SOT-ENG-026: safetyVariant = (%q, %t)", got, exists)
			}
		})
	}

	values := validSemanticCaseValues(t)
	unexpected := SafetyVariantAdversarial
	values.SafetyVariant = &unexpected
	if _, err := NewSemanticCase(values); err == nil {
		t.Fatal("SOT-ENG-026: 非 safety case の safetyVariant を受理した")
	}
	values.CoverageIDs = []string{"boundary-budget-limit"}
	unknown := SafetyVariant("unknown")
	values.SafetyVariant = &unknown
	if _, err := NewSemanticCase(values); err == nil {
		t.Fatal("SOT-ENG-026: 未定義 safetyVariant を受理した")
	}
}

func TestSemanticCaseはrequest境界とexpectedを一致させる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     Request
		expected    SemanticExpected
		shouldError bool
	}{
		{
			name:     "有効requestとplan",
			request:  mustRawRequest(t, "行政手続法を検索", nil, nil),
			expected: mustExpectedPlan(t, "single"),
		},
		{
			name:        "不正requestとplan",
			request:     mustRawRequest(t, "", nil, nil),
			expected:    mustExpectedPlan(t, "single"),
			shouldError: true,
		},
		{
			name:     "query error",
			request:  mustRawRequest(t, "", nil, nil),
			expected: mustExpectedRequestError(t, RequestErrorFieldQuery),
		},
		{
			name: "queryはrefより先に判定",
			request: mustRawRequest(
				t,
				"",
				invalidRawRequestRef(t),
				nil,
			),
			expected: mustExpectedRequestError(t, RequestErrorFieldQuery),
		},
		{
			name: "limitはrefより先に判定",
			request: mustRawRequest(
				t,
				"行政手続法を検索",
				invalidRawRequestRef(t),
				intPointer(0),
			),
			expected: mustExpectedRequestError(t, RequestErrorFieldLimitPerAttempt),
		},
		{
			name: "ref error",
			request: mustRawRequest(
				t,
				"行政手続法を検索",
				invalidRawRequestRef(t),
				nil,
			),
			expected: mustExpectedRequestError(t, RequestErrorFieldRef),
		},
		{
			name:        "request error field不一致",
			request:     mustRawRequest(t, "", nil, nil),
			expected:    mustExpectedRequestError(t, RequestErrorFieldRef),
			shouldError: true,
		},
		{
			name:        "有効requestにrequest error",
			request:     mustRawRequest(t, "行政手続法を検索", nil, nil),
			expected:    mustExpectedRequestError(t, RequestErrorFieldQuery),
			shouldError: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := validSemanticCaseValues(t)
			values.Request = test.request
			values.Expected = test.expected
			_, err := NewSemanticCase(values)
			if test.shouldError && err == nil {
				t.Fatal("SOT-ENG-026: request と expected の不整合を受理した")
			}
			if !test.shouldError && err != nil {
				t.Fatalf("SOT-ENG-026: request と expected の整合 error = %v", err)
			}
		})
	}
}

func TestSemanticCaseは選択意味のpackAvailabilityを導出する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		decision     string
		enabledPacks []string
		shouldError  bool
	}{
		{name: "single available", decision: "single", enabledPacks: []string{"judicial-cases"}},
		{name: "single disabled", decision: "single", shouldError: true},
		{name: "hedged available", decision: "hedged", enabledPacks: []string{"judicial-cases"}},
		{name: "hedged一部disabled", decision: "hedged", shouldError: true},
		{
			name:         "clarification available",
			decision:     "needs_clarification",
			enabledPacks: []string{"judicial-cases"},
		},
		{name: "clarification disabled", decision: "needs_clarification", shouldError: true},
		{name: "capability unavailable", decision: "capability_unavailable"},
		{
			name:         "capabilityがavailable",
			decision:     "capability_unavailable",
			enabledPacks: []string{"judicial-cases"},
			shouldError:  true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			expected := planWithSelectedJudicialMeaning(t, test.decision)
			values := validSemanticCaseValues(t)
			values.Expected = expected
			values.EnabledPacks = test.enabledPacks
			_, err := NewSemanticCase(values)
			if test.shouldError && err == nil {
				t.Fatal("SOT-ENG-026: pack availability の不整合を受理した")
			}
			if !test.shouldError && err != nil {
				t.Fatalf("SOT-ENG-026: pack availability error = %v", err)
			}
		})
	}
}

func TestSemanticCaseはconstructorとgetterで深く複製する(t *testing.T) {
	t.Parallel()

	values := validSemanticCaseValues(t)
	coverageIDs := []string{"concept-single", "intent-law-search"}
	enabledPacks := []string{"judicial-cases"}
	values.CoverageIDs = coverageIDs
	values.EnabledPacks = enabledPacks
	values.Expected = planWithSelectedJudicialMeaning(t, "single")
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewSemanticCase() error = %v", err)
	}
	coverageIDs[0] = "changed"
	enabledPacks[0] = "changed"
	values.Request.query = "changed"
	values.Expected.(ExpectedPlan).reasonCodes[0] = legalquery.ReasonCodeAmbiguousCandidates

	gotCoverage := semanticCase.CoverageIDs()
	gotPacks := semanticCase.EnabledPacks()
	gotRequest := semanticCase.Request()
	gotExpected := semanticCase.Expected().(ExpectedPlan)
	gotCoverage[0] = "changed-again"
	gotPacks[0] = "changed-again"
	gotRequest.query = "changed-again"
	gotExpected.reasonCodes[0] = legalquery.ReasonCodeAmbiguousCandidates

	if semanticCase.CoverageIDs()[0] != "concept-single" ||
		semanticCase.EnabledPacks()[0] != "judicial-cases" ||
		semanticCase.Request().Query() != "行政手続法を検索" ||
		semanticCase.Expected().(ExpectedPlan).ReasonCodes()[0] !=
			legalquery.ReasonCodeSingleClearCandidate {
		t.Fatal("SOT-ENG-026: SemanticCase の状態が外部から変更された")
	}
}

func TestSemanticCaseGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	values := validSemanticCaseValues(t)
	values.Expected = planWithSelectedJudicialMeaning(t, "single")
	values.EnabledPacks = []string{"judicial-cases"}
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewSemanticCase() error = %v", err)
	}

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				semanticCase.CoverageIDs()[0] = "changed"
				semanticCase.EnabledPacks()[0] = "changed"
				request := semanticCase.Request()
				request.query = "changed"
				expected := semanticCase.Expected().(ExpectedPlan)
				expected.reasonCodes[0] = legalquery.ReasonCodeAmbiguousCandidates
			}
		}()
	}
	wait.Wait()
	if semanticCase.CoverageIDs()[0] != "intent-law-search" ||
		semanticCase.EnabledPacks()[0] != "judicial-cases" ||
		semanticCase.Request().Query() != "行政手続法を検索" {
		t.Fatal("SOT-ENG-026: 並行 getter が SemanticCase を変更した")
	}
}

func TestSemanticCaseはJSONから直接復元できない(t *testing.T) {
	t.Parallel()

	if err := json.Unmarshal([]byte(`{}`), &SemanticCase{}); err == nil {
		t.Fatal("SOT-ENG-026: SemanticCase を直接 JSON 復元できた")
	}
}

func validSemanticCaseValues(t *testing.T) SemanticCaseValues {
	t.Helper()
	return SemanticCaseValues{
		ArtifactKind:   ArtifactKindSemanticCase,
		SchemaVersion:  1,
		CaseID:         "development-law-search",
		LeakageGroupID: "law-search",
		CoverageIDs:    []string{"intent-law-search"},
		EnabledPacks:   []string{},
		Request:        mustRawRequest(t, "行政手続法を検索", nil, intPointer(20)),
		Expected:       mustExpectedPlan(t, "single"),
	}
}

func mustRawRequest(
	t *testing.T,
	query string,
	ref *RequestRef,
	limit *int,
) Request {
	t.Helper()
	request, err := NewRequest(RequestValues{
		Query:           query,
		Ref:             ref,
		LimitPerAttempt: limit,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: raw Request を作成できません: %v", err)
	}
	return request
}

func invalidRawRequestRef(t *testing.T) *RequestRef {
	t.Helper()
	key, err := NewRequestKey(RequestKeyValues{
		SourceID:     "",
		ResourceType: "",
		ResourceID:   "",
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: raw key を作成できません: %v", err)
	}
	ref, err := NewRequestRef(RequestRefValues{ProviderID: "", Key: key})
	if err != nil {
		t.Fatalf("SOT-ENG-026: raw ref を作成できません: %v", err)
	}
	return &ref
}

func mustExpectedPlan(t *testing.T, decision string) ExpectedPlan {
	t.Helper()
	expected, err := decodeSemanticExpectedV1(
		mustJSONBytes(t, validExpectedPlansForAllDecisions()[decision]),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: ExpectedPlan を作成できません: %v", err)
	}
	return expected.(ExpectedPlan)
}

func mustExpectedRequestError(
	t *testing.T,
	field RequestErrorField,
) ExpectedRequestError {
	t.Helper()
	expected, err := NewExpectedRequestError(ExpectedRequestErrorValues{
		ErrorCode: model.ErrorCodeInvalidArgument,
		Field:     field,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: ExpectedRequestError を作成できません: %v", err)
	}
	return expected
}

func planWithSelectedJudicialMeaning(t *testing.T, decision string) ExpectedPlan {
	t.Helper()
	source := validExpectedPlansForAllDecisions()[decision]
	if decision == "single" {
		meaning := source["meanings"].([]any)[0].(map[string]any)
		meaning["meaningId"] = "judicial-search"
		meaning["requiredPacks"] = []any{"judicial-cases"}
		meaning["steps"] = []any{validJudicialSearchStep()}
		source["selectedMeaningIds"] = []any{"judicial-search"}
	}
	if decision == "hedged" || decision == "needs_clarification" {
		meanings := source["meanings"].([]any)
		first := meanings[0].(map[string]any)
		first["meaningId"] = "judicial-search"
		first["requiredPacks"] = []any{"judicial-cases"}
		first["steps"] = []any{validJudicialSearchStep()}
		if decision == "hedged" {
			source["selectedMeaningIds"] = []any{
				"judicial-search",
				meanings[1].(map[string]any)["meaningId"],
			}
		} else {
			source["selectedMeaningIds"] = []any{"judicial-search"}
		}
	}
	expected, err := decodeSemanticExpectedV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: pack plan を作成できません: %v", err)
	}
	return expected.(ExpectedPlan)
}

func intPointer(value int) *int {
	return &value
}
