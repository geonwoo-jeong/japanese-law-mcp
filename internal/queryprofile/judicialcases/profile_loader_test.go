package judicialcases

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func TestLoadEmbeddedは裁判例二能力と共有校正を固定する(t *testing.T) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-025: LoadEmbedded() のエラー = %v", err)
	}
	metadata := profile.Metadata()
	if metadata.ProfileID() != "judicial-cases" ||
		metadata.SchemaVersion() != 1 ||
		metadata.ProfileVersion() != "judicial-cases-2026-07-30-9" ||
		metadata.RankingVersion() != "legal-query-ranking-2026-07-28-1" ||
		metadata.CueSetVersion() != "judicial-cases-cues-2026-07-30-4" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if value, present := metadata.Selection().
		BranchRetentionMargin(); present || value != 0 {
		t.Fatalf(
			"profile-metadata-branch-retention-presence: active = (%d, %t)",
			value,
			present,
		)
	}
	if conditional := metadata.ConditionalTieBreaks(); len(conditional) != 0 {
		t.Fatalf(
			"profile-metadata-conditional-tie-break: conditional = %#v",
			conditional,
		)
	}
	const lawVersion = "e-gov-law-api-v2-laws-2026-07-27+ndl-common-abbreviations-2026-07-27"
	if metadata.LawNameLexiconVersion() != lawVersion ||
		metadata.LegalConceptLexiconVersion() != "legal-concept-2026-07-30-2" {
		t.Fatalf(
			"lexicon versions = %q, %q",
			metadata.LawNameLexiconVersion(),
			metadata.LegalConceptLexiconVersion(),
		)
	}

	targets := metadata.Targets()
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindJudicialDecisionSearch,
		legalquery.InputKindJudicialDecisionRead,
	}
	if len(targets) != len(wantKinds) {
		t.Fatalf("targets = %#v", targets)
	}
	for index, target := range targets {
		if target.InputKind() != wantKinds[index] {
			t.Fatalf("targets[%d] = %#v", index, target)
		}
	}

	score := metadata.Score()
	selection := metadata.Selection()
	if score.Minimum() != 0 ||
		score.Maximum() != 405 ||
		score.HighConfidenceAt() != 130 ||
		score.MediumConfidenceAt() != 80 ||
		selection.SingleThreshold() != 85 ||
		selection.MinimumExecutionThreshold() != 80 ||
		selection.SingleMargin() != 25 ||
		selection.HedgeMargin() != 10 {
		t.Fatalf("共有校正が core と一致しません: score=%#v selection=%#v", score, selection)
	}
	if len(profile.CueVocabulary()) == 0 {
		t.Fatal("cue vocabulary が空です")
	}

	var _ legalquery.QueryProfile = profile
}

func TestProfileGetterはcueとmetadataを変更させない(t *testing.T) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() のエラー = %v", err)
	}
	cues := profile.CueVocabulary()
	cues[0].ProfileID = "changed"
	cues[0].MatchGroup = "changed"
	cues[0].Terms[0] = "changed"
	targets := profile.Metadata().Targets()
	targets[0] = legalquery.QueryProfileTarget{}

	nextCues := profile.CueVocabulary()
	if nextCues[0].ProfileID != "judicial-cases" ||
		nextCues[0].MatchGroup == "changed" ||
		nextCues[0].Terms[0] == "changed" ||
		profile.Metadata().Targets()[0].InputKind() !=
			legalquery.InputKindJudicialDecisionSearch {
		t.Fatal("SOT-ENG-025: profile getter から内部状態を変更できました")
	}
}

func TestLoadは閉じたJSONとProfile固有契約を厳格に検証する(t *testing.T) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	tests := map[string]struct {
		profile []byte
		cues    []byte
	}{
		"profile 未知項目": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"schemaVersion": 1,`),
				[]byte(`"schemaVersion": 1, "unknown": true,`),
				1,
			),
			cues: embeddedCues,
		},
		"profile trailing": {
			profile: append(
				append([]byte(nil), embeddedProfile...),
				[]byte(`{}`)...,
			),
			cues: embeddedCues,
		},
		"profile 重複 key": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"schemaVersion": 1,`),
				[]byte(`"schemaVersion": 1, "schemaVersion": 1,`),
				1,
			),
			cues: embeddedCues,
		},
		"未採用の条件付き順位": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"lexicons": {`),
				[]byte(
					`"conditionalTieBreaks": {`+
						`"lawAliasCollisionGroupsOverCandidateLimit": [`+
						`"evidence_set", "step_count", `+
						`"source_position", "meaning_signature"`+
						`]`+
						`}, "lexicons": {`,
				),
				1,
			),
			cues: embeddedCues,
		},
		"未採用の空条件付き順位": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"lexicons": {`),
				[]byte(`"conditionalTieBreaks": {}, "lexicons": {`),
				1,
			),
			cues: embeddedCues,
		},
		"cue 未知項目": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"schemaVersion": 3,`),
				[]byte(`"schemaVersion": 3, "unknown": true,`),
				1,
			),
		},
		"cue 重複 key": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"schemaVersion": 3,`),
				[]byte(`"schemaVersion": 3, "schemaVersion": 3,`),
				1,
			),
		},
		"cue 版不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`judicial-cases-cues-2026-07-30-4`),
				[]byte(`judicial-cases-cues-2026-07-29-9`),
				1,
			),
		},
		"profile ID 不一致": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"profileId": "judicial-cases"`),
				[]byte(`"profileId": "core"`),
				1,
			),
			cues: embeddedCues,
		},
		"法概念辞書版不一致": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`legal-concept-2026-07-30-2`),
				[]byte(`legal-concept-2026-07-30-9`),
				1,
			),
			cues: embeddedCues,
		},
		"target 不一致": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"inputKind": "judicial_decision_search"`),
				[]byte(`"inputKind": "law_search"`),
				1,
			),
			cues: embeddedCues,
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(
				test.profile,
				test.cues,
				lawNames,
				concepts,
			); err == nil {
				t.Fatal("不正な profile data を受理しました")
			}
		})
	}
}

