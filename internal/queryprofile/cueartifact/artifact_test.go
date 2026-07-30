package cueartifact

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestLoadはschemaVersion3の閉じたJSONだけを受理する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"最上位の未知項目": strings.Replace(
			validCueJSON(),
			`"schemaVersion":3`,
			`"schemaVersion":3,"unknown":true`,
			1,
		),
		"cue の未知項目": strings.Replace(
			validCueJSON(),
			`"syntaxRole":"task_predicate"`,
			`"syntaxRole":"task_predicate","unknown":true`,
			1,
		),
		"最上位の重複 key": strings.Replace(
			validCueJSON(),
			`"schemaVersion":3`,
			`"schemaVersion":3,"schemaVersion":3`,
			1,
		),
		"cue の重複 key": strings.Replace(
			validCueJSON(),
			`"cueId":"task-search"`,
			`"cueId":"task-search","cueId":"task-search"`,
			1,
		),
		"trailing value": validCueJSON() + `{}`,
		"途中 EOF":         strings.TrimSuffix(validCueJSON(), `}`),
		"最上位が配列":         `[]`,
		"null":           strings.Replace(validCueJSON(), `"task"`, `null`, 1),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("cue-loader-schema-v3: 不正な JSON を受理しました")
			}
		})
	}
}

func TestLoadは旧版未知版と不正なsyntaxRoleを拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"schema version 1": strings.Replace(
			validCueJSON(),
			`"schemaVersion":3`,
			`"schemaVersion":1`,
			1,
		),
		"schema version 2": strings.Replace(
			validCueJSON(),
			`"schemaVersion":3`,
			`"schemaVersion":2`,
			1,
		),
		"未知 schema version": strings.Replace(
			validCueJSON(),
			`"schemaVersion":3`,
			`"schemaVersion":99`,
			1,
		),
		"syntaxRole 欠落": strings.Replace(
			validCueJSON(),
			`,"syntaxRole":"task_predicate"`,
			"",
			1,
		),
		"未知 syntaxRole": strings.Replace(
			validCueJSON(),
			`"syntaxRole":"task_predicate"`,
			`"syntaxRole":"unknown"`,
			1,
		),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("cue-loader-schema-v3: 未対応の cue data を受理しました")
			}
		})
	}
}

func TestLoadはIDと配列の完全順序を検証する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"profileId": strings.Replace(
			validCueJSON(),
			`"profileId":"core"`,
			`"profileId":"Core"`,
			1,
		),
		"cueSetVersion": strings.Replace(
			validCueJSON(),
			`"cueSetVersion":"core-cues-v1"`,
			`"cueSetVersion":"core_cues_v1"`,
			1,
		),
		"cueId": strings.Replace(
			validCueJSON(),
			`"cueId":"task-search"`,
			`"cueId":"task_search"`,
			1,
		),
		"cueId 順序": cueJSON(
			`{"cueId":"b","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索"]},` +
				`{"cueId":"a","category":"task","value":"read","syntaxRole":"task_predicate","terms":["読む"]}`,
		),
		"cueId 重複": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索"]},` +
				`{"cueId":"a","category":"task","value":"read","syntaxRole":"task_predicate","terms":["読む"]}`,
		),
		"term 順序": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索してください","検索"]}`,
		),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("SOT-ENG-030: 不正な ID または配列順を受理しました")
			}
		})
	}
}

func TestLoadは全syntaxRoleを受理する(t *testing.T) {
	t.Parallel()

	value := cueJSON(
		`{"cueId":"a","category":"resource","value":"law","syntaxRole":"none","terms":["法令"]},` +
			`{"cueId":"b","category":"task","value":"search","syntaxRole":"task_expression","terms":["探してください"]},` +
			`{"cueId":"c","category":"unsupported","value":"translation","intentGroup":"translation","signal":"unsupported_translation","syntaxRole":"task_object","terms":["翻訳"]},` +
			`{"cueId":"d","category":"task","value":"read","syntaxRole":"task_predicate","terms":["読んでください"]}`,
	)
	artifact, err := Load([]byte(value))
	if err != nil {
		t.Fatalf("cue-loader-schema-v3: Load() のエラー = %v", err)
	}
	entries := artifact.Entries()
	want := []legalquery.CueSyntaxRole{
		legalquery.CueSyntaxRoleNone,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate,
	}
	if len(entries) != len(want) {
		t.Fatalf("entries の件数 = %d、期待値は %d です", len(entries), len(want))
	}
	for index, role := range want {
		if entries[index].SyntaxRole() != role {
			t.Fatalf(
				"entries[%d].SyntaxRole() = %q、期待値は %q です",
				index,
				entries[index].SyntaxRole(),
				role,
			)
		}
	}
}

