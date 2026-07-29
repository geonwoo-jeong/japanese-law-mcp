package judicialcases

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCoreとJudicialProfileは個別指定でも各法概念を直前Resourceへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"ネット中傷の条文と成年後見の裁判例をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindLawContentSearch,
			legalquery.InputKindJudicialDecisionSearch,
		},
		[]string{"名誉毀損"},
		"成年後見",
	)
}

func TestCoreとJudicialProfileは逆順の個別指定でも各法概念を直前Resourceへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"成年後見の裁判例とネット中傷の条文をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindJudicialDecisionSearch,
			legalquery.InputKindLawContentSearch,
		},
		[]string{"名誉毀損"},
		"成年後見",
	)
}

func TestCoreとJudicialProfileは共通主題の個別Resourceをともに保持する(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"成年後見の条文と裁判例をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindLawContentSearch,
			legalquery.InputKindJudicialDecisionSearch,
		},
		[]string{"成年後見"},
		"成年後見",
	)
}

func TestCoreとJudicialProfileは個別一般語も各Resourceへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"「営業秘密」の条文と「工場騒音」の裁判例をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindLawContentSearch,
			legalquery.InputKindJudicialDecisionSearch,
		},
		[]string{"営業秘密"},
		"工場騒音",
	)
}

func TestCoreとJudicialProfileは同じ法令Resourceの一般語をすべて保持する(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"「営業秘密」と「個人情報」を両方含む条文と「工場騒音」の裁判例を検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindLawContentSearch,
			legalquery.InputKindJudicialDecisionSearch,
		},
		[]string{"営業秘密", "個人情報"},
		"工場騒音",
	)
}

func TestCoreとJudicialProfileは一般語と法概念も各Resourceへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"「工場騒音」の裁判例と成年後見の条文をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindJudicialDecisionSearch,
			legalquery.InputKindLawContentSearch,
		},
		[]string{"成年後見"},
		"工場騒音",
	)
}

func TestCoreとJudicialProfileは逆順の共通一般語を両Resourceに保持する(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"「工場騒音」の裁判例と条文をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindJudicialDecisionSearch,
			legalquery.InputKindLawContentSearch,
		},
		[]string{"工場騒音"},
		"工場騒音",
	)
}

func TestCoreとJudicialProfileは同じ法令Resourceの法概念をすべて保持する(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"成年後見とネット中傷を両方含む条文と「工場騒音」の裁判例を検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindLawContentSearch,
			legalquery.InputKindJudicialDecisionSearch,
		},
		[]string{"成年後見", "名誉毀損"},
		"工場騒音",
	)
}

func TestCoreとJudicialProfileは比較正規化した包含ResourceCueを一つに扱う(
	t *testing.T,
) {
	t.Parallel()

	assertMixedResourceSelection(
		t,
		"「工場騒音」の裁 判例本文と成年後見の条文をそれぞれ検索してください。",
		[]legalquery.LogicalInputKind{
			legalquery.InputKindJudicialDecisionSearch,
			legalquery.InputKindLawContentSearch,
		},
		[]string{"成年後見"},
		"工場騒音",
	)
}

func assertMixedResourceSelection(
	t *testing.T,
	query string,
	wantKinds []legalquery.LogicalInputKind,
	wantLawTerms []string,
	wantJudicialQuery string,
) {
	t.Helper()

	request := mustMixedProfileRequest(t, query)
	runtime := newMixedProfileRuntime(t)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-025: individual plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != len(wantKinds) {
		t.Fatalf("SOT-ARCH-025: individual steps = %#v", steps)
	}
	for index, wantKind := range wantKinds {
		if steps[index].InputKind() != wantKind {
			t.Fatalf(
				"SOT-ARCH-025: individual steps[%d] = %#v",
				index,
				steps[index],
			)
		}
		switch input := steps[index].LogicalInput().(type) {
		case legalquery.LawContentSearchIntentV1:
			if !slices.Equal(input.AllTerms(), wantLawTerms) {
				t.Fatalf(
					"SOT-ARCH-025: individual law content input = %#v",
					input,
				)
			}
		case legalquery.JudicialDecisionSearchIntentV1:
			if input.Query() != wantJudicialQuery {
				t.Fatalf(
					"SOT-ARCH-025: individual judicial input = %#v",
					input,
				)
			}
		}
	}
}