func TestLoadは裁判例に必要な意味の欠落と任意項目を拒否する(
	t *testing.T,
) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	var document map[string]any
	if err := json.Unmarshal(embeddedCues, &document); err != nil {
		t.Fatalf("cue data を解析できません: %v", err)
	}
	rawCues, ok := document["cues"].([]any)
	if !ok {
		t.Fatal("cue data に cues 配列がありません")
	}
	filtered := make([]any, 0, len(rawCues))
	for _, raw := range rawCues {
		cue, cueOK := raw.(map[string]any)
		if !cueOK {
			t.Fatal("cue data に object ではない値があります")
		}
		if cue["category"] == "resource" &&
			cue["value"] == "judicial_decision" {
			continue
		}
		filtered = append(filtered, cue)
	}
	document["cues"] = filtered
	missing, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("意味欠落 fixture を作成できません: %v", err)
	}
	if _, err := Load(
		embeddedProfile,
		missing,
		lawNames,
		concepts,
	); err == nil {
		t.Fatal("裁判例 profile に必要な意味の欠落を受理しました")
	}

	if err := json.Unmarshal(embeddedCues, &document); err != nil {
		t.Fatalf("cue data を再解析できません: %v", err)
	}
	rawCues = document["cues"].([]any)
	rawCues[0].(map[string]any)["intentGroup"] = "translation"
	optional, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("任意項目 fixture を作成できません: %v", err)
	}
	if _, err := Load(
		embeddedProfile,
		optional,
		lawNames,
		concepts,
	); err == nil {
		t.Fatal("裁判例 cue の intentGroup を受理しました")
	}
}

func TestLoadは共有辞書依存の欠落を起動時に拒否する(t *testing.T) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	if _, err := Load(
		embeddedProfile,
		embeddedCues,
		nil,
		concepts,
	); err == nil {
		t.Fatal("nil law name lexicon を受理しました")
	}
	if _, err := Load(
		embeddedProfile,
		embeddedCues,
		lawNames,
		nil,
	); err == nil {
		t.Fatal("nil legal concept lexicon を受理しました")
	}
}

func TestLoadは法令名辞書版ずれで起動を失敗させる(t *testing.T) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	profileJSON := bytes.Replace(
		embeddedProfile,
		[]byte(`e-gov-law-api-v2-laws-2026-07-27+ndl-common-abbreviations-2026-07-27`),
		[]byte(`unused-law-name-lexicon-2026-07-28-1`),
		1,
	)
	if _, err := Load(
		profileJSON,
		embeddedCues,
		lawNames,
		concepts,
	); err == nil {
		t.Fatal("profile-metadata-profile-ownership: 法令名辞書版ずれを受理しました")
	}
}

func TestLoadは不正なcue語彙を拒否する(t *testing.T) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	tests := map[string][]byte{
		"空配列": bytes.Replace(
			embeddedCues,
			[]byte(`"cues": [`),
			[]byte(`"cues": []`),
			1,
		),
		"未対応 category": bytes.Replace(
			embeddedCues,
			[]byte(`"category": "operator"`),
			[]byte(`"category": "unknown"`),
			1,
		),
		"空 term": bytes.Replace(
			embeddedCues,
			[]byte(`"それぞれ"`),
			[]byte(`""`),
			1,
		),
		"重複 term": bytes.Replace(
			embeddedCues,
			[]byte("\"それぞれ\",\n        \"について\""),
			[]byte("\"それぞれ\",\n        \"それぞれ\""),
			1,
		),
		"cueId 逆順": bytes.Replace(
			embeddedCues,
			[]byte(`"cueId": "operator-individual"`),
			[]byte(`"cueId": "z-operator-individual"`),
			1,
		),
	}
	for name, cues := range tests {
		name := name
		cues := cues
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if bytes.Equal(cues, embeddedCues) {
				t.Fatalf("fixture %q を変更できませんでした", name)
			}
			if _, err := Load(
				embeddedProfile,
				cues,
				lawNames,
				concepts,
			); err == nil {
				t.Fatal("不正な cue data を受理しました")
			}
		})
	}
}

func mustEmbeddedConceptLexicon(
	t *testing.T,
) *legalconceptlexicon.Lexicon {
	t.Helper()

	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法概念辞書を読み込めません: %v", err)
	}
	return concepts
}

func mustEmbeddedLawNameLexicon(t *testing.T) *lawnamelexicon.Lexicon {
	t.Helper()

	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法令名辞書を読み込めません: %v", err)
	}
	return lawNames
}
