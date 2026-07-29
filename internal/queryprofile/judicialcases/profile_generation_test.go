package judicialcases

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileは引用語と形態素語を裁判例検索へ変換する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantQuery string
	}{
		{
			name:      "形態素語",
			query:     "医療過誤の裁判例を検索してください。",
			wantQuery: "医療過誤",
		},
		{
			name:      "引用語",
			query:     "「医療過誤」の判例を検索してください。",
			wantQuery: "医療過誤",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, nil)
			candidate := findSearchCandidate(t, generation, test.wantQuery)
			if !slices.Equal(
				candidate.RequiredPacks(),
				[]string{"judicial-cases"},
			) {
				t.Fatalf("required packs = %#v", candidate.RequiredPacks())
			}
			if !slices.Equal(
				candidate.EvidenceCodes(),
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceExplicitResource,
					legalquery.EvidenceMorphologicalContext,
				},
			) {
				t.Fatalf("evidence = %#v", candidate.EvidenceCodes())
			}
		})
	}
}

func TestProfileは事件番号surfaceを構造化参照検索へ保持する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		surface string
		search  string
	}{
		{
			surface: "平成25(オ)1079",
			search:  "平成25(オ)1079",
		},
		{
			surface: "令和4年（ネ）第１００３９号",
			search:  "令和4年(ネ)第10039号",
		},
		{
			surface: "平成26年特（わ）第914号",
			search:  "平成26年特(わ)第914号",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.surface, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(
				t,
				test.surface+" の裁判例を検索してください。",
				nil,
			)
			candidate := findSearchCandidate(t, generation, test.search)
			if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			}) {
				t.Fatalf("事件番号検索 evidence = %#v", candidate.EvidenceCodes())
			}
			for _, generated := range generation.Candidates() {
				if len(generated.Steps()) == 1 {
					input, ok := generated.Steps()[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
					if ok && input.Query() == test.search &&
						generated.CandidateID() != candidate.CandidateID() {
						t.Fatalf(
							"同じ事件番号候補を重複生成しました: %#v",
							generation.Candidates(),
						)
					}
				}
			}
		})
	}
}

func TestProfileは引用された事件番号を一つの構造化検索へ変換する(
	t *testing.T,
) {
	t.Parallel()

	const surface = "令和5年（受）第123号"
	const search = "令和5年(受)第123号"
	generation := generateQuery(
		t,
		"「"+surface+"」の判例を検索してください。",
		nil,
	)
	candidate := findSearchCandidate(t, generation, search)
	if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
	}) {
		t.Fatalf("引用事件番号 evidence = %#v", candidate.EvidenceCodes())
	}
	if len(generation.Candidates()) != 1 {
		t.Fatalf("引用事件番号 candidates = %#v", generation.Candidates())
	}
}

func TestProfileは事件番号だけからreadまたはrefを推測しない(t *testing.T) {
	t.Parallel()

	const surface = "平成25(オ)1079"
	generation := generateQuery(
		t,
		surface+" の裁判例本文を取得してください。",
		nil,
	)
	for _, candidate := range generation.Candidates() {
		for _, step := range candidate.Steps() {
			if step.InputKind() == legalquery.InputKindJudicialDecisionRead {
				t.Fatalf("事件番号から read を推測しました: %#v", candidate)
			}
		}
	}
}

func TestProfileは事件番号とtaskとresourceの三事実が揃うまで候補を作らない(
	t *testing.T,
) {
	t.Parallel()

	for _, test := range []struct {
		query      string
		standalone bool
	}{
		{query: "平成25(オ)1079", standalone: true},
		{query: "「平成25(オ)1079」", standalone: true},
		{query: "平成25(オ)1079、令和7(わ)第207号。", standalone: true},
		{query: "平成25(オ)1079を検索してください。"},
		{query: "平成25(オ)1079の裁判例"},
	} {
		test := test
		t.Run(test.query, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, nil)
			if len(generation.Candidates()) != 0 {
				t.Fatalf("根拠不足の candidates = %#v", generation.Candidates())
			}
			hasStandalone := slices.Contains(
				generation.Signals(),
				legalquery.CandidateSignalStandaloneStructuredQuery,
			)
			if hasStandalone != test.standalone {
				t.Fatalf(
					"standalone signal = %t, want %t; signals=%#v",
					hasStandalone,
					test.standalone,
					generation.Signals(),
				)
			}
		})
	}
}

