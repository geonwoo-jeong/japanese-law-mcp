package judicialcases

import (
	"bytes"
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
		want := legalquery.CueSyntaxRoleNone
		if cue.CueID == "task-read" || cue.CueID == "task-search" {
			want = legalquery.CueSyntaxRoleTaskExpression
		}
		if cue.SyntaxRole != want {
			t.Fatalf(
				"cue-loader-schema-v3: cue %q の syntaxRole = %q、期待値は %q です",
				cue.CueID,
				cue.SyntaxRole,
				want,
			)
		}
	}

	cues[0].SyntaxRole = legalquery.CueSyntaxRole("changed")
	if profile.CueVocabulary()[0].SyntaxRole ==
		legalquery.CueSyntaxRole("changed") {
		t.Fatal("cue-loader-schema-v3: getter から syntaxRole を変更できました")
	}
}

func TestCueLoaderSchemaV3は旧版欠落未知roleを拒否する(
	t *testing.T,
) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
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
		"task predicate": bytes.Replace(
			embeddedCues,
			[]byte(`"syntaxRole": "task_expression"`),
			[]byte(`"syntaxRole": "task_predicate"`),
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
