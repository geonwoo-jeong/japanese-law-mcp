package provideronboarding

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplacementsUseDirectActiveSuccessors(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeSOTReferenceFixture(
		t,
		repository,
		"廃止",
		"- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
		"有効",
		false,
	)
	direct, err := replacementsUseDirectActiveSuccessors(
		repository,
		[]interfaceSOTReferenceReplacement{{
			previousSOTID: "SOT-IF-061",
			currentSOTID:  "SOT-IF-077",
		}},
		make(map[string]sotDocument),
	)
	if err != nil {
		t.Fatalf("直接後継を確認できませんでした: %v", err)
	}
	if !direct {
		t.Fatal("廃止 SOT から有効な直接後継への置換が拒否されました")
	}
}

func TestReplacementsSupportDeliverySOTSuccessors(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeTestFile(
		t,
		repository,
		"sot/60-delivery/01-previous.md",
		"# SOT-DEL-001: 旧規定\n\n- 状態: 廃止\n"+
			"- 後継: [SOT-DEL-002: 後継規定](02-current.md)\n\n"+
			"## 規定\n\n旧規定。\n",
	)
	writeTestFile(
		t,
		repository,
		"sot/60-delivery/02-current.md",
		"# SOT-DEL-002: 後継規定\n\n- 状態: 有効\n\n## 規定\n\n後継規定。\n",
	)
	writeTestFile(
		t,
		repository,
		"sot/60-delivery/00-index.md",
		"# 提供 SOT\n\n- [01-previous.md](01-previous.md)\n"+
			"- [02-current.md](02-current.md)\n",
	)
	direct, err := replacementsUseDirectActiveSuccessors(
		repository,
		[]interfaceSOTReferenceReplacement{{
			previousSOTID: "SOT-DEL-001",
			currentSOTID:  "SOT-DEL-002",
		}},
		make(map[string]sotDocument),
	)
	if err != nil {
		t.Fatalf("提供 SOT の直接後継を確認できませんでした: %v", err)
	}
	if !direct {
		t.Fatal("提供 SOT の直接後継が拒否されました")
	}
}

func TestReplacementsDoNotTreatSemanticReferenceChangesAsTraceabilityOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		previousState string
		successorLine string
		currentState  string
	}{
		{
			name:          "previous is active",
			previousState: "有効",
			currentState:  "有効",
		},
		{
			name:          "successor differs",
			previousState: "廃止",
			successorLine: "- 後継: [SOT-IF-078: 別の後継](77-current.md)\n",
			currentState:  "有効",
		},
		{
			name:          "current is draft",
			previousState: "廃止",
			successorLine: "- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
			currentState:  "草案",
		},
		{
			name:          "current is retired",
			previousState: "廃止",
			successorLine: "- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
			currentState:  "廃止",
		},
		{
			name:          "successor link target differs",
			previousState: "廃止",
			successorLine: "- 後継: [SOT-IF-077: 後継規定](78-other.md)\n",
			currentState:  "有効",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := t.TempDir()
			writeSOTReferenceFixture(
				t,
				repository,
				test.previousState,
				test.successorLine,
				test.currentState,
				false,
			)
			direct, err := replacementsUseDirectActiveSuccessors(
				repository,
				[]interfaceSOTReferenceReplacement{{
					previousSOTID: "SOT-IF-061",
					currentSOTID:  "SOT-IF-077",
				}},
				make(map[string]sotDocument),
			)
			if err != nil {
				t.Fatalf("意味変更の分類で予期しないエラーです: %v", err)
			}
			if direct {
				t.Fatal("意味上の SOT 参照変更が追跡可能性だけの更新になりました")
			}
		})
	}
}

