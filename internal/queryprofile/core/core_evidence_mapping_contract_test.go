package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const (
	coreEvidenceMappingInputKindsID = "core-evidence-mapping-input-kinds"
	coreEvidenceMappingPositiveID   = "core-evidence-mapping-topic-positive"
	coreEvidenceMappingRefSpanID    = "core-evidence-mapping-ref-no-span"
)

func TestCoreEvidenceMappingInputKinds(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		kind             legalquery.LogicalInputKind
		required         []evidencePair
		resourceBindings int
	}{
		{
			name:  "law_search",
			query: "独禁法の正式な法令を検索してください。",
			kind:  legalquery.InputKindLawSearch,
			required: []evidencePair{
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias},
			},
		},
		{
			name:  "law_content_search",
			query: "法令本文から「営業秘密」を含む条文を検索してください。",
			kind:  legalquery.InputKindLawContentSearch,
			required: []evidencePair{
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceGeneralTerm},
			},
			resourceBindings: 2,
		},
		{
			name:  "law_read",
			query: "法令ID 345AC0000000048 の本文を取得してください。",
			kind:  legalquery.InputKindLawRead,
			required: []evidencePair{
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialIdentifier},
			},
		},
		{
			name:  "law_article_read",
			query: "商法第512条第1項の条文を読んでください。",
			kind:  legalquery.InputKindLawArticleRead,
			required: []evidencePair{
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
			},
		},
		{
			name:  "law_updates",
			query: "2026年5月15日の更新一覧を列挙する",
			kind:  legalquery.InputKindLawUpdates,
			required: []evidencePair{
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
				{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
				{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fixture := mustCoreEvidenceFixture(t, test.query, nil)
			draft := mustCoreSingleKindDraft(
				t,
				fixture.drafts,
				test.kind,
			)

			evaluation := mustCoreEvidenceEvaluation(
				t,
				fixture.input,
				fixture.cues,
				[]candidateDraft{draft},
			)
			evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
			assertCoreAllowedEvidence(t, test.kind, evidence)
			assertCoreResourceCueValues(
				t,
				fixture.input,
				fixture.cues,
				evidence,
				coreResourceMeanings(test.kind),
				true,
			)
			for _, pair := range test.required {
				assertEvidencePair(t, evidence, pair)
			}
			if test.resourceBindings > 0 {
				var count int
				for _, value := range evidence {
					if value.Layer() ==
						profileevidence.LayerExplicitTaskResource &&
						value.Code() ==
							legalquery.EvidenceExplicitResource {
						count++
					}
				}
				if count != test.resourceBindings {
					t.Fatalf(
						"%s: %s の compatible resource binding 数 = %d, want %d",
						coreEvidenceMappingInputKindsID,
						test.kind,
						count,
						test.resourceBindings,
					)
				}
			}
			assertCoreForbiddenFactRejected(t, test.kind)
			assertCoreDraftClusterEligible(t, evaluation, 0)
		})
	}

	t.Run("単一stepの同じ表記を別節から借りない", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"民法の法令を検索してください。民法は参考です。",
			nil,
		)
		if len(fixture.input.LawNameMentions()) < 2 {
			t.Fatalf(
				"%s: 別節の同じ法令名fixtureがありません",
				coreEvidenceMappingInputKindsID,
			)
		}
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{
				mustCoreSingleKindDraft(
					t,
					fixture.drafts,
					legalquery.InputKindLawSearch,
				),
			},
		)
		evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
		assertEvidencePair(
			t,
			evidence,
			evidencePair{
				profileevidence.LayerExplicitTaskResource,
				legalquery.EvidenceExplicitTask,
			},
		)
		assertEvidencePair(
			t,
			evidence,
			evidencePair{
				profileevidence.LayerExplicitTaskResource,
				legalquery.EvidenceExplicitResource,
			},
		)
		if countCoreEvidenceByFactPrefix(evidence, "law-name-") != 1 {
			t.Fatalf(
				"%s: 別節の同じ法令名を単一stepへ束縛しました",
				coreEvidenceMappingInputKindsID,
			)
		}
	})

	t.Run("asOfとupdatesの日付を区別する", func(t *testing.T) {
		asOfTests := []struct {
			name  string
			query string
			kind  legalquery.LogicalInputKind
		}{
			{
				name:  "law_search",
				query: "2024年4月1日時点の民法を検索してください。",
				kind:  legalquery.InputKindLawSearch,
			},
			{
				name:  "law_content_search",
				query: "2024年4月1日時点の法令本文で「行政指導」を検索してください。",
				kind:  legalquery.InputKindLawContentSearch,
			},
			{
				name:  "law_read",
				query: "2024年4月1日時点の民法を読んでください。",
				kind:  legalquery.InputKindLawRead,
			},
			{
				name:  "law_article_read",
				query: "2024年4月1日時点の商法第512条第1項を読んでください。",
				kind:  legalquery.InputKindLawArticleRead,
			},
		}
		for _, test := range asOfTests {
			test := test
			t.Run(test.name, func(t *testing.T) {
				fixture := mustCoreEvidenceFixture(t, test.query, nil)
				evaluation := mustCoreEvidenceEvaluation(
					t,
					fixture.input,
					fixture.cues,
					[]candidateDraft{
						mustCoreSingleKindDraft(
							t,
							fixture.drafts,
							test.kind,
						),
					},
				)
				date := mustCoreEvidenceByFactPrefix(
					t,
					mustCoreStepEvidence(t, evaluation, 0, 0),
					"date-",
				)
				assertCoreDateEvidence(
					t,
					date,
					false,
					false,
				)
			})
		}

		fixture := mustCoreEvidenceFixture(
			t,
			"2024年4月1日の更新一覧を列挙する",
			nil,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{
				mustCoreSingleKindDraft(
					t,
					fixture.drafts,
					legalquery.InputKindLawUpdates,
				),
			},
		)
		evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
		date := mustCoreEvidenceByFactPrefix(t, evidence, "date-")
		assertCoreDateEvidence(t, date, true, true)
		task := mustCoreEvidenceByCode(
			t,
			evidence,
			legalquery.EvidenceExplicitTask,
		)
		taskSpan, taskHasSpan := task.Span()
		key := assertCoreDraftClusterEligible(t, evaluation, 0)
		members := key.Members()
		if !taskHasSpan || len(members) != 1 ||
			members[0].EvidenceSpan() != taskSpan {
			t.Fatalf(
				"%s: updates の evidenceSpan が task cue ではありません",
				coreEvidenceMappingInputKindsID,
			)
		}
	})

	t.Run("同じ日付をasOfとupdatesへ重複束縛しない", func(t *testing.T) {
		same := mustCoreEvidenceFixture(
			t,
			"2024年4月1日時点の民法を検索し、2024年4月1日の更新一覧も列挙する",
			nil,
		)
		sameDraft := mustCoreDraftWithKinds(
			t,
			same.drafts,
			legalquery.InputKindLawSearch,
			legalquery.InputKindLawUpdates,
		)
		assertCoreDraftNotEligible(
			t,
			same.input,
			same.cues,
			sameDraft,
			coreEvidenceMappingInputKindsID,
		)

		different := mustCoreEvidenceFixture(
			t,
			"2024年4月1日時点の民法を検索し、2024年4月2日の更新一覧も列挙する",
			nil,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			different.input,
			different.cues,
			[]candidateDraft{
				mustCoreDraftWithKinds(
					t,
					different.drafts,
					legalquery.InputKindLawSearch,
					legalquery.InputKindLawUpdates,
				),
			},
		)
		members := assertCoreDraftClusterEligible(t, evaluation, 0).Members()
		if len(members) != 2 ||
			members[0].TopicOrdinal() != 1 ||
			members[1].TopicOrdinal() != 1 {
			t.Fatalf(
				"%s: 非分離multi-stepのtopicOrdinal = %#v",
				coreEvidenceMappingInputKindsID,
				members,
			)
		}
	})

	t.Run("複数stepで明示cueを共有しない", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"個人情報保護法の法令を検索し、「個人データ」を含む条文も検索してください。",
			nil,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{
				mustCoreDraftWithKinds(
					t,
					fixture.drafts,
					legalquery.InputKindLawSearch,
					legalquery.InputKindLawContentSearch,
				),
			},
		)
		lawSearch := mustCoreStepEvidence(t, evaluation, 0, 0)
		contentSearch := mustCoreStepEvidence(t, evaluation, 0, 1)
		assertCoreResourceCueValues(
			t,
			fixture.input,
			fixture.cues,
			lawSearch,
			coreResourceMeanings(legalquery.InputKindLawSearch),
			true,
		)
		assertCoreResourceCueValues(
			t,
			fixture.input,
			fixture.cues,
			contentSearch,
			coreResourceMeanings(legalquery.InputKindLawContentSearch),
			true,
		)
		assertNoSharedExplicitFact(t, lawSearch, contentSearch)
		members := assertCoreDraftClusterEligible(t, evaluation, 0).Members()
		if len(members) != 2 ||
			members[0].TopicOrdinal() != 1 ||
			members[1].TopicOrdinal() != 1 {
			t.Fatalf(
				"%s: 非分離の異種stepが主題分離されました: %#v",
				coreEvidenceMappingInputKindsID,
				members,
			)
		}
	})

	t.Run("異なるresource意味を代用しない", func(t *testing.T) {
		tests := []struct {
			name  string
			query string
			kind  legalquery.LogicalInputKind
		}{
			{
				name:  "law_article_readへlawを代用しない",
				query: "商法第512条第1項の法令を読んでください。",
				kind:  legalquery.InputKindLawArticleRead,
			},
			{
				name:  "law_readへlaw_provisionを代用しない",
				query: "法令ID 345AC0000000048 の法令本文を読んでください。",
				kind:  legalquery.InputKindLawRead,
			},
		}
		for _, test := range tests {
			test := test
			t.Run(test.name, func(t *testing.T) {
				fixture := mustCoreEvidenceFixture(t, test.query, nil)
				draft := candidateDraft{}
				if test.kind == legalquery.InputKindLawRead {
					draft = mustCoreLawIDReadDraft(
						t,
						"345AC0000000048",
					)
				} else {
					draft = mustCoreSingleKindDraft(
						t,
						fixture.drafts,
						test.kind,
					)
				}
				evaluation := mustCoreEvidenceEvaluation(
					t,
					fixture.input,
					fixture.cues,
					[]candidateDraft{draft},
				)
				assertCoreResourceCueValues(
					t,
					fixture.input,
					fixture.cues,
					mustCoreStepEvidence(t, evaluation, 0, 0),
					coreResourceMeanings(test.kind),
					false,
				)
			})
		}

		t.Run("law_updatesへlawを代用しない", func(t *testing.T) {
			fixture := mustCoreEvidenceFixture(
				t,
				"2024年4月1日の法令を列挙する",
				nil,
			)
			assertCoreDraftNotEligible(
				t,
				fixture.input,
				fixture.cues,
				mustCoreSingleKindDraft(
					t,
					fixture.drafts,
					legalquery.InputKindLawUpdates,
				),
				coreEvidenceMappingInputKindsID,
			)
		})
	})
}