func TestLoadは比較用正規化語の同一owner重複を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"同じ entry": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索 して","検索して"]}`,
		),
		"異なる cueId": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索 して"]},` +
				`{"cueId":"b","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索して"]}`,
		),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load([]byte(value)); err == nil {
				t.Fatal(
					"cue-loader-duplicate-term-owner: 正規化語の重複を受理しました",
				)
			}
		})
	}
}

func TestLoadは同じtupleの長短語を受理する(t *testing.T) {
	t.Parallel()

	artifact, err := Load([]byte(cueJSON(
		`{"cueId":"a","category":"unsupported","value":"translation","intentGroup":"translation","signal":"unsupported_translation","syntaxRole":"task_expression","terms":["英語に翻訳"]},` +
			`{"cueId":"b","category":"unsupported","value":"translation","intentGroup":"translation","signal":"unsupported_translation","syntaxRole":"task_expression","terms":["英語に翻訳してください"]}`,
	)))
	if err != nil {
		t.Fatalf(
			"cue-loader-longest-same-tuple: 同じ tuple の長短語を拒否しました: %v",
			err,
		)
	}
	if len(artifact.Entries()) != 2 {
		t.Fatalf("cue-loader-longest-same-tuple: entries = %#v", artifact.Entries())
	}
	vocabulary := artifact.Vocabulary()
	if len(vocabulary) != 2 ||
		vocabulary[0].MatchGroup == "" ||
		vocabulary[0].MatchGroup != vocabulary[1].MatchGroup {
		t.Fatalf(
			"cue-loader-longest-same-tuple: 同じ tuple の match group = %#v",
			vocabulary,
		)
	}
}

func TestLoadは異なるtupleのprefix衝突を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"value": cueJSON(
			`{"cueId":"a","category":"task","value":"read","syntaxRole":"task_predicate","terms":["検索"]},` +
				`{"cueId":"b","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索してください"]}`,
		),
		"optional field の有無": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_expression","terms":["検索"]},` +
				`{"cueId":"b","category":"task","value":"search","intentGroup":"search","syntaxRole":"task_expression","terms":["検索してください"]}`,
		),
		"syntaxRole": cueJSON(
			`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_expression","terms":["検索"]},` +
				`{"cueId":"b","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索してください"]}`,
		),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load([]byte(value)); err == nil {
				t.Fatal(
					"cue-loader-tuple-conflict: 異なる tuple の prefix 衝突を受理しました",
				)
			}
		})
	}
}

func TestLoadCrossProfileReuseはprofile間の同じ正規化語を許可する(
	t *testing.T,
) {
	t.Parallel()

	core, err := Load([]byte(cueJSON(
		`{"cueId":"a","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索"]}`,
	)))
	if err != nil {
		t.Fatalf("cue-loader-cross-profile-reuse: core の Load() = %v", err)
	}
	judicialJSON := strings.Replace(
		cueJSON(
			`{"cueId":"a","category":"resource","value":"judicial_decision","syntaxRole":"none","terms":["検索"]}`,
		),
		`"profileId":"core"`,
		`"profileId":"judicial-cases"`,
		1,
	)
	judicial, err := Load([]byte(judicialJSON))
	if err != nil {
		t.Fatalf(
			"cue-loader-cross-profile-reuse: judicial-cases の Load() = %v",
			err,
		)
	}
	coreVocabulary := core.Vocabulary()
	judicialVocabulary := judicial.Vocabulary()
	if len(coreVocabulary) != 1 ||
		len(judicialVocabulary) != 1 ||
		coreVocabulary[0].ProfileID != "core" ||
		judicialVocabulary[0].ProfileID != "judicial-cases" {
		t.Fatalf(
			"cue-loader-cross-profile-reuse: profile が分離されていません: %#v, %#v",
			coreVocabulary,
			judicialVocabulary,
		)
	}
}

