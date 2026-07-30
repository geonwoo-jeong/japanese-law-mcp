package core

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCueLoaderSchemaV3は全cueの構文roleを必須にする(
	t *testing.T,
) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("cue-loader-schema-v3: LoadEmbedded() のエラー = %v", err)
	}
	cues := profile.CueVocabulary()
	if profile.Metadata().SchemaVersion() != 1 {
		t.Fatalf(
			"cue-loader-schema-v3: profile schemaVersion = %d",
			profile.Metadata().SchemaVersion(),
		)
	}
	if len(cues) == 0 {
		t.Fatal("cue-loader-schema-v3: cue vocabulary が空です")
	}
	for index, cue := range cues {
		if cue.MatchGroup == "" ||
			!testValidCueSyntaxRole(cue.SyntaxRole) {
			t.Fatalf(
				"cue-loader-schema-v3: cues[%d] = %#v",
				index,
				cue,
			)
		}
	}

	cues[0].SyntaxRole = legalquery.CueSyntaxRole("changed")
	if profile.CueVocabulary()[0].SyntaxRole ==
		legalquery.CueSyntaxRole("changed") {
		t.Fatal("cue-loader-schema-v3: getter から syntaxRole を変更できました")
	}
}

func TestCueLoaderSchemaV3は構文roleを意味ごとに分離する(
	t *testing.T,
) {
	t.Parallel()

	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() のエラー = %v", err)
	}
	roles := make(map[string]legalquery.CueSyntaxRole)
	for _, cue := range profile.CueVocabulary() {
		roles[cue.CueID] = cue.SyntaxRole
	}
	expected := map[string]legalquery.CueSyntaxRole{
		"syntax-task-predicate-create":                 legalquery.CueSyntaxRoleTaskPredicate,
		"task-read":                                    legalquery.CueSyntaxRoleTaskPredicate,
		"task-search":                                  legalquery.CueSyntaxRoleTaskPredicate,
		"unsupported-explicit-task-expression":         legalquery.CueSyntaxRoleTaskExpression,
		"unsupported-legal-advice-expression":          legalquery.CueSyntaxRoleTaskExpression,
		"unsupported-legal-advice-object":              legalquery.CueSyntaxRoleTaskObject,
		"unsupported-relationship-analysis-expression": legalquery.CueSyntaxRoleTaskExpression,
		"unsupported-relationship-analysis-object":     legalquery.CueSyntaxRoleTaskObject,
		"unsupported-translation-expression":           legalquery.CueSyntaxRoleTaskExpression,
		"unsupported-translation-object":               legalquery.CueSyntaxRoleTaskObject,
		"unsupported-version-comparison-expression":    legalquery.CueSyntaxRoleTaskExpression,
		"unsupported-version-comparison-object":        legalquery.CueSyntaxRoleTaskObject,
	}
	for cueID, want := range expected {
		if got := roles[cueID]; got != want {
			t.Fatalf(
				"cue-loader-schema-v3: cue %q の syntaxRole = %q、期待値は %q です",
				cueID,
				got,
				want,
			)
		}
	}
}

func TestCueLoaderSchemaV3は旧版欠落未知roleを拒否する(
	t *testing.T,
) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	tests := map[string][]byte{
		"schema version 1": bytes.Replace(
			embeddedCues,
			[]byte(`"schemaVersion": 3,`),
			[]byte(`"schemaVersion": 1,`),
			1,
		),
		"schema version 2": bytes.Replace(
			embeddedCues,
			[]byte(`"schemaVersion": 3,`),
			[]byte(`"schemaVersion": 2,`),
			1,
		),
		"未知の schema version": bytes.Replace(
			embeddedCues,
			[]byte(`"schemaVersion": 3,`),
			[]byte(`"schemaVersion": 99,`),
			1,
		),
		"syntaxRole 欠落": bytes.Replace(
			embeddedCues,
			[]byte(`      "syntaxRole": "none",`+"\n"),
			nil,
			1,
		),
		"未知の syntaxRole": bytes.Replace(
			embeddedCues,
			[]byte(`"syntaxRole": "none"`),
			[]byte(`"syntaxRole": "unknown"`),
			1,
		),
	}
	for name, cues := range tests {
		name, cues := name, cues
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
				t.Fatal("cue-loader-schema-v3: 不正な cue data を受理しました")
			}
		})
	}
}

func TestCueLoaderSchemaV3は必須対象外意図群を要求する(
	t *testing.T,
) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	var document map[string]any
	if err := json.Unmarshal(embeddedCues, &document); err != nil {
		t.Fatalf("cue data を解析できません: %v", err)
	}
	values, ok := document["cues"].([]any)
	if !ok {
		t.Fatal("cue data に cues 配列がありません")
	}
	filtered := make([]any, 0, len(values))
	for _, value := range values {
		cue, cueOK := value.(map[string]any)
		if !cueOK {
			t.Fatal("cue data に object ではない値があります")
		}
		if cue["intentGroup"] == "explicit_out_of_scope_task" {
			continue
		}
		filtered = append(filtered, cue)
	}
	if len(filtered) == len(values) {
		t.Fatal("explicit_out_of_scope_task の fixture がありません")
	}
	document["cues"] = filtered
	invalid, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("fixture を JSON 化できません: %v", err)
	}
	if _, err := Load(
		embeddedProfile,
		invalid,
		lawNames,
		concepts,
	); err == nil {
		t.Fatal("SOT-ENG-028: 必須 intentGroup の欠落を受理しました")
	}
}

func TestCueLoaderSchemaV3は対象外意図の閉じた対応を検証する(
	t *testing.T,
) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	var document map[string]any
	if err := json.Unmarshal(embeddedCues, &document); err != nil {
		t.Fatalf("cue data を解析できません: %v", err)
	}
	byID := cuesByID(t, document)
	byID["unsupported-legal-advice-expression"]["intentGroup"] = "translation"
	invalid, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("fixture を作成できません: %v", err)
	}
	if _, err := Load(
		embeddedProfile,
		invalid,
		lawNames,
		concepts,
	); err == nil {
		t.Fatal("SOT-ENG-028: intentGroup/value/signal の不一致を受理しました")
	}
}

func testValidCueSyntaxRole(value legalquery.CueSyntaxRole) bool {
	switch value {
	case legalquery.CueSyntaxRoleNone,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate:
		return true
	default:
		return false
	}
}
