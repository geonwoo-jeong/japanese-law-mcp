package profileevidence_test

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const clusterKeyVerificationID = "profile-evidence-cluster-key"

func TestProfileEvidenceClusterKey(t *testing.T) {
	t.Run("layer優先と原文step順でkeyを作る", func(t *testing.T) {
		semantic := mustSpan(t, 0, 2)
		targetLater := mustSpan(t, 12, 18)
		explicit := mustSpan(t, 30, 36)
		targetFirst := mustSpan(t, 4, 8)
		targetSecond := mustSpan(t, 8, 11)
		values := profileevidence.MappingValues{
			ProfileID: "core",
			Facts: []profileevidence.FactValues{
				{FactID: "semantic", Span: &semantic},
				{FactID: "target-later", Span: &targetLater},
				{FactID: "explicit", Span: &explicit},
				{FactID: "target-first", Span: &targetFirst},
				{FactID: "target-second", Span: &targetSecond},
			},
			Drafts: []profileevidence.DraftValues{
				{
					DraftID: "draft-one",
					Steps: []profileevidence.StepValues{
						{
							StepID:        "step-two",
							SourceOrdinal: 2,
							TopicOrdinal:  2,
							Evidence: []profileevidence.EvidenceValues{
								{
									FactID:              "target-second",
									Layer:               profileevidence.LayerTargetAnchor,
									Code:                legalquery.EvidenceGeneralTerm,
									IndependentPositive: true,
									ClusterSpan:         true,
								},
								{
									FactID:      "target-first",
									Layer:       profileevidence.LayerTargetAnchor,
									Code:        legalquery.EvidenceGeneralTerm,
									ClusterSpan: true,
								},
							},
						},
						{
							StepID:        "step-one",
							SourceOrdinal: 1,
							TopicOrdinal:  1,
							Evidence: []profileevidence.EvidenceValues{
								{
									FactID:              "semantic",
									Layer:               profileevidence.LayerSemanticExpansion,
									Code:                legalquery.EvidenceLegalConcept,
									IndependentPositive: true,
									ClusterSpan:         true,
								},
								{
									FactID:      "target-later",
									Layer:       profileevidence.LayerTargetAnchor,
									Code:        legalquery.EvidenceOfficialAlias,
									ClusterSpan: true,
								},
								{
									FactID:      "explicit",
									Layer:       profileevidence.LayerExplicitTaskResource,
									Code:        legalquery.EvidenceExplicitTask,
									ClusterSpan: true,
								},
							},
						},
					},
				},
			},
		}

		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		key, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil {
			t.Fatalf("%s: cluster key を作成できません: %v", clusterKeyVerificationID, err)
		}
		if !eligible || key.Canonical() != "1:30:36|2:4:8" {
			t.Fatalf(
				"%s: cluster key が一致しません: eligible=%t key=%q",
				clusterKeyVerificationID,
				eligible,
				key.Canonical(),
			)
		}
		members := key.Members()
		if len(members) != 2 ||
			members[0].TopicOrdinal() != 1 ||
			members[0].EvidenceSpan() != explicit ||
			members[1].TopicOrdinal() != 2 ||
			members[1].EvidenceSpan() != targetFirst {
			t.Fatalf("%s: cluster member が一致しません", clusterKeyVerificationID)
		}

		members[0] = profileevidence.ClusterMember{}
		if key.Members()[0].EvidenceSpan() != explicit {
			t.Fatalf("%s: member 戻り値から key が変更されました", clusterKeyVerificationID)
		}
	})

	t.Run("evidence登録順に依存しない", func(t *testing.T) {
		values := singleStepValues(t)
		first, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: 最初の mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		slices.Reverse(values.Drafts[0].Steps[0].Evidence)
		second, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: 二回目の mapping を作成できません: %v", clusterKeyVerificationID, err)
		}

		firstKey, firstEligible, err := first.ClusterKey("draft-one")
		if err != nil {
			t.Fatalf("%s: 最初の key を作成できません: %v", clusterKeyVerificationID, err)
		}
		secondKey, secondEligible, err := second.ClusterKey("draft-one")
		if err != nil {
			t.Fatalf("%s: 二回目の key を作成できません: %v", clusterKeyVerificationID, err)
		}
		if !firstEligible || !secondEligible ||
			firstKey.Canonical() != secondKey.Canonical() {
			t.Fatalf("%s: evidence 順で key が変わりました", clusterKeyVerificationID)
		}
	})

	t.Run("同じpairを別stepごとに保持する", func(t *testing.T) {
		span := mustSpan(t, 2, 6)
		values := profileevidence.MappingValues{
			ProfileID: "core",
			Facts: []profileevidence.FactValues{
				{FactID: "shared", Span: &span},
			},
			Drafts: []profileevidence.DraftValues{
				{
					DraftID: "draft-one",
					Steps: []profileevidence.StepValues{
						clusterStep("step-one", 1, 1, "shared", true),
						clusterStep("step-two", 2, 1, "shared", false),
					},
				},
			},
		}
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		key, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil {
			t.Fatalf("%s: key を作成できません: %v", clusterKeyVerificationID, err)
		}
		if !eligible || len(key.Members()) != 2 ||
			key.Canonical() != "1:2:6|1:2:6" {
			t.Fatalf("%s: 別 step の同じ pair を失いました", clusterKeyVerificationID)
		}
	})

	t.Run("正の根拠はtopic単位で確認する", func(t *testing.T) {
		firstSpan := mustSpan(t, 0, 2)
		secondSpan := mustSpan(t, 4, 6)
		values := profileevidence.MappingValues{
			ProfileID: "core",
			Facts: []profileevidence.FactValues{
				{FactID: "first", Span: &firstSpan},
				{FactID: "second", Span: &secondSpan},
			},
			Drafts: []profileevidence.DraftValues{
				{
					DraftID: "draft-one",
					Steps: []profileevidence.StepValues{
						clusterStep("step-one", 1, 1, "first", true),
						clusterStep("step-two", 2, 1, "second", false),
					},
				},
			},
		}
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		_, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil || !eligible {
			t.Fatalf("%s: 同じ主題の正の根拠を step ごとに要求しました: %v", clusterKeyVerificationID, err)
		}

		values.Drafts[0].Steps[1].TopicOrdinal = 2
		mapping, err = profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: 二主題 mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		key, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil || eligible || key.Canonical() != "" {
			t.Fatalf("%s: 根拠のない主題を追加分岐にしました: %v", clusterKeyVerificationID, err)
		}
	})

	t.Run("spanなしrefへ偽spanを作らない", func(t *testing.T) {
		values := singleStepValues(t)
		values.Drafts[0].Steps[0].Evidence =
			values.Drafts[0].Steps[0].Evidence[:1]
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: ref mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		key, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil || eligible || len(key.Members()) != 0 || key.Canonical() != "" {
			t.Fatalf("%s: span のない ref へ偽 span を作りました: %v", clusterKeyVerificationID, err)
		}
	})

	t.Run("不正な構造をfail-closedにする", func(t *testing.T) {
		tests := []struct {
			name   string
			change func(*profileevidence.MappingValues)
		}{
			{
				name: "存在しないfact",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[1].FactID = "missing"
				},
			},
			{
				name: "sourceOrdinalが零",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].SourceOrdinal = 0
				},
			},
			{
				name: "topicOrdinalが二から始まる",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].TopicOrdinal = 2
				},
			},
			{
				name: "spanなしfactをclusterへ使う",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[0].ClusterSpan = true
				},
			},
			{
				name: "boundaryへtarget codeを使う",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[0].Code =
						legalquery.EvidenceOfficialAlias
				},
			},
			{
				name: "未知のlayer",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[1].Layer =
						profileevidence.Layer("unknown")
				},
			},
			{
				name: "clarification layerをevidenceに使う",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[1].Layer =
						profileevidence.LayerClarificationOrReject
				},
			},
			{
				name: "未知のcode",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[1].Code =
						legalquery.EvidenceCode("unknown")
				},
			},
			{
				name: "同じfactとcodeのflagが競合する",
				change: func(values *profileevidence.MappingValues) {
					conflict := values.Drafts[0].Steps[0].Evidence[1]
					conflict.IndependentPositive = true
					values.Drafts[0].Steps[0].Evidence = append(
						values.Drafts[0].Steps[0].Evidence,
						conflict,
					)
				},
			},
			{
				name: "誤記補助codeを正の根拠にする",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence = append(
						values.Drafts[0].Steps[0].Evidence,
						profileevidence.EvidenceValues{
							FactID:              "law-name",
							Layer:               profileevidence.LayerTargetAnchor,
							Code:                legalquery.EvidenceUniqueTypoCorrection,
							IndependentPositive: true,
						},
					)
				},
			},
			{
				name: "誤記補助codeをcluster spanにする",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence = append(
						values.Drafts[0].Steps[0].Evidence,
						profileevidence.EvidenceValues{
							FactID:      "law-name",
							Layer:       profileevidence.LayerTargetAnchor,
							Code:        legalquery.EvidenceUniqueTypoCorrection,
							ClusterSpan: true,
						},
					)
				},
			},
			{
				name: "形態素根拠へ一般語codeを重ねる",
				change: func(values *profileevidence.MappingValues) {
					values.Drafts[0].Steps[0].Evidence[1].Layer =
						profileevidence.LayerSemanticExpansion
					values.Drafts[0].Steps[0].Evidence[1].Code =
						legalquery.EvidenceMorphologicalContext
					values.Drafts[0].Steps[0].Evidence = append(
						values.Drafts[0].Steps[0].Evidence,
						profileevidence.EvidenceValues{
							FactID: "law-name",
							Layer:  profileevidence.LayerSemanticExpansion,
							Code:   legalquery.EvidenceGeneralTerm,
						},
					)
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				values := singleStepValues(t)
				test.change(&values)
				if _, err := profileevidence.NewMapping(values); err == nil {
					t.Fatalf("%s: 不正な mapping を受理しました", clusterKeyVerificationID)
				}
			})
		}
	})

	t.Run("誤記補助codeを基本根拠へだけ併記する", func(t *testing.T) {
		values := singleStepValues(t)
		values.Drafts[0].Steps[0].Evidence = append(
			values.Drafts[0].Steps[0].Evidence,
			profileevidence.EvidenceValues{
				FactID: "law-name",
				Layer:  profileevidence.LayerTargetAnchor,
				Code:   legalquery.EvidenceUniqueTypoCorrection,
			},
		)
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: 誤記補助codeを併記できません: %v", clusterKeyVerificationID, err)
		}
		key, eligible, err := mapping.ClusterKey("draft-one")
		if err != nil || !eligible || key.Canonical() != "1:4:10" {
			t.Fatalf("%s: 基本根拠のcluster keyが一致しません: %v", clusterKeyVerificationID, err)
		}
	})

	t.Run("候補draftの共通上限を超えない", func(t *testing.T) {
		values := singleStepValues(t)
		template := values.Drafts[0]
		values.Drafts = make([]profileevidence.DraftValues, 17)
		for index := range values.Drafts {
			current := template
			current.DraftID = "draft-" + string(rune('a'+index))
			values.Drafts[index] = current
		}
		if _, err := profileevidence.NewMapping(values); err == nil {
			t.Fatalf("%s: 17 件の draft を受理しました", clusterKeyVerificationID)
		}
	})

	t.Run("完全に同じevidenceだけを一件へ縮約する", func(t *testing.T) {
		values := singleStepValues(t)
		duplicate := values.Drafts[0].Steps[0].Evidence[1]
		values.Drafts[0].Steps[0].Evidence = append(
			values.Drafts[0].Steps[0].Evidence,
			duplicate,
		)
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: 同じ evidence を縮約できません: %v", clusterKeyVerificationID, err)
		}
		evidence, err := mapping.StepEvidence("draft-one", "step-one")
		if err != nil {
			t.Fatalf("%s: evidence を取得できません: %v", clusterKeyVerificationID, err)
		}
		if len(evidence) != 2 {
			t.Fatalf("%s: 同じ evidence が残りました: %d", clusterKeyVerificationID, len(evidence))
		}
	})

	t.Run("未知draftとstepを拒否する", func(t *testing.T) {
		mapping, err := profileevidence.NewMapping(singleStepValues(t))
		if err != nil {
			t.Fatalf("%s: mapping を作成できません: %v", clusterKeyVerificationID, err)
		}
		if _, _, err := mapping.ClusterKey("missing"); err == nil {
			t.Fatalf("%s: 未知 draft を受理しました", clusterKeyVerificationID)
		}
		if _, err := mapping.StepEvidence("draft-one", "missing"); err == nil {
			t.Fatalf("%s: 未知 step を受理しました", clusterKeyVerificationID)
		}
	})
}

func clusterStep(
	stepID string,
	sourceOrdinal int,
	topicOrdinal int,
	factID string,
	independentPositive bool,
) profileevidence.StepValues {
	return profileevidence.StepValues{
		StepID:        stepID,
		SourceOrdinal: sourceOrdinal,
		TopicOrdinal:  topicOrdinal,
		Evidence: []profileevidence.EvidenceValues{
			{
				FactID:              factID,
				Layer:               profileevidence.LayerTargetAnchor,
				Code:                legalquery.EvidenceGeneralTerm,
				IndependentPositive: independentPositive,
				ClusterSpan:         true,
			},
		},
	}
}