func TestReplacementsFailClosedForMalformedLifecycleMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		successorLine string
		duplicateLink bool
	}{
		{
			name:          "malformed successor",
			successorLine: "- 後継: SOT-IF-077\n",
		},
		{
			name:          "duplicate index entry",
			successorLine: "- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
			duplicateLink: true,
		},
		{
			name: "duplicate successor",
			successorLine: "- 後継: [SOT-IF-077: 後継規定](77-current.md)\n" +
				"- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := t.TempDir()
			writeSOTReferenceFixture(
				t,
				repository,
				"廃止",
				test.successorLine,
				"有効",
				test.duplicateLink,
			)
			_, err := replacementsUseDirectActiveSuccessors(
				repository,
				[]interfaceSOTReferenceReplacement{{
					previousSOTID: "SOT-IF-061",
					currentSOTID:  "SOT-IF-077",
				}},
				make(map[string]sotDocument),
			)
			if err == nil {
				t.Fatal("不正な SOT lifecycle metadata が許可されました")
			}
		})
	}
}

func TestReplacementsFailClosedForMissingOrUnindexedSOT(t *testing.T) {
	t.Parallel()

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		repository := t.TempDir()
		writeSOTReferenceFixture(
			t,
			repository,
			"廃止",
			"- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
			"有効",
			false,
		)
		_, err := replacementsUseDirectActiveSuccessors(
			repository,
			[]interfaceSOTReferenceReplacement{{
				previousSOTID: "SOT-IF-060",
				currentSOTID:  "SOT-IF-077",
			}},
			make(map[string]sotDocument),
		)
		if err == nil {
			t.Fatal("存在しない旧 SOT が許可されました")
		}
	})

	t.Run("unindexed", func(t *testing.T) {
		t.Parallel()
		repository := t.TempDir()
		writeSOTReferenceFixture(
			t,
			repository,
			"廃止",
			"- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
			"有効",
			false,
		)
		writeTestFile(
			t,
			repository,
			"sot/40-interfaces/00-index.md",
			"# interface SOT\n\n- [77-current.md](77-current.md)\n",
		)
		_, err := replacementsUseDirectActiveSuccessors(
			repository,
			[]interfaceSOTReferenceReplacement{{
				previousSOTID: "SOT-IF-061",
				currentSOTID:  "SOT-IF-077",
			}},
			make(map[string]sotDocument),
		)
		if err == nil {
			t.Fatal("index にない旧 SOT が許可されました")
		}
	})
}