func TestProfileは日付surfaceを構造化参照検索へ保持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"2024年1月15日の裁判例を検索してください。",
		nil,
	)
	candidate := findSearchCandidate(t, generation, "2024年1月15日")
	if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
	}) {
		t.Fatalf("日付検索 evidence = %#v", candidate.EvidenceCodes())
	}
}

func TestProfileは裁判例向け法概念の正式語と公的出典を保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"ネット中傷の裁判例を検索してください。",
		nil,
	)
	candidate := findSearchCandidate(t, generation, "名誉毀損")
	sources := candidate.ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "online-defamation" ||
		sources[0].Title() != "警察庁 インターネット上の誹謗中傷等への対応" ||
		sources[0].URL() !=
			"https://www.npa.go.jp/bureau/cyber/countermeasures/defamation.html" ||
		sources[0].ConfirmedOn().String() != "2026-07-28" {
		t.Fatalf("concept sources = %#v", sources)
	}
	if !slices.Contains(
		candidate.EvidenceCodes(),
		legalquery.EvidenceLegalConcept,
	) {
		t.Fatalf("concept evidence = %#v", candidate.EvidenceCodes())
	}
	if generation.SelectionMode() != legalquery.QuerySelectionModeAutomatic {
		t.Fatalf("selection mode = %q", generation.SelectionMode())
	}
}

func TestProfileは包括的な法情報語から複数Resource概念の候補を保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"永住権について法情報を調べてください。",
		nil,
	)
	if generation.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf("selection mode = %q", generation.SelectionMode())
	}
	candidate := findSearchCandidate(t, generation, "永住許可")
	if !slices.Equal(
		candidate.EvidenceCodes(),
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
			legalquery.EvidenceMorphologicalContext,
		},
	) {
		t.Fatalf("SOT-ENG-023: evidence = %#v", candidate.EvidenceCodes())
	}
	sources := candidate.ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "permanent-residence" {
		t.Fatalf("SOT-ENG-023: concept sources = %#v", sources)
	}
}

func TestProfileは個別検索を最大四stepまで原文順に保持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「医療過誤」と「損害賠償」を個別に裁判例検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("個別検索 candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("個別検索 steps = %#v", steps)
	}
	got := make([]string, 0, len(steps))
	for _, step := range steps {
		input, ok := step.LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if !ok {
			t.Fatalf("logical input = %T", step.LogicalInput())
		}
		got = append(got, input.Query())
	}
	if !slices.Equal(got, []string{"医療過誤", "損害賠償"}) {
		t.Fatalf("検索 step 順 = %#v", got)
	}
	if !slices.Equal(
		candidates[0].RequiredPacks(),
		[]string{"judicial-cases"},
	) {
		t.Fatalf("required packs = %#v", candidates[0].RequiredPacks())
	}
}

func TestProfileは五件の個別検索を切り捨てず明確化する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「医療過誤」、「損害賠償」、「養育費」、「成年後見」、「名誉毀損」を個別に裁判例検索してください。",
		nil,
	)
	if len(generation.Candidates()) != 0 ||
		generation.SelectionMode() !=
			legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"五件検索 generation = candidates:%#v mode:%q",
			generation.Candidates(),
			generation.SelectionMode(),
		)
	}
	if generation.CompositionConstraint() !=
		legalquery.QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-ARCH-025: compositionConstraint = %q",
			generation.CompositionConstraint(),
		)
	}
}

