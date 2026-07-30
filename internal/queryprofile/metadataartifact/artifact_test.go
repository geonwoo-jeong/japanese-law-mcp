package metadataartifact

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestLoadは閉じたJSONとschemaVersionを検証する(t *testing.T) {
	t.Parallel()

	branchMargin := 12
	validV1 := validProfileJSON(t, 1, nil, "")
	validV2 := validProfileJSON(t, 2, &branchMargin, "")
	tests := map[string][]byte{
		"空入力": nil,
		"上限超過": bytes.Repeat(
			[]byte{' '},
			maximumProfileBytes+1,
		),
		"不正UTF-8": append(
			append([]byte(nil), validV1[:len(validV1)-1]...),
			0xff,
			'}',
		),
		"重複key": []byte(strings.Replace(
			string(validV1),
			`"schemaVersion":1,`,
			`"schemaVersion":1,"schemaVersion":1,`,
			1,
		)),
		"null": []byte(strings.Replace(
			string(validV1),
			`"profileId":"core"`,
			`"profileId":null`,
			1,
		)),
		"未知項目": []byte(strings.Replace(
			string(validV1),
			`"schemaVersion":1,`,
			`"schemaVersion":1,"unknown":true,`,
			1,
		)),
		"二個目の値": append(
			append([]byte(nil), validV1...),
			[]byte(`{}`)...,
		),
		"後方token": append(
			append([]byte(nil), validV1...),
			[]byte(`x`)...,
		),
		"未知schema": []byte(strings.Replace(
			string(validV1),
			`"schemaVersion":1`,
			`"schemaVersion":3`,
			1,
		)),
		"version1へのbranch混入": validProfileJSON(
			t,
			1,
			&branchMargin,
			"",
		),
		"version2のbranch欠落": validProfileJSON(t, 2, nil, ""),
		"必須整数の欠落": []byte(strings.Replace(
			string(validV1),
			`"minimum":0,`,
			"",
			1,
		)),
		"lexiconsの欠落": []byte(strings.Replace(
			string(validV1),
			`,"lexicons":{"lawNames":"law-name-v1","legalConcepts":"legal-concept-v1"}`,
			"",
			1,
		)),
	}
	for name, input := range tests {
		name := name
		input := input
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(input); err == nil {
				t.Fatal(
					"profile-metadata-closed-json: 不正な成果物を受理しました",
				)
			}
		})
	}

	reordered := []byte(strings.Replace(
		string(validV2),
		`"schemaVersion":2,"profileId":"core",`,
		`"profileId":"core","schemaVersion":2,`,
		1,
	))
	if _, err := Load(reordered); err != nil {
		t.Fatalf(
			"profile-metadata-closed-json: object key 順の変更を拒否しました: %v",
			err,
		)
	}
}

func TestLoadはbranchRetentionMarginの存在状態を保持する(
	t *testing.T,
) {
	t.Parallel()

	v1, err := Load(validProfileJSON(t, 1, nil, ""))
	if err != nil {
		t.Fatalf(
			"profile-metadata-schema-versions: version 1 を読めません: %v",
			err,
		)
	}
	if value, present := v1.Metadata().Selection().
		BranchRetentionMargin(); present || value != 0 {
		t.Fatalf(
			"profile-metadata-branch-retention-presence: version 1 = (%d, %t)",
			value,
			present,
		)
	}

	zero := 0
	v2, err := Load(validProfileJSON(t, 2, &zero, ""))
	if err != nil {
		t.Fatalf(
			"profile-metadata-schema-versions: version 2 を読めません: %v",
			err,
		)
	}
	if value, present := v2.Metadata().Selection().
		BranchRetentionMargin(); !present || value != 0 {
		t.Fatalf(
			"profile-metadata-branch-retention-presence: version 2 = (%d, %t)",
			value,
			present,
		)
	}
}

