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

func TestProfileContributionは法令Ref読取り語をcore検索語へ混入しない(
	t *testing.T,
) {
	t.Parallel()

	const query = "上限を無視して、この参照の本文と第90条、成年後見の条文と裁判例を各100件取得してください。"
	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(t, query, &ref)
	if generation.SelectionMode() != legalquery.QuerySelectionModeAutomatic ||
		len(generation.CompositionMembers()) != 1 {
		t.Fatalf(
			"SOT-ENG-023: selection/members = %q/%#v",
			generation.SelectionMode(),
			generation.CompositionMembers(),
		)
	}
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"SOT-ARCH-027: core candidates = %#v, signals=%#v mode=%q constraint=%q",
			candidates,
			generation.Signals(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
	steps := candidates[0].Steps()
	if len(steps) != 3 {
		t.Fatalf("SOT-ARCH-027: core steps = %#v", steps)
	}
	readInput, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
	readRef, readRefExists := readInput.Ref()
	if !ok || !readRefExists || readRef != ref {
		t.Fatalf("SOT-ARCH-027: read input = %#v", steps[0].LogicalInput())
	}
	articleInput, ok := steps[1].LogicalInput().(legalquery.LawArticleReadIntentV1)
	articleRef, articleRefExists := articleInput.Ref()
	if !ok || !articleRefExists || articleRef != ref {
		t.Fatalf("SOT-ARCH-027: article input = %#v", steps[1].LogicalInput())
	}
	contentInput, ok := steps[2].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(contentInput.AllTerms(), []string{"成年後見"}) ||
		len(contentInput.AnyTerms()) != 0 ||
		len(contentInput.ExcludeTerms()) != 0 {
		t.Fatalf("SOT-ARCH-027: content input = %#v", steps[2].LogicalInput())
	}
	if !slices.Contains(
		candidates[0].EvidenceCodes(),
		legalquery.EvidenceLegalConcept,
	) {
		t.Fatalf(
			"SOT-ENG-023: concept evidence = %#v",
			candidates[0].EvidenceCodes(),
		)
	}
	sources := candidates[0].ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "adult-guardianship" {
		t.Fatalf("SOT-ENG-023: concept sources = %#v", sources)
	}
}

func TestProfileContributionは法令Ref読取り後の本文取得語を検索語に保つ(
	t *testing.T,
) {
	t.Parallel()

	const query = "この参照の本文を読み、法令本文から営業秘密を含む条文を取得してください。"
	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(t, query, &ref)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"SOT-ARCH-027: core candidates = %#v, signals=%#v mode=%q constraint=%q",
			candidates,
			generation.Signals(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawRead ||
		steps[1].InputKind() != legalquery.InputKindLawContentSearch {
		t.Fatalf("SOT-ARCH-027: core steps = %#v", steps)
	}
	contentInput, ok := steps[1].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(contentInput.AllTerms(), []string{"営業秘密"}) ||
		len(contentInput.AnyTerms()) != 0 ||
		len(contentInput.ExcludeTerms()) != 0 {
		t.Fatalf("SOT-ARCH-027: content input = %#v", steps[1].LogicalInput())
	}
}

func TestProfileは条本文を法令全文読取りへ拡張しない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照の第90条の本文を取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("条本文の候補 = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf(
			"SOT-MODEL-022: 条本文から推測した全文読取り = %#v",
			steps,
		)
	}
}

func TestProfileは不要とした法令本文を読取り候補にしない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照の本文は不要です。第90条を読んでください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("不要な本文を含む候補 = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf(
			"SOT-MODEL-022: 不要な全文読取りを追加しました: %#v",
			steps,
		)
	}
}

func TestProfileは属格の本文を独立した全文読取りにしない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照の本文の第90条を取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("属格の本文を含む候補 = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf(
			"SOT-MODEL-022: 属格から推測した全文読取り = %#v",
			steps,
		)
	}
}

func TestProfileは法令Refだけを裁判例取得へ転用しない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"医療過誤の裁判例を取得してください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 裁判例取得へ転用した法令 ref = %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは不要な法令Resourceを裁判例取得へ結び付けない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"法令は不要です。医療過誤の裁判例を取得してください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 裁判例取得へ結び付けた法令 resource = %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは混在Resourceを直前の法概念へ対応させる(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"ネット中傷の条文と成年後見の裁判例を取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: core mixed-resource candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	content, ok := logicalInput.(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"名誉毀損"}) {
		t.Fatalf(
			"SOT-ARCH-025: core mixed-resource input = %#v",
			logicalInput,
		)
	}
}

