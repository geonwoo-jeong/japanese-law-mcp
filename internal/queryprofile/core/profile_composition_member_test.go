package core

import (
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProfileContributionは明示したcore候補をcompositionMemberにする(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と商法をそれぞれ検索してください。"
	generation := generateQuery(t, query, nil)
	if generation.SelectionMode() != legalquery.QuerySelectionModeAutomatic ||
		len(generation.HedgePairs()) != 0 {
		t.Fatalf(
			"SOT-MODEL-028: selection/hedge = %q/%#v",
			generation.SelectionMode(),
			generation.HedgePairs(),
		)
	}
	candidates := generation.Candidates()
	members := generation.CompositionMembers()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 2 ||
		len(members) != 1 {
		t.Fatalf(
			"SOT-MODEL-028: candidates/members = %#v/%#v",
			candidates,
			members,
		)
	}
	member := members[0]
	if member.CandidateID() != candidates[0].CandidateID() ||
		member.Role() !=
			legalquery.QueryCandidateCompositionRoleRequiredMember {
		t.Fatalf("SOT-MODEL-028: member = %#v", member)
	}
	origins := member.StepOrigins()
	steps := candidates[0].Steps()
	if len(origins) != len(steps) {
		t.Fatalf("SOT-MODEL-028: step origins = %#v", origins)
	}
	for index, wantStart := range []int{
		strings.Index(query, "民法"),
		strings.Index(query, "商法"),
	} {
		if origins[index].StepID() != steps[index].StepID() ||
			origins[index].SourceStartByte() != wantStart {
			t.Fatalf(
				"SOT-MODEL-028: origins[%d] = %#v, want start=%d",
				index,
				origins[index],
				wantStart,
			)
		}
	}
}

func TestProfileContributionは裁判例ref読取り語をcore検索語へ混入しない(
	t *testing.T,
) {
	t.Parallel()

	const query = "この裁判例参照を読み、「駅構内転倒」を含む条文と裁判例をそれぞれ検索してください。"
	ref := mustCoreCompositionJudicialRef(t)
	generation := generateQuery(t, query, &ref)
	if generation.SelectionMode() != legalquery.QuerySelectionModeAutomatic ||
		len(generation.HedgePairs()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: selection/hedge = %q/%#v",
			generation.SelectionMode(),
			generation.HedgePairs(),
		)
	}
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-027: core candidates = %#v, signals=%#v mode=%q constraint=%q",
			candidates,
			generation.Signals(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
	step := candidates[0].Steps()[0]
	content, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"駅構内転倒"}) ||
		len(content.AnyTerms()) != 0 ||
		len(content.ExcludeTerms()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: core content input = %#v",
			step.LogicalInput(),
		)
	}
	members := generation.CompositionMembers()
	if len(members) != 1 ||
		members[0].CandidateID() != candidates[0].CandidateID() {
		t.Fatalf("SOT-MODEL-028: composition members = %#v", members)
	}
	origins := members[0].StepOrigins()
	wantStart := strings.Index(query, "駅構内転倒")
	if len(origins) != 1 ||
		origins[0].StepID() != step.StepID() ||
		origins[0].SourceStartByte() != wantStart {
		t.Fatalf(
			"SOT-MODEL-028: step origins = %#v, want start=%d",
			origins,
			wantStart,
		)
	}
}

func TestProfileContributionは検索後の裁判例Ref読取りでも検索語を保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "裁判例で駅構内転倒を含む条文を検索してこの裁判例参照を読んでください。"
	ref := mustCoreCompositionJudicialRef(t)
	generation := generateQuery(t, query, &ref)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-027: core candidates = %#v, signals=%#v mode=%q constraint=%q",
			candidates,
			generation.Signals(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	content, ok := logicalInput.(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"駅構内転倒"}) ||
		len(content.AnyTerms()) != 0 ||
		len(content.ExcludeTerms()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: core content input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileContributionは合成不適格候補のsidecarを区別する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name            string
		query           string
		wantMemberCount int
	}{
		{
			name:            "弱い一般語",
			query:           "制度について法情報を探したいです。",
			wantMemberCount: 0,
		},
		{
			name:            "hedgeする代替解釈",
			query:           "実行検証用に保証を法令名と条文の二候補で検索してください。",
			wantMemberCount: 2,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			generation := generateQuery(t, test.query, nil)
			if len(generation.CompositionMembers()) !=
				test.wantMemberCount {
				t.Fatalf(
					"SOT-MODEL-028: composition members = %#v, want %d",
					generation.CompositionMembers(),
					test.wantMemberCount,
				)
			}
		})
	}
}

func mustCoreCompositionJudicialRef(
	t *testing.T,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "95570/detail2",
	})
	if err != nil {
		t.Fatalf("試験用裁判例 ref key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用裁判例 ref を作成できません: %v", err)
	}
	return ref
}
