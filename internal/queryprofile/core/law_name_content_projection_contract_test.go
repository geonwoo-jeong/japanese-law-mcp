package core

import (
	"sort"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const coreLawNameProjectionContractID = "core-law-name-content-projection"

func TestCoreLawNameContentProjection(t *testing.T) {
	fixture := newCoreProjectionFixture(t)

	t.Run("引用句と個別主題を経路順に投影する", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			surface    string
			individual bool
			wantCode   legalquery.EvidenceCode
		}{
			{
				name:     "同じspanの引用句",
				query:    "法令本文から「民法」を検索してください。",
				surface:  "民法",
				wantCode: legalquery.EvidenceGeneralTerm,
			},
			{
				name:       "明示した個別主題",
				query:      "民法と商法を個別に法令本文で検索してください。",
				surface:    "民法",
				individual: true,
				wantCode:   legalquery.EvidenceOfficialAlias,
			},
			{
				name:       "引用句を個別主題より優先する",
				query:      "「民法」と商法を個別に法令本文で検索してください。",
				surface:    "民法",
				individual: true,
				wantCode:   legalquery.EvidenceGeneralTerm,
			},
			{
				name:       "混合経路の個別主題",
				query:      "「民法」と商法を個別に法令本文で検索してください。",
				surface:    "商法",
				individual: true,
				wantCode:   legalquery.EvidenceOfficialAlias,
			},
			{
				name:     "引用した誤記を原文のまま使う",
				query:    "法令本文から「労契去」を検索してください。",
				surface:  "労契去",
				wantCode: legalquery.EvidenceGeneralTerm,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				input, cues := fixture.inputAndCues(t, test.query)
				span := lawSpanBySurface(t, input, test.surface)
				var individual *legalquery.QuerySpan
				if test.individual {
					current := span
					individual = &current
				}
				context := mustCoreProjectionContext(
					t,
					span,
					individual,
					nil,
				)
				option := mustCoreProjectedOption(
					t,
					fixture.profile,
					input,
					cues,
					context,
				)
				assertCoreProjectionOption(
					t,
					option,
					test.surface,
					test.wantCode,
					1,
				)
			})
		}
	})

	t.Run("検証済み共有末尾主題だけを投影する", func(t *testing.T) {
		input := fixture.input(t, "民法と商法")
		for _, surface := range []string{"民法", "商法"} {
			span := lawSpanBySurface(t, input, surface)
			validated := mustCoreValidatedTopic(
				t,
				span,
				coreProjectionTaskResourceEvidence(true),
			)
			projectionContext := mustCoreProjectionContext(
				t,
				span,
				nil,
				&validated,
			)
			option := mustCoreProjectedOption(
				t,
				fixture.profile,
				input,
				resolvedCues{},
				projectionContext,
			)
			assertCoreProjectionOption(
				t,
				option,
				surface,
				legalquery.EvidenceOfficialAlias,
				1,
			)
		}

		quotedInput := fixture.input(t, "「民法」と商法")
		quotedSpan := lawSpanBySurface(t, quotedInput, "民法")
		validated := mustCoreValidatedTopic(
			t,
			quotedSpan,
			coreProjectionTaskResourceEvidence(true),
		)
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			quotedInput,
			resolvedCues{},
			mustCoreProjectionContext(t, quotedSpan, nil, &validated),
		)
	})

	t.Run("個別主題と共有末尾主題の完全一致を要求する", func(t *testing.T) {
		input, cues := fixture.inputAndCues(
			t,
			"民法第103条と商法を個別に法令本文で検索してください。",
		)
		lawSpan := lawSpanBySurface(t, input, "民法")
		articles := input.ArticleMentions()
		if len(articles) != 1 {
			t.Fatalf(
				"%s: article fixture = %#v",
				coreLawNameProjectionContractID,
				articles,
			)
		}
		combined := mustCoreProjectionSpan(
			t,
			lawSpan.StartByte(),
			articles[0].Span().EndByte(),
		)
		individualContext := mustCoreProjectionContext(
			t,
			lawSpan,
			&combined,
			nil,
		)
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			input,
			cues,
			individualContext,
		)

		mismatch := mustCoreProjectionSpan(
			t,
			lawSpan.StartByte(),
			lawSpan.StartByte()+len("民"),
		)
		validated := mustCoreValidatedTopic(
			t,
			mismatch,
			coreProjectionTaskResourceEvidence(true),
		)
		sharedContext := mustCoreProjectionContext(
			t,
			lawSpan,
			nil,
			&validated,
		)
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			input,
			resolvedCues{},
			sharedContext,
		)
	})

	t.Run("readとlaw searchとの競合を閉じて拒否する", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			individual bool
		}{
			{
				name:       "個別主題とread",
				query:      "民法を個別に法令本文で検索して読んでください。",
				individual: true,
			},
			{
				name:       "個別主題とlaw search",
				query:      "民法を個別に法令名と法令本文で検索してください。",
				individual: true,
			},
			{
				name:  "引用句とread",
				query: "法令本文から「民法」を検索して読んでください。",
			},
			{
				name:  "引用句とlaw search",
				query: "法令名と法令本文から「民法」を検索してください。",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				input, cues := fixture.inputAndCues(t, test.query)
				span := lawSpanBySurface(t, input, "民法")
				var individual *legalquery.QuerySpan
				if test.individual {
					current := span
					individual = &current
				}
				projectionContext := mustCoreProjectionContext(
					t,
					span,
					individual,
					nil,
				)
				assertCoreProjectionRejected(
					t,
					fixture.profile,
					input,
					cues,
					projectionContext,
				)
			})
		}

		input, cues := fixture.inputAndCues(
			t,
			"民法第103条を個別に法令本文で検索して読んでください。",
		)
		if len(input.ArticleMentions()) != 1 {
			t.Fatalf(
				"%s: article read fixture = %#v",
				coreLawNameProjectionContractID,
				input.ArticleMentions(),
			)
		}
		span := lawSpanBySurface(t, input, "民法")
		individual := span
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			input,
			cues,
			mustCoreProjectionContext(t, span, &individual, nil),
		)
	})

	t.Run("別節のtaskとresourceを借りない", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			individual bool
		}{
			{
				name:  "引用句",
				query: "「民法」を検索してください。法令本文から「商法」を検索してください。",
			},
			{
				name:       "個別主題",
				query:      "民法を個別に検索してください。商法を個別に法令本文で検索してください。",
				individual: true,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				input, cues := fixture.inputAndCues(t, test.query)
				first := lawSpanBySurface(t, input, "民法")
				var firstTopic *legalquery.QuerySpan
				if test.individual {
					current := first
					firstTopic = &current
				}
				assertCoreProjectionRejected(
					t,
					fixture.profile,
					input,
					cues,
					mustCoreProjectionContext(
						t,
						first,
						firstTopic,
						nil,
					),
				)

				second := lawSpanBySurface(t, input, "商法")
				var secondTopic *legalquery.QuerySpan
				if test.individual {
					current := second
					secondTopic = &current
				}
				option := mustCoreProjectedOption(
					t,
					fixture.profile,
					input,
					cues,
					mustCoreProjectionContext(
						t,
						second,
						secondTopic,
						nil,
					),
				)
				wantCode := legalquery.EvidenceGeneralTerm
				if test.individual {
					wantCode = legalquery.EvidenceOfficialAlias
				}
				assertCoreProjectionOption(t, option, "商法", wantCode, 1)
			})
		}
	})

	t.Run("裸の法令名と識別子と誤記補正だけを拒否する", func(t *testing.T) {
		input, cues := fixture.inputAndCues(
			t,
			"民法を法令本文で検索してください。",
		)
		span := lawSpanBySurface(t, input, "民法")
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			input,
			cues,
			mustCoreProjectionContext(t, span, nil, nil),
		)

		identifierInput := fixture.input(
			t,
			"法令ID 129AC0000000089 を個別に法令本文で検索してください。",
		)
		if len(identifierInput.LawNameMentions()) != 0 {
			t.Fatalf(
				"%s: 識別子を法令名として扱いました: %#v",
				coreLawNameProjectionContractID,
				identifierInput.LawNameMentions(),
			)
		}

		typoInput, typoCues := fixture.inputAndCues(
			t,
			"労契去を個別に法令本文で検索してください。",
		)
		typoSpan := lawSpanBySurface(t, typoInput, "労契去")
		individual := typoSpan
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			typoInput,
			typoCues,
			mustCoreProjectionContext(t, typoSpan, &individual, nil),
		)
		validated := mustCoreValidatedTopic(
			t,
			typoSpan,
			coreProjectionTaskResourceEvidence(true),
		)
		assertCoreProjectionRejected(
			t,
			fixture.profile,
			typoInput,
			resolvedCues{},
			mustCoreProjectionContext(t, typoSpan, nil, &validated),
		)
	})

	t.Run("同じ表記の複数identityを一つの検索語と正の根拠にする", func(t *testing.T) {
		preprocessed := fixture.preprocess(t, "民法")
		input := mustCoreProjectionInput(t, preprocessed)
		laws := input.LawNameMentions()
		if len(laws) != 1 {
			t.Fatalf(
				"%s: law name fixture = %#v",
				coreLawNameProjectionContractID,
				laws,
			)
		}
		first := laws[0]
		other, err := legalquery.NewLawNameMention(
			legalquery.LawNameMentionValues{
				Span:       first.Span(),
				Surface:    first.Surface(),
				LawID:      "test-second-law",
				RevisionID: "test-second-revision",
				LawNumber:  "令和八年法律第百号",
				Canonical:  "試験用第二民法",
				MatchKind:  legalquery.PreprocessMatchRegisteredTerm,
			},
		)
		if err != nil {
			t.Fatalf(
				"%s: 複数 identity fixture を作成できません: %v",
				coreLawNameProjectionContractID,
				err,
			)
		}
		laws = append(laws, other)
		sort.Slice(laws, func(left int, right int) bool {
			return laws[left].LawID() < laws[right].LawID()
		})
		input = replaceCoreProjectionLawMentions(t, preprocessed, laws)
		span := lawSpanBySurface(t, input, "民法")
		validated := mustCoreValidatedTopic(
			t,
			span,
			coreProjectionTaskResourceEvidence(true),
		)
		option := mustCoreProjectedOption(
			t,
			fixture.profile,
			input,
			resolvedCues{},
			mustCoreProjectionContext(t, span, nil, &validated),
		)
		assertCoreProjectionOption(
			t,
			option,
			"民法",
			legalquery.EvidenceOfficialAlias,
			2,
		)
	})

	t.Run("Unicode空白だけを原文表記の前後から除く", func(t *testing.T) {
		query := "対象は\u3000民法\u00a0です"
		surface := "\u3000民法\u00a0"
		startByte := len("対象は")
		span := mustCoreProjectionSpan(t, startByte, startByte+len(surface))
		mention, err := legalquery.NewLawNameMention(
			legalquery.LawNameMentionValues{
				Span:       span,
				Surface:    surface,
				LawID:      "test-whitespace-law",
				RevisionID: "test-whitespace-revision",
				LawNumber:  "令和八年法律第百一号",
				Canonical:  "置換してはならない民法典",
				MatchKind:  legalquery.PreprocessMatchRegisteredTerm,
			},
		)
		if err != nil {
			t.Fatalf(
				"%s: 空白付き法令名を作成できません: %v",
				coreLawNameProjectionContractID,
				err,
			)
		}
		preprocessed, err := legalquery.NewPreprocessResult(
			legalquery.PreprocessResultValues{
				Query:           query,
				ComparisonKey:   querynormalization.ComparisonKey(query),
				LawNameMentions: []legalquery.LawNameMention{mention},
			},
		)
		if err != nil {
			t.Fatalf(
				"%s: 空白付き前処理結果を作成できません: %v",
				coreLawNameProjectionContractID,
				err,
			)
		}
		input := mustCoreProjectionInput(t, preprocessed)
		validated := mustCoreValidatedTopic(
			t,
			span,
			coreProjectionTaskResourceEvidence(true),
		)
		option := mustCoreProjectedOption(
			t,
			fixture.profile,
			input,
			resolvedCues{},
			mustCoreProjectionContext(t, span, nil, &validated),
		)
		assertCoreProjectionOption(
			t,
			option,
			"民法",
			legalquery.EvidenceOfficialAlias,
			1,
		)
	})

	t.Run("共有末尾主題の証票を閉じて検証し複製する", func(t *testing.T) {
		span := mustCoreProjectionSpan(t, 0, len("民法"))
		validTask := profileevidence.EvidenceValues{
			FactID: "cue-task",
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitTask,
		}
		validResource := profileevidence.EvidenceValues{
			FactID: "cue-resource",
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitResource,
		}
		tests := []struct {
			name     string
			evidence []profileevidence.EvidenceValues
		}{
			{name: "taskなし", evidence: nil},
			{name: "明示resourceなし", evidence: []profileevidence.EvidenceValues{validTask}},
			{name: "task重複", evidence: []profileevidence.EvidenceValues{validTask, validTask}},
			{name: "resource重複", evidence: []profileevidence.EvidenceValues{validTask, validResource, validResource}},
			{name: "異なるlayer", evidence: []profileevidence.EvidenceValues{{FactID: "cue-task", Layer: profileevidence.LayerTargetAnchor, Code: legalquery.EvidenceExplicitTask}}},
			{name: "許可外code", evidence: []profileevidence.EvidenceValues{{FactID: "cue-task", Layer: profileevidence.LayerExplicitTaskResource, Code: legalquery.EvidenceGeneralTerm}}},
			{name: "factIDなし", evidence: []profileevidence.EvidenceValues{{Layer: profileevidence.LayerExplicitTaskResource, Code: legalquery.EvidenceExplicitTask}}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if _, err := newCoreValidatedContentTopic(
					span,
					test.evidence,
					coreLawProvisionBindingExplicitResource,
				); err == nil {
					t.Fatalf(
						"%s: 無効な証票を受理しました",
						coreLawNameProjectionContractID,
					)
				}
			})
		}

		evidence := []profileevidence.EvidenceValues{validTask, validResource}
		if _, err := newCoreValidatedContentTopic(
			span,
			evidence,
			coreLawProvisionBindingTerminalTask,
		); err == nil {
			t.Fatalf(
				"%s: terminal task 束縛へ明示 resource を混在しました",
				coreLawNameProjectionContractID,
			)
		}
		if _, err := newCoreValidatedContentTopic(span, evidence, 0); err == nil {
			t.Fatalf(
				"%s: 未定義の law provision 束縛を受理しました",
				coreLawNameProjectionContractID,
			)
		}
		if _, err := newCoreValidatedContentTopic(
			span,
			[]profileevidence.EvidenceValues{validTask},
			coreLawProvisionBindingTerminalTask,
		); err != nil {
			t.Fatalf(
				"%s: 検証済みの resource 省略証票を拒否しました: %v",
				coreLawNameProjectionContractID,
				err,
			)
		}
		validated := mustCoreValidatedTopic(t, span, evidence)
		projectionContext := mustCoreProjectionContext(
			t,
			span,
			nil,
			&validated,
		)
		evidence[0].FactID = "changed-input"
		validated.baseEvidence[0].FactID = "changed-proof"
		if projectionContext.sharedTerminalTopic.baseEvidence[0].FactID != "cue-task" {
			t.Fatalf(
				"%s: 証票の根拠を複製していません: %#v",
				coreLawNameProjectionContractID,
				projectionContext.sharedTerminalTopic.baseEvidence,
			)
		}
	})
}
