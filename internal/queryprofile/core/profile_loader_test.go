package core

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func TestLoadEmbeddedは法令コア五能力と辞書版を固定する(t *testing.T) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-025: LoadEmbedded() のエラー = %v", err)
	}
	metadata := profile.Metadata()
	if metadata.ProfileID() != "core" ||
		metadata.SchemaVersion() != 1 ||
		metadata.ProfileVersion() != "core-2026-07-30-33" ||
		metadata.RankingVersion() != "legal-query-ranking-2026-07-28-1" ||
		metadata.CueSetVersion() != "core-cues-2026-07-30-15" {
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
	conditionName :=
		legalquery.ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit
	conditional := metadata.ConditionalTieBreaks()
	if len(conditional) != 1 ||
		!slices.Equal(
			conditional[conditionName],
			[]legalquery.QueryTieBreak{
				legalquery.QueryTieBreakEvidenceSet,
				legalquery.QueryTieBreakStepCount,
				legalquery.QueryTieBreakSourcePosition,
				legalquery.QueryTieBreakMeaningSignature,
			},
		) {
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
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead,
		legalquery.InputKindLawUpdates,
	}
	if len(targets) != len(wantKinds) {
		t.Fatalf("targets = %#v", targets)
	}
	for index, target := range targets {
		if target.InputKind() != wantKinds[index] {
			t.Fatalf("targets[%d] = %#v", index, target)
		}
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
	conditionName :=
		legalquery.ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit
	conditional := profile.Metadata().ConditionalTieBreaks()
	conditional[conditionName][0] =
		legalquery.QueryTieBreakMeaningSignature

	nextCues := profile.CueVocabulary()
	if nextCues[0].ProfileID != "core" ||
		nextCues[0].MatchGroup == "changed" ||
		nextCues[0].Terms[0] == "changed" ||
		profile.Metadata().Targets()[0].InputKind() != legalquery.InputKindLawSearch ||
		profile.Metadata().ConditionalTieBreaks()[conditionName][0] !=
			legalquery.QueryTieBreakEvidenceSet {
		t.Fatal("SOT-ENG-025: profile getter から内部状態を変更できました")
	}
}

func TestLoadは未知項目trailing値辞書版不一致を拒否する(t *testing.T) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
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
			profile: append(append([]byte(nil), embeddedProfile...), []byte(`{}`)...),
			cues:    embeddedCues,
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
		"profile 不正 UTF-8": {
			profile: append([]byte{0xff}, embeddedProfile...),
			cues:    embeddedCues,
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
		"cue schema version 不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"schemaVersion": 3,`),
				[]byte(`"schemaVersion": 99,`),
				1,
			),
		},
		"cue profile ID 不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"profileId": "core",`),
				[]byte(`"profileId": "other",`),
				1,
			),
		},
		"profile と cue の所有 ID がともに不一致": {
			profile: bytes.Replace(
				embeddedProfile,
				[]byte(`"profileId": "core",`),
				[]byte(`"profileId": "other",`),
				1,
			),
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"profileId": "core",`),
				[]byte(`"profileId": "other",`),
				1,
			),
		},
		"cue ID 順序不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"cueId": "operator-all"`),
				[]byte(`"cueId": "zz-order"`),
				1,
			),
		},
		"廃止 cue meaning": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"value": "all"`),
				[]byte(`"value": "document_article"`),
				1,
			),
		},
		"cue 版不一致": {
			profile: embeddedProfile,
			cues: bytes.Replace(
				embeddedCues,
				[]byte(`"cueSetVersion": "`),
				[]byte(`"cueSetVersion": "mismatch-`),
				1,
			),
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
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(test.profile, test.cues, lawNames, concepts); err == nil {
				t.Fatal("不正な profile data を受理しました")
			}
		})
	}
}

func TestEmbeddedCueDataは対象外意図群とsignalを明示する(t *testing.T) {
	t.Parallel()

	var document map[string]any
	if err := json.Unmarshal(embeddedCues, &document); err != nil {
		t.Fatalf("cue data を解析できません: %v", err)
	}
	cues, ok := document["cues"].([]any)
	if !ok {
		t.Fatal("cue data に cues 配列がありません")
	}
	unsupportedCount := 0
	for index, value := range cues {
		cue, cueOK := value.(map[string]any)
		if !cueOK || cue["category"] != "unsupported" {
			continue
		}
		unsupportedCount++
		if intentGroup, groupOK := cue["intentGroup"].(string); !groupOK ||
			intentGroup == "" {
			t.Fatalf(
				"SOT-ENG-028: cues[%d].intentGroup が明示されていません",
				index,
			)
		}
		if signal, signalOK := cue["signal"].(string); !signalOK ||
			signal == "" {
			t.Fatalf(
				"SOT-ENG-028: cues[%d].signal が明示されていません",
				index,
			)
		}
	}
	if unsupportedCount == 0 {
		t.Fatal("SOT-ENG-028: 対象外 cue がありません")
	}
}