func TestArtifactはprofileとcueSetVersionの一致を検証する(t *testing.T) {
	t.Parallel()

	artifact, err := Load([]byte(validCueJSON()))
	if err != nil {
		t.Fatalf("Load() のエラー = %v", err)
	}
	if err := artifact.MatchProfile("core", "core-cues-v1"); err != nil {
		t.Fatalf("cue-loader-profile-version-match: 一致する版のエラー = %v", err)
	}
	for _, values := range [][2]string{
		{"other", "core-cues-v1"},
		{"core", "other-cues-v1"},
	} {
		if err := artifact.MatchProfile(values[0], values[1]); err == nil {
			t.Fatal("cue-loader-profile-version-match: 不一致の版を受理しました")
		}
	}
	var nilArtifact *Artifact
	if err := nilArtifact.MatchProfile("core", "core-cues-v1"); err == nil {
		t.Fatal("cue-loader-profile-version-match: nil artifact を受理しました")
	}
}

func TestArtifactのgetterと意味validatorは内部状態を変更させない(t *testing.T) {
	t.Parallel()

	artifact, err := Load([]byte(validCueJSON()))
	if err != nil {
		t.Fatalf("Load() のエラー = %v", err)
	}
	entries := artifact.Entries()
	terms := entries[0].Terms()
	terms[0] = "変更"
	vocabulary := artifact.Vocabulary()
	vocabulary[0].ProfileID = "changed"
	vocabulary[0].MatchGroup = "changed"
	vocabulary[0].Terms[0] = "変更"

	validatorError := errors.New("検証失敗")
	if err := artifact.ValidateEntries(func(entry Entry) error {
		callbackTerms := entry.Terms()
		callbackTerms[0] = "変更"
		return validatorError
	}); !errors.Is(err, validatorError) {
		t.Fatalf("意味 validator のエラー = %v", err)
	}

	const readers = 16
	var wait sync.WaitGroup
	wait.Add(readers)
	for range readers {
		go func() {
			defer wait.Done()
			current := artifact.Entries()
			current[0].Terms()[0] = "変更"
			currentVocabulary := artifact.Vocabulary()
			currentVocabulary[0].Terms[0] = "変更"
		}()
	}
	wait.Wait()

	nextEntries := artifact.Entries()
	nextVocabulary := artifact.Vocabulary()
	if nextEntries[0].Terms()[0] != "検索" ||
		nextVocabulary[0].ProfileID != "core" ||
		nextVocabulary[0].MatchGroup == "changed" ||
		nextVocabulary[0].Terms[0] != "検索" {
		t.Fatal("SOT-ENG-030: getter または validator から内部状態を変更できました")
	}
}

func TestLoadはJSON安全境界を検証する(t *testing.T) {
	t.Parallel()

	invalidUTF8 := append([]byte(validCueJSON()), 0xff)
	if _, err := Load(invalidUTF8); err == nil {
		t.Fatal("無効な UTF-8 を受理しました")
	}

	deep := `{"schemaVersion":3,"profileId":"core","cueSetVersion":"core-cues-v1","cues":`
	for range 16 {
		deep += "["
	}
	deep += "0"
	for range 16 {
		deep += "]"
	}
	deep += "}"
	if _, err := Load([]byte(deep)); err == nil {
		t.Fatal("JSON depth の上限超過を受理しました")
	}

	if _, err := Load([]byte(strings.Repeat(" ", maximumArtifactBytes+1))); err == nil {
		t.Fatal("file size の上限超過を受理しました")
	}
}

func validCueJSON() string {
	return cueJSON(
		`{"cueId":"task-search","category":"task","value":"search","syntaxRole":"task_predicate","terms":["検索","検索してください"]}`,
	)
}

func cueJSON(entries string) string {
	return `{"schemaVersion":3,"profileId":"core","cueSetVersion":"core-cues-v1","cues":[` +
		entries + `]}`
}
