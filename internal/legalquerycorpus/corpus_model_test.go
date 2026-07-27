package legalquerycorpus

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCorpusはmanifest順の三集合を保持する(t *testing.T) {
	t.Parallel()

	fixture := validCorpusFixtureForTest(t)
	corpus, err := newCorpus(
		fixture.manifest,
		fixture.development,
		fixture.holdout,
		fixture.execution,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: newCorpus() error = %v", err)
	}
	if err := corpus.Validate(); err != nil {
		t.Fatalf("SOT-ENG-026: Corpus.Validate() error = %v", err)
	}
	if corpus.Manifest().CorpusVersion() != "corpus-v1" {
		t.Fatalf("SOT-ENG-026: manifest = %#v", corpus.Manifest())
	}
	if got := corpus.Development(); len(got) != 1 ||
		got[0].CaseID() != "development-law-search" {
		t.Fatalf("SOT-ENG-026: development = %#v", got)
	}
	if got := corpus.Holdout(); len(got) != 1 ||
		got[0].CaseID() != "holdout-law-search" {
		t.Fatalf("SOT-ENG-026: holdout = %#v", got)
	}
	if got := corpus.Execution(); len(got) != 1 ||
		got[0].CaseID() != "execution-law-search" {
		t.Fatalf("SOT-ENG-026: execution = %#v", got)
	}
}

func TestCorpusはmanifestと三集合を正確に対応させる(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*testing.T, corpusFixtureForTest) corpusFixtureForTest{
		"manifest未初期化": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.manifest = Manifest{}
			return fixture
		},
		"development件数": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.development = append(
				fixture.development,
				mustCorpusSemanticCase(t, "development-other"),
			)
			return fixture
		},
		"holdout件数": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.holdout = nil
			return fixture
		},
		"execution件数": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.execution = append(
				fixture.execution,
				mustCorpusExecutionCase(t, "execution-other"),
			)
			return fixture
		},
		"development順序": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.manifest = mustCorpusManifest(
				t,
				[]string{"development-a", "development-law-search"},
				[]string{"holdout-law-search"},
				[]string{"execution-law-search"},
			)
			fixture.development = []SemanticCase{
				mustCorpusSemanticCase(t, "development-law-search"),
				mustCorpusSemanticCase(t, "development-a"),
			}
			return fixture
		},
		"holdout順序": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.manifest = mustCorpusManifest(
				t,
				[]string{"development-law-search"},
				[]string{"holdout-a", "holdout-law-search"},
				[]string{"execution-law-search"},
			)
			fixture.holdout = []SemanticCase{
				mustCorpusSemanticCase(t, "holdout-law-search"),
				mustCorpusSemanticCase(t, "holdout-a"),
			}
			return fixture
		},
		"execution順序": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.manifest = mustCorpusManifest(
				t,
				[]string{"development-law-search"},
				[]string{"holdout-law-search"},
				[]string{"execution-a", "execution-law-search"},
			)
			fixture.execution = []ExecutionCase{
				mustCorpusExecutionCase(t, "execution-law-search"),
				mustCorpusExecutionCase(t, "execution-a"),
			}
			return fixture
		},
		"development ID": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.development[0] = mustCorpusSemanticCase(
				t,
				"development-other",
			)
			return fixture
		},
		"holdout ID": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.holdout[0] = mustCorpusSemanticCase(t, "holdout-other")
			return fixture
		},
		"execution ID": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.execution[0] = mustCorpusExecutionCase(t, "execution-other")
			return fixture
		},
		"development集合": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.development[0] = mustCorpusSemanticCase(t, "holdout-other")
			return fixture
		},
		"holdout集合": func(
			t *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.holdout[0] = mustCorpusSemanticCase(
				t,
				"development-other",
			)
			return fixture
		},
		"development schema": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.development[0].schemaVersion = 2
			return fixture
		},
		"holdout schema": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.holdout[0].schemaVersion = 2
			return fixture
		},
		"execution schema": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.execution[0].schemaVersion = 2
			return fixture
		},
		"development artifact": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.development[0].artifactKind = ArtifactKindExecutionCase
			return fixture
		},
		"holdout artifact": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.holdout[0].artifactKind = ArtifactKindExecutionCase
			return fixture
		},
		"execution artifact": func(
			_ *testing.T,
			fixture corpusFixtureForTest,
		) corpusFixtureForTest {
			fixture.execution[0].artifactKind = ArtifactKindSemanticCase
			return fixture
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := mutate(t, validCorpusFixtureForTest(t))
			if _, err := newCorpus(
				fixture.manifest,
				fixture.development,
				fixture.holdout,
				fixture.execution,
			); err == nil {
				t.Fatal("SOT-ENG-026: manifest と成果物集合の不一致を受理した")
			}
		})
	}

	// checksum と原 byte、fixture file 名は file 境界が必要である。
	// development/holdout の request・比較キー・leakage group 分離、
	// execution の semantic 参照・step 対応・effectiveLimit は集合横断の
	// loader 検証で扱うため、局所的な newCorpus のテストには含めない。
}