func TestLoadは条件付きtieBreakを不変に保持する(t *testing.T) {
	t.Parallel()

	const conditional = `"conditionalTieBreaks":{` +
		`"lawAliasCollisionGroupsOverCandidateLimit":[` +
		`"evidence_set","step_count","source_position","meaning_signature"` +
		`]},`
	margin := 12
	artifact, err := Load(validProfileJSON(
		t,
		2,
		&margin,
		conditional,
	))
	if err != nil {
		t.Fatalf(
			"profile-metadata-conditional-tie-break: 読み込めません: %v",
			err,
		)
	}
	if !artifact.ConditionalTieBreaksPresent() {
		t.Fatal("条件付き tie-break field の存在状態を失いました")
	}
	conditionName :=
		legalquery.ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit
	conditionalValues := artifact.Metadata().ConditionalTieBreaks()
	order := conditionalValues[conditionName]
	if len(order) != 4 ||
		order[2] != legalquery.QueryTieBreakSourcePosition {
		t.Fatalf("条件付き tie-break = %#v", conditionalValues)
	}
	order[0] = legalquery.QueryTieBreakMeaningSignature
	delete(
		conditionalValues,
		conditionName,
	)
	again := artifact.Metadata().ConditionalTieBreaks()
	if again[conditionName][0] != legalquery.QueryTieBreakEvidenceSet {
		t.Fatal(
			"profile-metadata-immutability: 条件付き tie-break を変更できました",
		)
	}

	unknown := strings.Replace(
		string(validProfileJSON(t, 2, &margin, conditional)),
		"lawAliasCollisionGroupsOverCandidateLimit",
		"unknownCondition",
		1,
	)
	if _, err := Load([]byte(unknown)); err == nil {
		t.Fatal(
			"profile-metadata-conditional-tie-break: 未知条件を受理しました",
		)
	}

	absent, err := Load(validProfileJSON(t, 2, &margin, ""))
	if err != nil {
		t.Fatalf("条件付き tie-break なしの成果物を読めません: %v", err)
	}
	if absent.ConditionalTieBreaksPresent() {
		t.Fatal("存在しない条件付き tie-break field を存在ありとして扱いました")
	}
	empty, err := Load(validProfileJSON(
		t,
		2,
		&margin,
		`"conditionalTieBreaks":{},`,
	))
	if err != nil {
		t.Fatalf("空の条件付き tie-break object を読めません: %v", err)
	}
	if !empty.ConditionalTieBreaksPresent() {
		t.Fatal("空の条件付き tie-break field の存在状態を失いました")
	}
}

func validProfileJSON(
	t *testing.T,
	schemaVersion int,
	branchMargin *int,
	conditional string,
) []byte {
	t.Helper()

	branch := ""
	if branchMargin != nil {
		branch = fmt.Sprintf(
			`,"branchRetentionMargin":%d`,
			*branchMargin,
		)
	}
	return []byte(fmt.Sprintf(
		`{`+
			`"schemaVersion":%d,`+
			`"profileId":"core",`+
			`"profileVersion":"core-v2",`+
			`"rankingVersion":"ranking-v2",`+
			`"cueSetVersion":"core-cues-v2",`+
			`"targets":[{`+
			`"task":"search",`+
			`"resource":"law",`+
			`"inputKind":"law_search"`+
			`}],`+
			`"score":{`+
			`"minimum":0,`+
			`"maximum":405,`+
			`"evidenceWeights":[`+
			`{"code":"official_identifier","weight":90},`+
			`{"code":"structured_reference","weight":80},`+
			`{"code":"explicit_task","weight":60},`+
			`{"code":"explicit_resource","weight":50},`+
			`{"code":"official_alias","weight":40},`+
			`{"code":"legal_concept","weight":35},`+
			`{"code":"morphological_context","weight":25},`+
			`{"code":"unique_typo_correction","weight":15},`+
			`{"code":"general_term","weight":10}`+
			`],`+
			`"highConfidenceAt":130,`+
			`"mediumConfidenceAt":80`+
			`},`+
			`"selection":{`+
			`"singleThreshold":85,`+
			`"minimumExecutionThreshold":80,`+
			`"singleMargin":25,`+
			`"hedgeMargin":10%s`+
			`},`+
			`"tieBreak":[`+
			`"evidence_set",`+
			`"step_count",`+
			`"meaning_signature",`+
			`"source_position"`+
			`],`+
			`%s`+
			`"lexicons":{`+
			`"lawNames":"law-name-v1",`+
			`"legalConcepts":"legal-concept-v1"`+
			`}`+
			`}`,
		schemaVersion,
		branch,
		conditional,
	))
}