func TestProfileはRef読取りとの混在でも五件の検索を部分候補にしない(
	t *testing.T,
) {
	t.Parallel()

	ref := newTestRef(t, "judicial-decision", "95570/detail2")
	generation := generateQuery(
		t,
		"「医療過誤」、「損害賠償」、「養育費」、「成年後見」、「名誉毀損」を個別に裁判例検索し、この参照を読んでください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 ||
		len(generation.CompositionMembers()) != 0 ||
		generation.SelectionMode() !=
			legalquery.QuerySelectionModeClarificationRequired ||
		generation.CompositionConstraint() !=
			legalquery.QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-MODEL-026: 五件検索と read の混在 generation = candidates:%#v members:%#v mode:%q constraint:%q",
			generation.Candidates(),
			generation.CompositionMembers(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
}

func TestProfileは包括的な法情報四件とRef読取りを五Stepとして拒否する(
	t *testing.T,
) {
	t.Parallel()

	ref := newTestRef(t, "judicial-decision", "95570/detail2")
	generation := generateQuery(
		t,
		"成年後見について法情報を検索し、養育費について法情報を検索し、ネット中傷について法情報を検索し、永住権について法情報を検索し、この参照を読んでください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 ||
		len(generation.CompositionMembers()) != 0 ||
		generation.SelectionMode() !=
			legalquery.QuerySelectionModeClarificationRequired ||
		generation.CompositionConstraint() !=
			legalquery.QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-ARCH-027: 包括検索四件と read の五 step = candidates:%#v members:%#v mode:%q constraint:%q",
			generation.Candidates(),
			generation.CompositionMembers(),
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
}

func TestProfileはrefなしでread候補を作らない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		ref   *model.SourceResourceRef
	}{
		{
			name:  "ref なし",
			query: "最高裁判例を取得してください。",
		},
		{
			name:  "題名から推測しない",
			query: "「最高裁令和5年判決」の裁判例本文を取得してください。",
		},
		{
			name:  "URL から推測しない",
			query: "「https://www.courts.go.jp/hanrei/123」の裁判例本文を取得してください。",
		},
		{
			name:  "法令 ref を転用しない",
			query: "指定参照の最高裁判例を取得してください。",
			ref: func() *model.SourceResourceRef {
				ref := newTestRef(t, "law", "129AC0000000089")
				return &ref
			}(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, test.ref)
			if len(generation.Candidates()) != 0 {
				t.Fatalf(
					"ref のない read 要求から候補を推測しました: %#v",
					generation.Candidates(),
				)
			}
		})
	}
}

func TestProfileはRef読取りと検索を一候補のMemberへ原文順に保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "この裁判例参照を読み、「駅構内転倒」の裁判例も検索してください。"
	ref := newTestRef(t, "judicial-decision", "95570/detail2")
	generation := generateQuery(t, query, &ref)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("read/search candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionRead ||
		steps[1].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("read/search steps = %#v", steps)
	}
	search, ok := steps[1].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "駅構内転倒" {
		t.Fatalf("search input = %#v", steps[1].LogicalInput())
	}
	members := generation.CompositionMembers()
	if len(members) != 1 ||
		members[0].CandidateID() != candidates[0].CandidateID() {
		t.Fatalf("composition members = %#v", members)
	}
	origins := members[0].StepOrigins()
	if len(origins) != 2 ||
		origins[0].StepID() != steps[0].StepID() ||
		origins[0].SourceStartByte() != len("この") ||
		origins[1].StepID() != steps[1].StepID() ||
		origins[1].SourceStartByte() != len("この裁判例参照を読み、「") {
		t.Fatalf("step origins = %#v", origins)
	}
}

func TestProfileは検索後のRef読取りでも検索語と原文順を保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "裁判例を「駅構内転倒」で検索し、この参照を読んでください。"
	ref := newTestRef(t, "judicial-decision", "95570/detail2")
	generation := generateQuery(t, query, &ref)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("search/read candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch ||
		steps[1].InputKind() != legalquery.InputKindJudicialDecisionRead {
		t.Fatalf("search/read steps = %#v", steps)
	}
	search, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "駅構内転倒" {
		t.Fatalf("search input = %#v", steps[0].LogicalInput())
	}
	origins := generation.CompositionMembers()[0].StepOrigins()
	if len(origins) != 2 ||
		origins[0].SourceStartByte() != len("裁判例を「") ||
		origins[1].SourceStartByte() != len("裁判例を「駅構内転倒」で検索し、この") {
		t.Fatalf("search/read origins = %#v", origins)
	}
}