func TestLoadは対象外cueの不正schemaとsignal衝突を拒否する(t *testing.T) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	tests := map[string]func(map[string]any){
		"intentGroup 欠落": func(document map[string]any) {
			cue := firstUnsupportedCue(t, document)
			delete(cue, "intentGroup")
		},
		"未知の signal": func(document map[string]any) {
			cue := firstUnsupportedCue(t, document)
			cue["signal"] = "unsupported_unknown"
		},
		"異なる signal 間の正規化語衝突": func(document map[string]any) {
			byID := cuesByID(t, document)
			legalAdvice := byID["unsupported-legal-advice-expression"]
			translation := byID["unsupported-translation-expression"]
			legalAdvice["terms"] = append(
				legalAdvice["terms"].([]any),
				"比較　してください",
			)
			translation["terms"] = append(
				translation["terms"].([]any),
				"比較してください",
			)
		},
	}
	for name, mutate := range tests {
		name := name
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var document map[string]any
			if err := json.Unmarshal(embeddedCues, &document); err != nil {
				t.Fatalf("cue data を解析できません: %v", err)
			}
			mutate(document)
			invalid, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("不正 cue data を作成できません: %v", err)
			}
			if _, err := Load(
				embeddedProfile,
				invalid,
				lawNames,
				concepts,
			); err == nil {
				t.Fatal("SOT-ENG-028: 不正な対象外 cue data を受理しました")
			}
		})
	}
}

func firstUnsupportedCue(
	t *testing.T,
	document map[string]any,
) map[string]any {
	t.Helper()
	for _, cue := range cuesByID(t, document) {
		if cue["category"] == "unsupported" {
			return cue
		}
	}
	t.Fatal("対象外 cue がありません")
	return nil
}

func cuesByID(
	t *testing.T,
	document map[string]any,
) map[string]map[string]any {
	t.Helper()
	values, ok := document["cues"].([]any)
	if !ok {
		t.Fatal("cue data に cues 配列がありません")
	}
	result := make(map[string]map[string]any, len(values))
	for index, value := range values {
		cue, cueOK := value.(map[string]any)
		if !cueOK {
			t.Fatalf("cues[%d] が object ではありません", index)
		}
		cueID, idOK := cue["cueId"].(string)
		if !idOK || cueID == "" {
			t.Fatalf("cues[%d].cueId がありません", index)
		}
		result[cueID] = cue
	}
	return result
}

func TestLoadは法令別名衝突上限の条件付き順位不一致を拒否する(
	t *testing.T,
) {
	t.Parallel()

	var document map[string]any
	if err := json.Unmarshal(embeddedProfile, &document); err != nil {
		t.Fatalf("profile data を解析できません: %v", err)
	}
	conditional, ok := document["conditionalTieBreaks"].(map[string]any)
	if !ok {
		t.Fatal("条件付き tie-break の宣言がありません")
	}
	conditional["lawAliasCollisionGroupsOverCandidateLimit"] = []any{
		"evidence_set",
		"step_count",
		"meaning_signature",
		"source_position",
	}
	invalid, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("不正 profile data を作成できません: %v", err)
	}
	lawNames, concepts := mustEmbeddedLexicons(t)
	if _, err := Load(invalid, embeddedCues, lawNames, concepts); err == nil {
		t.Fatal("条件付き tie-break の不一致を受理しました")
	}
}

func TestLoadは辞書依存の欠落を起動時に拒否する(t *testing.T) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	if _, err := Load(embeddedProfile, embeddedCues, nil, concepts); err == nil {
		t.Fatal("nil law name lexicon を受理しました")
	}
	if _, err := Load(embeddedProfile, embeddedCues, lawNames, nil); err == nil {
		t.Fatal("nil legal concept lexicon を受理しました")
	}
}

func mustEmbeddedLexicons(
	t *testing.T,
) (*lawnamelexicon.Lexicon, *legalconceptlexicon.Lexicon) {
	t.Helper()
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法令名辞書を読み込めません: %v", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("法概念辞書を読み込めません: %v", err)
	}
	return lawNames, concepts
}