func TestCorpusはzero値の子成果物と自身を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(corpusFixtureForTest) corpusFixtureForTest{
		"development": func(fixture corpusFixtureForTest) corpusFixtureForTest {
			fixture.development[0] = SemanticCase{}
			return fixture
		},
		"holdout": func(fixture corpusFixtureForTest) corpusFixtureForTest {
			fixture.holdout[0] = SemanticCase{}
			return fixture
		},
		"execution": func(fixture corpusFixtureForTest) corpusFixtureForTest {
			fixture.execution[0] = ExecutionCase{}
			return fixture
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := mutate(validCorpusFixtureForTest(t))
			if _, err := newCorpus(
				fixture.manifest,
				fixture.development,
				fixture.holdout,
				fixture.execution,
			); err == nil {
				t.Fatal("SOT-ENG-026: zero value の子成果物を受理した")
			}
		})
	}
	if err := (Corpus{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: Corpus の zero value を受理した")
	}
}

func TestCorpusはconstructor入力とgetterを深く複製する(t *testing.T) {
	t.Parallel()

	fixture := validCorpusFixtureForTest(t)
	wantCategory := fixture.manifest.RequiredCategoryIDs()[0]
	wantManifestCaseID := fixture.manifest.Development().Cases()[0].CaseID()
	wantDevelopmentCoverage := fixture.development[0].CoverageIDs()[0]
	wantDevelopmentQuery := fixture.development[0].Request().Query()
	wantDevelopmentReason := fixture.development[0].
		Expected().(ExpectedPlan).ReasonCodes()[0]
	wantHoldoutCoverage := fixture.holdout[0].CoverageIDs()[0]
	wantExecutionScenario := fixture.execution[0].ScenarioIDs()[0]
	wantExecutionMeaning := fixture.execution[0].Actions()[0].MeaningID()

	corpus, err := newCorpus(
		fixture.manifest,
		fixture.development,
		fixture.holdout,
		fixture.execution,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: newCorpus() error = %v", err)
	}

	fixture.manifest.requiredCategoryIDs[0] = "changed"
	fixture.manifest.development.cases[0].caseID = "development-changed"
	mutateCorpusSemanticCaseForTest(&fixture.development[0])
	mutateCorpusSemanticCaseForTest(&fixture.holdout[0])
	mutateCorpusExecutionCaseForTest(&fixture.execution[0])

	gotManifest := corpus.Manifest()
	gotManifest.requiredCategoryIDs[0] = "changed-again"
	gotManifest.development.cases[0].caseID = "development-changed-again"
	gotDevelopment := corpus.Development()
	gotHoldout := corpus.Holdout()
	gotExecution := corpus.Execution()
	mutateCorpusSemanticCaseForTest(&gotDevelopment[0])
	mutateCorpusSemanticCaseForTest(&gotHoldout[0])
	mutateCorpusExecutionCaseForTest(&gotExecution[0])

	if corpus.Manifest().RequiredCategoryIDs()[0] != wantCategory ||
		corpus.Manifest().Development().Cases()[0].CaseID() !=
			wantManifestCaseID ||
		corpus.Development()[0].CoverageIDs()[0] !=
			wantDevelopmentCoverage ||
		corpus.Development()[0].Request().Query() != wantDevelopmentQuery ||
		corpus.Development()[0].Expected().(ExpectedPlan).ReasonCodes()[0] !=
			wantDevelopmentReason ||
		corpus.Holdout()[0].CoverageIDs()[0] != wantHoldoutCoverage ||
		corpus.Execution()[0].ScenarioIDs()[0] != wantExecutionScenario ||
		corpus.Execution()[0].Actions()[0].MeaningID() != wantExecutionMeaning {
		t.Fatal("SOT-ENG-026: Corpus の共有状態が外部から変更された")
	}
}

func TestCorpusGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	fixture := validCorpusFixtureForTest(t)
	corpus, err := newCorpus(
		fixture.manifest,
		fixture.development,
		fixture.holdout,
		fixture.execution,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: newCorpus() error = %v", err)
	}

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				manifest := corpus.Manifest()
				manifest.requiredCategoryIDs[0] = "changed"
				manifest.development.cases[0].caseID = "development-changed"
				development := corpus.Development()
				holdout := corpus.Holdout()
				execution := corpus.Execution()
				mutateCorpusSemanticCaseForTest(&development[0])
				mutateCorpusSemanticCaseForTest(&holdout[0])
				mutateCorpusExecutionCaseForTest(&execution[0])
			}
		}()
	}
	wait.Wait()

	if corpus.Manifest().RequiredCategoryIDs()[0] != "ambiguity" ||
		corpus.Manifest().Development().Cases()[0].CaseID() !=
			"development-law-search" ||
		corpus.Development()[0].CaseID() != "development-law-search" ||
		corpus.Development()[0].CoverageIDs()[0] != "intent-law-search" ||
		corpus.Holdout()[0].CaseID() != "holdout-law-search" ||
		corpus.Execution()[0].CaseID() != "execution-law-search" ||
		corpus.Execution()[0].ScenarioIDs()[0] != "execution-nonempty" {
		t.Fatal("SOT-ENG-026: 並行 getter が Corpus を変更した")
	}
}