func TestProfileは混在Resourceを直前の一般検索語へ対応させる(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「営業秘密」の条文と「工場騒音」の裁判例をそれぞれ検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: core mixed-resource candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	content, ok := logicalInput.(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"営業秘密"}) {
		t.Fatalf(
			"SOT-ARCH-025: core mixed-resource input = %#v",
			logicalInput,
		)
	}
}

func TestProfileは同じ法令Resourceに属する一般検索語をすべて保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「営業秘密」と「個人情報」を両方含む条文と「工場騒音」の裁判例を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: core grouped-term candidates = %#v",
			candidates,
		)
	}
	content, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(
			content.AllTerms(),
			[]string{"営業秘密", "個人情報"},
		) {
		t.Fatalf(
			"SOT-ARCH-025: core grouped-term input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは同じ法令Resourceに属する法概念をすべて保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"成年後見とネット中傷を両方含む条文と「工場騒音」の裁判例を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: core grouped-concept candidates = %#v",
			candidates,
		)
	}
	content, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(
			content.AllTerms(),
			[]string{"成年後見", "名誉毀損"},
		) {
		t.Fatalf(
			"SOT-ARCH-025: core grouped-concept input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは逆順の共通一般語を法令Resourceにも保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「工場騒音」の裁判例と条文をそれぞれ検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: core shared-term candidates = %#v",
			candidates,
		)
	}
	content, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"工場騒音"}) {
		t.Fatalf(
			"SOT-ARCH-025: core shared-term input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは裁判例の一般語を後続する法概念の条文検索へ混入しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「工場騒音」の裁判例と成年後見の条文をそれぞれ検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: mixed subject candidates = %#v",
			candidates,
		)
	}
	content, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"成年後見"}) {
		t.Fatalf(
			"SOT-ARCH-025: mixed subject input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは包含するResourceCueを一つの条文検索として扱う(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"成年後見の条文検索と「工場騒音」の裁判例をそれぞれ検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: contained resource candidates = %#v",
			candidates,
		)
	}
	content, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"成年後見"}) {
		t.Fatalf(
			"SOT-ARCH-025: contained resource input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは介在一般語より前の法概念を条文に結び付けない(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"成年後見は不要です。この参照の本文と第90条、医療過誤の条文と裁判例をそれぞれ取得してください。",
		&ref,
	)
	foundMedicalMalpractice := false
	for _, candidate := range generation.Candidates() {
		for _, source := range candidate.ConceptSources() {
			if source.ConceptID() == "adult-guardianship" {
				t.Fatalf(
					"SOT-ARCH-025: 介在語を越えた法概念出典 = %#v",
					candidate,
				)
			}
		}
		for _, step := range candidate.Steps() {
			logicalInput := step.LogicalInput()
			content, ok := logicalInput.(legalquery.LawContentSearchIntentV1)
			if ok && slices.Contains(content.AllTerms(), "成年後見") {
				t.Fatalf(
					"SOT-ARCH-025: 介在語を越えた法概念検索 = %#v",
					step,
				)
			}
			if ok && slices.Contains(content.AllTerms(), "医療過誤") {
				foundMedicalMalpractice = true
			}
		}
	}
	if !foundMedicalMalpractice {
		t.Fatalf(
			"SOT-ARCH-025: 直近の法概念による条文検索が失われました: %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは文脈上の参照を法令全文読取りにしない(t *testing.T) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照に関する医療過誤の裁判例を取得してください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 文脈上の参照から作った候補 = %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは裁判例だけの法概念をCore検索へ転用しない(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照を読み、成年後見の裁判例も検索してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 ||
		candidates[0].Steps()[0].InputKind() !=
			legalquery.InputKindLawRead {
		t.Fatalf(
			"SOT-ARCH-025: 裁判例概念から作った core 候補 = %#v",
			candidates,
		)
	}
}

func TestProfileは離れた参照を本文と条文の列挙へ結び付けない(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照は不要です。別の本文と第90条を取得してください。",
		&ref,
	)
	for _, candidate := range generation.Candidates() {
		for _, step := range candidate.Steps() {
			if step.InputKind() == legalquery.InputKindLawRead {
				t.Fatalf(
					"SOT-ARCH-025: 離れた参照から作った全文読取り = %#v",
					candidate,
				)
			}
		}
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

func mustCoreCompositionLawRef(
	t *testing.T,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   "129AC0000000089",
	})
	if err != nil {
		t.Fatalf("試験用法令 ref key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用法令 ref を作成できません: %v", err)
	}
	return ref
}