func TestClassifyTraceOnlyMatrixPathsUsesMergeBaseAndEffectiveSnapshot(t *testing.T) {
	t.Parallel()

	const matrixPath = "conformance/providers/provider-a.yaml"
	repository := newTestGitRepository(t, map[string]string{
		matrixPath: "before\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeSOTReferenceFixture(
		t,
		repository,
		"廃止",
		"- 後継: [SOT-IF-077: 後継規定](77-current.md)\n",
		"有効",
		false,
	)
	writeTestFile(t, repository, matrixPath, "after\n")

	client := newGitClient(repository)
	comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	changes, err := client.collectChanges(
		t.Context(),
		comparison,
		changeSources{workingTree: true, untracked: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	classified, err := classifyTraceOnlyMatrixPaths(
		t.Context(),
		client,
		repository,
		comparison,
		changes,
		func(gotRepository, providerID string, previous, current []byte) (
			interfaceSOTReferenceChange,
			bool,
			error,
		) {
			if gotRepository != resolvedTestPath(t, repository) && gotRepository != repository {
				t.Fatalf("repository = %q", gotRepository)
			}
			if providerID != "provider-a" || string(previous) != "before\n" ||
				string(current) != "after\n" {
				t.Fatalf(
					"classifier input = provider:%q before:%q after:%q",
					providerID,
					previous,
					current,
				)
			}
			return interfaceSOTReferenceChange{
				replacements: []interfaceSOTReferenceReplacement{{
					previousSOTID: "SOT-IF-061",
					currentSOTID:  "SOT-IF-077",
				}},
			}, true, nil
		},
	)
	if err != nil {
		t.Fatalf("追跡可能性だけの matrix を分類できませんでした: %v", err)
	}
	if _, exists := classified[matrixPath]; !exists || len(classified) != 1 {
		t.Fatalf("追跡可能性 matrix = %#v", classified)
	}
}

func TestClassifyTraceOnlyMatrixPathsPropagatesStrictComparisonFailure(t *testing.T) {
	t.Parallel()

	const matrixPath = "conformance/providers/provider-a.yaml"
	repository := newTestGitRepository(t, map[string]string{matrixPath: "before\n"})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(t, repository, matrixPath, "after\n")
	client := newGitClient(repository)
	comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	changes, err := client.collectChanges(
		t.Context(),
		comparison,
		changeSources{workingTree: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = classifyTraceOnlyMatrixPaths(
		t.Context(),
		client,
		repository,
		comparison,
		changes,
		func(string, string, []byte, []byte) (interfaceSOTReferenceChange, bool, error) {
			return interfaceSOTReferenceChange{}, false, errors.New("strict parse failure")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "strict parse failure") {
		t.Fatalf("strict comparison failure が伝播しませんでした: %v", err)
	}
}

func TestClassifyTraceOnlyMatrixPathsSkipsAddedAndDeletedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial map[string]string
		prepare func(*testing.T, string)
		sources changeSources
	}{
		{
			name:    "added",
			initial: map[string]string{"README.md": "base\n"},
			prepare: func(t *testing.T, repository string) {
				writeTestFile(
					t,
					repository,
					"conformance/providers/provider-a.yaml",
					"added\n",
				)
			},
			sources: changeSources{untracked: true},
		},
		{
			name: "deleted",
			initial: map[string]string{
				"conformance/providers/provider-a.yaml": "base\n",
			},
			prepare: func(t *testing.T, repository string) {
				if err := os.Remove(filepath.Join(
					repository,
					"conformance",
					"providers",
					"provider-a.yaml",
				)); err != nil {
					t.Fatal(err)
				}
			},
			sources: changeSources{workingTree: true},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := newTestGitRepository(t, test.initial)
			base := gitOutput(t, repository, "rev-parse", "HEAD")
			test.prepare(t, repository)
			client := newGitClient(repository)
			comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			changes, err := client.collectChanges(t.Context(), comparison, test.sources)
			if err != nil {
				t.Fatal(err)
			}
			classifierCalled := false
			classified, err := classifyTraceOnlyMatrixPaths(
				t.Context(),
				client,
				repository,
				comparison,
				changes,
				func(string, string, []byte, []byte) (
					interfaceSOTReferenceChange,
					bool,
					error,
				) {
					classifierCalled = true
					return interfaceSOTReferenceChange{}, true, nil
				},
			)
			if err != nil {
				t.Fatalf("file の追加削除を分類できませんでした: %v", err)
			}
			if classifierCalled || len(classified) != 0 {
				t.Fatalf(
					"追加削除で classifier が呼ばれました: called=%t paths=%#v",
					classifierCalled,
					classified,
				)
			}
		})
	}
}

func writeSOTReferenceFixture(
	t *testing.T,
	repository, previousState, successorLine, currentState string,
	duplicatePreviousIndexLink bool,
) {
	t.Helper()
	previous := "# SOT-IF-061: 旧規定\n\n- 状態: " + previousState + "\n" +
		successorLine + "\n## 規定\n\n旧規定。\n"
	current := "# SOT-IF-077: 後継規定\n\n- 状態: " + currentState +
		"\n\n## 規定\n\n後継規定。\n"
	index := "# interface SOT\n\n- [61-previous.md](61-previous.md)\n" +
		"- [77-current.md](77-current.md)\n"
	if duplicatePreviousIndexLink {
		index += "- [重複](61-previous.md)\n"
	}
	writeTestFile(t, repository, "sot/40-interfaces/61-previous.md", previous)
	writeTestFile(t, repository, "sot/40-interfaces/77-current.md", current)
	writeTestFile(t, repository, "sot/40-interfaces/00-index.md", index)
}