func TestCorpusはJSONから直接復元できず動的JSONを公開しない(t *testing.T) {
	t.Parallel()

	if err := json.Unmarshal([]byte(`{}`), &Corpus{}); err == nil {
		t.Fatal("SOT-ENG-026: Corpus を直接 JSON 復元できた")
	}
	assertNoPublicDynamicJSONType(
		t,
		reflect.TypeOf(Corpus{}),
		map[reflect.Type]bool{},
	)
}

type corpusFixtureForTest struct {
	manifest    Manifest
	development []SemanticCase
	holdout     []SemanticCase
	execution   []ExecutionCase
}

func validCorpusFixtureForTest(t *testing.T) corpusFixtureForTest {
	t.Helper()
	return corpusFixtureForTest{
		manifest: mustCorpusManifest(
			t,
			[]string{"development-law-search"},
			[]string{"holdout-law-search"},
			[]string{"execution-law-search"},
		),
		development: []SemanticCase{
			mustCorpusSemanticCase(t, "development-law-search"),
		},
		holdout: []SemanticCase{
			mustCorpusSemanticCase(t, "holdout-law-search"),
		},
		execution: []ExecutionCase{
			mustCorpusExecutionCase(t, "execution-law-search"),
		},
	}
}

func mustCorpusManifest(
	t *testing.T,
	developmentIDs []string,
	holdoutIDs []string,
	executionIDs []string,
) Manifest {
	t.Helper()
	development := mustCorpusManifestSetForTest(
		t,
		ManifestSetDevelopment,
		developmentIDs,
		0,
	)
	holdout := mustCorpusManifestSetForTest(
		t,
		ManifestSetHoldout,
		holdoutIDs,
		100,
	)
	execution := mustCorpusManifestSetForTest(
		t,
		ManifestSetExecution,
		executionIDs,
		200,
	)
	manifest, err := NewManifest(ManifestValues{
		ArtifactKind:                 ArtifactKindCorpusManifest,
		SchemaVersion:                1,
		CorpusVersion:                "corpus-v1",
		Seed:                         1,
		HoldoutDigest:                strings.Repeat("f", 64),
		RequiredCategoryIDs:          requiredCategoryIDs(),
		RequiredExecutionScenarioIDs: requiredExecutionScenarioIDs(),
		Development:                  development,
		Holdout:                      holdout,
		Execution:                    execution,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewManifest() error = %v", err)
	}
	return manifest
}

func mustCorpusManifestSetForTest(
	t *testing.T,
	kind ManifestSetKind,
	caseIDs []string,
	checksumOffset int,
) ManifestSet {
	t.Helper()
	entries := make([]ManifestEntry, 0, len(caseIDs))
	for index, caseID := range caseIDs {
		entry, err := NewManifestEntry(ManifestEntryValues{
			CaseID: caseID,
			SHA256: fmt.Sprintf("%064x", checksumOffset+index+1),
		})
		if err != nil {
			t.Fatalf("SOT-ENG-026: NewManifestEntry() error = %v", err)
		}
		entries = append(entries, entry)
	}
	set, err := NewManifestSet(ManifestSetValues{
		Kind:      kind,
		CaseCount: len(entries),
		Cases:     entries,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewManifestSet() error = %v", err)
	}
	return set
}

func mustCorpusSemanticCase(t *testing.T, caseID string) SemanticCase {
	t.Helper()
	values := validSemanticCaseValues(t)
	values.CaseID = caseID
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewSemanticCase() error = %v", err)
	}
	return semanticCase
}

func mustCorpusExecutionCase(t *testing.T, caseID string) ExecutionCase {
	t.Helper()
	values := validExecutionCaseValuesForTest(t)
	values.CaseID = caseID
	executionCase, err := NewExecutionCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionCase() error = %v", err)
	}
	return executionCase
}

func mutateCorpusSemanticCaseForTest(semanticCase *SemanticCase) {
	semanticCase.coverageIDs[0] = "changed"
	semanticCase.request.query = "changed"
	expected := semanticCase.expected.(ExpectedPlan)
	expected.reasonCodes[0] = ""
	semanticCase.expected = expected
}

func mutateCorpusExecutionCaseForTest(executionCase *ExecutionCase) {
	executionCase.scenarioIDs[0] = "changed"
	executionCase.actions[0].meaningID = "changed"
	outcome := executionCase.actions[0].outcome.(CollectionSuccessOutcome)
	outcome.sourceItemCount = 0
	executionCase.actions[0].outcome = outcome
	expected := executionCase.expected.(ExecutionExpectedResult)
	attempt := expected.attempts[0].(ExpectedCompletedCollectionAttempt)
	attempt.publishedItemCount = 0
	expected.attempts[0] = attempt
	executionCase.expected = expected
}