func TestCoreEvidenceMappingRequiresPositivePerTopic(t *testing.T) {
	fixture := mustCoreEvidenceFixture(
		t,
		"民法と商法をそれぞれ検索してください。",
		nil,
	)
	draft := mustCoreDraftWithKinds(
		t,
		fixture.drafts,
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawSearch,
	)
	evaluation := mustCoreEvidenceEvaluation(
		t,
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
	)
	key := assertCoreDraftClusterEligible(t, evaluation, 0)
	members := key.Members()
	if len(members) != 2 ||
		members[0].TopicOrdinal() != 1 ||
		members[1].TopicOrdinal() != 2 {
		t.Fatalf(
			"%s: 二主題の cluster member = %#v",
			coreEvidenceMappingPositiveID,
			members,
		)
	}

	t.Run("同じ主題の異種stepは同じordinalにする", func(t *testing.T) {
		content, err := legalquery.NewLawContentSearchIntentV1(
			legalquery.LawContentSearchIntentV1Values{
				AllTerms: []string{"民法"},
			},
		)
		if err != nil {
			t.Fatalf(
				"%s: content inputを構築できません: %v",
				coreEvidenceMappingPositiveID,
				err,
			)
		}
		grouped := cloneCoreEvidenceTestDraft(draft)
		sameTopic := grouped.steps[0]
		sameTopic.input = content
		grouped.steps = []stepDraft{
			grouped.steps[0],
			sameTopic,
			grouped.steps[1],
		}
		for index := range grouped.steps {
			grouped.steps[index].topicOrdinal = 0
		}
		values, err := withCoreEvidenceBindings(
			fixture.input,
			fixture.cues,
			[]candidateDraft{grouped},
		)
		if err != nil {
			t.Fatalf(
				"%s: ordinal bindingを構築できません: %v",
				coreEvidenceMappingPositiveID,
				err,
			)
		}
		steps := values[0].steps
		if len(steps) != 3 ||
			steps[0].topicOrdinal != 1 ||
			steps[1].topicOrdinal != 1 ||
			steps[2].topicOrdinal != 2 {
			t.Fatalf(
				"%s: 同一主題のordinal = %#v",
				coreEvidenceMappingPositiveID,
				steps,
			)
		}
	})

	withoutPositive := cloneCoreEvidenceTestDraft(draft)
	for index := range withoutPositive.steps[1].evidenceBindings {
		withoutPositive.steps[1].evidenceBindings[index].IndependentPositive = false
	}
	assertCoreDraftNotEligible(
		t,
		fixture.input,
		fixture.cues,
		withoutPositive,
		coreEvidenceMappingPositiveID,
	)
}