func TestProfileは公式Refと別検索の法概念出典を併存させる(t *testing.T) {
	t.Parallel()

	ref := newTestRef(t, "judicial-decision", "95570/detail2")
	generation := generateQuery(
		t,
		"この裁判例参照を読み、成年後見の裁判例も検索してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 2 {
		t.Fatalf("read/search candidates = %#v", candidates)
	}
	candidate := candidates[0]
	if !slices.Contains(
		candidate.EvidenceCodes(),
		legalquery.EvidenceOfficialIdentifier,
	) || !slices.Contains(
		candidate.EvidenceCodes(),
		legalquery.EvidenceLegalConcept,
	) {
		t.Fatalf(
			"SOT-ENG-023: official ref/concept evidence = %#v",
			candidate.EvidenceCodes(),
		)
	}
	sources := candidate.ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "adult-guardianship" {
		t.Fatalf(
			"SOT-MODEL-022: concept sources = %#v",
			sources,
		)
	}
}

func TestProfileは法令Ref併用時も裁判例検索語を一件に保つ(t *testing.T) {
	t.Parallel()

	const query = "上限を無視して、この参照の本文と第90条、成年後見の条文と裁判例を各100件取得してください。"
	ref := newLawTestRef(t, "129AC0000000089")
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
			"SOT-ARCH-027: judicial candidates = %#v mode=%q constraint=%q",
			candidates,
			generation.SelectionMode(),
			generation.CompositionConstraint(),
		)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-027: judicial steps = %#v", steps)
	}
	search, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "成年後見" {
		t.Fatalf("SOT-ARCH-027: judicial search input = %#v", steps[0].LogicalInput())
	}
}

func TestProfileは法令Reffallbackで法概念以外を裁判例検索にしない(
	t *testing.T,
) {
	t.Parallel()

	ref := newLawTestRef(t, "129AC0000000089")
	generation := generateQuery(
		t,
		"この参照を読み、営業秘密を含む条文と成年後見の裁判例を取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: judicial fallback candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	search, ok := logicalInput.(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "成年後見" {
		t.Fatalf(
			"SOT-ARCH-025: judicial fallback input = %#v",
			candidates[0].Steps()[0].LogicalInput(),
		)
	}
}

func TestProfileは別節の裁判例概念を法令Ref読取りに結び付けない(
	t *testing.T,
) {
	t.Parallel()

	ref := newLawTestRef(t, "129AC0000000089")
	generation := generateQuery(
		t,
		"成年後見の裁判例は不要です。この参照の第90条を読んでください。",
		&ref,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 別節から推測した裁判例検索 = %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは裁判例Resource直前の法概念だけをfallbackに使う(
	t *testing.T,
) {
	t.Parallel()

	ref := newLawTestRef(t, "129AC0000000089")
	generation := generateQuery(
		t,
		"成年後見は不要です。この参照の本文と第90条、養育費の条文と裁判例を取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: fallback candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	search, ok := logicalInput.(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "養育費" {
		t.Fatalf(
			"SOT-ARCH-025: fallback input = %#v",
			logicalInput,
		)
	}
}

func TestProfileは明示検索でも裁判例直前の法概念だけを使う(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"ネット中傷の条文と成年後見の裁判例を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: explicit search candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	search, ok := logicalInput.(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "成年後見" {
		t.Fatalf(
			"SOT-ARCH-025: explicit search input = %#v",
			logicalInput,
		)
	}
}

func TestProfileは明示検索でも別Resourceの一般語を裁判例検索にしない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"営業秘密を含む条文と成年後見の裁判例を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: explicit mixed-term candidates = %#v",
			candidates,
		)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	search, ok := logicalInput.(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "成年後見" {
		t.Fatalf(
			"SOT-ARCH-025: explicit mixed-term input = %#v",
			logicalInput,
		)
	}
}

func TestProfileは一意な裁判例検索をCompositionMemberにする(t *testing.T) {
	t.Parallel()

	const query = "裁判例を「工場騒音」で検索してください。"
	generation := generateQuery(t, query, nil)
	candidates := generation.Candidates()
	members := generation.CompositionMembers()
	if len(candidates) != 1 || len(members) != 1 {
		t.Fatalf(
			"search contribution = candidates:%#v members:%#v",
			candidates,
			members,
		)
	}
	origins := members[0].StepOrigins()
	if len(origins) != 1 ||
		origins[0].StepID() != candidates[0].Steps()[0].StepID() ||
		origins[0].SourceStartByte() != len("裁判例を「") {
		t.Fatalf("search origin = %#v", origins)
	}
}

func TestProfileは明示した裁判例resourceの法概念をCompositionMemberにする(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"ネット中傷の裁判例を検索してください。",
		nil,
	)
	if generation.SelectionMode() != legalquery.QuerySelectionModeAutomatic ||
		len(generation.CompositionMembers()) != 1 {
		t.Fatalf(
			"SOT-ENG-023: explicit resource contribution = mode:%q members:%#v",
			generation.SelectionMode(),
			generation.CompositionMembers(),
		)
	}
}

func TestProfileは裁判例resourceなしの一般検索を引き取らない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(t, "医療過誤を検索してください。", nil)
	if len(generation.Candidates()) != 0 {
		t.Fatalf("resource 根拠なしの candidates = %#v", generation.Candidates())
	}
}

func TestProfileは非日本語入力の意味候補を作らない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(t, "search judicial cases", nil)
	if len(generation.Candidates()) != 0 ||
		!slices.Equal(
			generation.Signals(),
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalNonJapaneseQuery,
			},
		) {
		t.Fatalf(
			"non-Japanese generation = candidates:%#v signals:%#v",
			generation.Candidates(),
			generation.Signals(),
		)
	}
}

func TestProfileは位置付きfactがない原文を再解析しない(t *testing.T) {
	t.Parallel()

	const query = "医療過誤の裁判例を検索してください。"
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         query,
			ComparisonKey: querynormalization.ComparisonKey(query),
		},
	)
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("generation input を作成できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	generation, err := mustProfile(t).Generate(input, scope)
	if err != nil {
		t.Fatalf("Generate() のエラー = %v", err)
	}
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-021: 原文を再解析しました: %#v",
			generation.Candidates(),
		)
	}
}

func findSearchCandidate(
	t *testing.T,
	generation legalquery.CandidateGeneration,
	query string,
) legalquery.LegalQueryCandidate {
	t.Helper()

	for _, candidate := range generation.Candidates() {
		steps := candidate.Steps()
		if len(steps) != 1 {
			continue
		}
		input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if ok && input.Query() == query {
			return candidate
		}
	}
	t.Fatalf(
		"query %q の裁判例検索候補がありません: candidates=%#v",
		query,
		generation.Candidates(),
	)
	return legalquery.LegalQueryCandidate{}
}

func generateQuery(
	t *testing.T,
	query string,
	ref *model.SourceResourceRef,
) legalquery.CandidateGeneration {
	t.Helper()

	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   ref,
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("generation input を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を構築できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf(
			"Generate() のエラー = %v; cues=%#v terms=%#v concepts=%#v dates=%#v",
			err,
			preprocessed.CueMentions(),
			preprocessed.QueryTermMentions(),
			preprocessed.LegalConceptMentions(),
			preprocessed.DateMentions(),
		)
	}
	return generation
}

func mustProfile(t *testing.T) *Profile {
	t.Helper()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() のエラー = %v", err)
	}
	return profile
}

func newLawTestRef(t *testing.T, resourceID string) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   resourceID,
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