func TestCoreEvidenceMappingRefHasNoSpan(t *testing.T) {
	ref := mustContractLawRef(t, "provider-one", "source-one")
	t.Run("resource cueなしの一意なreadを使う", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"これを読んでください。",
			&ref,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{mustCoreRefReadDraft(t, ref)},
		)
		evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
		assertCoreRefEvidence(t, evidence)
		assertCoreResourceCueValues(
			t,
			fixture.input,
			fixture.cues,
			evidence,
			nil,
			false,
		)
		task := mustCoreEvidenceByCode(
			t,
			evidence,
			legalquery.EvidenceExplicitTask,
		)
		taskSpan, exists := task.Span()
		members := assertCoreDraftClusterEligible(t, evaluation, 0).Members()
		if !exists || len(members) != 1 ||
			members[0].EvidenceSpan() != taskSpan {
			t.Fatalf(
				"%s: ref read のcluster spanがread cueではありません",
				coreEvidenceMappingRefSpanID,
			)
		}
	})

	t.Run("互換resource cueを受理する", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"この法令を読んでください。",
			&ref,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{mustCoreRefReadDraft(t, ref)},
		)
		evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
		assertCoreRefEvidence(t, evidence)
		assertCoreResourceCueValues(
			t,
			fixture.input,
			fixture.cues,
			evidence,
			[]string{"law"},
			true,
		)
	})

	t.Run("競合resource cueを拒否する", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"この条文を読んでください。",
			&ref,
		)
		assertCoreDraftNotEligible(
			t,
			fixture.input,
			fixture.cues,
			mustCoreRefReadDraft(t, ref),
			coreEvidenceMappingRefSpanID,
		)
	})

	t.Run("read relationが零件または複数なら拒否する", func(t *testing.T) {
		template := mustCoreRefReadDraft(t, ref)
		tests := []struct {
			name  string
			query string
		}{
			{
				name:  "零件",
				query: "この参照を確認してください。",
			},
			{
				name:  "二件",
				query: "これを読んでください。もう一度読んでください。",
			},
		}
		for _, test := range tests {
			test := test
			t.Run(test.name, func(t *testing.T) {
				fixture := mustCoreEvidenceFixture(
					t,
					test.query,
					&ref,
				)
				draft := cloneCoreEvidenceTestDraft(template)
				for index := range draft.steps {
					draft.steps[index].startByte = 0
					draft.steps[index].evidenceBindings = nil
				}
				assertCoreDraftNotEligible(
					t,
					fixture.input,
					fixture.cues,
					draft,
					coreEvidenceMappingRefSpanID,
				)
			})
		}
	})

	t.Run("refとarticleをreadへ束縛する", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"この参照の第3条を読んでください。",
			&ref,
		)
		evaluation := mustCoreEvidenceEvaluation(
			t,
			fixture.input,
			fixture.cues,
			[]candidateDraft{
				mustCoreRefArticleReadDraft(t, ref, "3"),
			},
		)
		evidence := mustCoreStepEvidence(t, evaluation, 0, 0)
		assertCoreRefEvidence(t, evidence)
		assertEvidencePair(
			t,
			evidence,
			evidencePair{
				profileevidence.LayerTargetAnchor,
				legalquery.EvidenceStructuredReference,
			},
		)
		assertCoreResourceCueValues(
			t,
			fixture.input,
			fixture.cues,
			evidence,
			nil,
			false,
		)
	})
}
