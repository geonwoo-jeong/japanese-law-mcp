package judicialcases

import (
	"context"
	_ "embed"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

var (
	//go:embed testdata/relation-v2/profile.json
	relationV2ProfileJSON []byte
)

func TestRelationV2ProfileはPositiveCueRoleとActive版の分離を固定する(
	t *testing.T,
) {
	t.Parallel()

	profile := mustRelationV2Profile(t)
	for cueID, definition := range profile.cueByID {
		want := legalquery.CueSyntaxRoleNone
		if definition.category == "task" {
			want = legalquery.CueSyntaxRoleTaskExpression
		}
		if definition.syntaxRole != want {
			t.Fatalf(
				"positive-cue-value-role-mapping: cue %q (%s/%s) の role = %q、期待値は %q",
				cueID,
				definition.category,
				definition.value,
				definition.syntaxRole,
				want,
			)
		}
	}

	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("active profile を読み込めません: %v", err)
	}
	activeMargin, activeMarginPresent :=
		active.Metadata().Selection().BranchRetentionMargin()
	nextMargin, nextMarginPresent :=
		profile.Metadata().Selection().BranchRetentionMargin()
	if active.Metadata().SchemaVersion() != 1 ||
		activeMarginPresent ||
		activeMargin != 0 ||
		active.Metadata().ProfileVersion() !=
			"judicial-cases-2026-07-30-9" ||
		active.Metadata().CueSetVersion() !=
			"judicial-cases-cues-2026-07-30-4" ||
		profile.Metadata().SchemaVersion() != 2 ||
		profile.Metadata().ProfileVersion() !=
			"judicial-cases-2026-07-31-11" ||
		profile.Metadata().RankingVersion() !=
			"legal-query-ranking-2026-07-31-2" ||
		!nextMarginPresent ||
		nextMargin != 12 ||
		profile.Metadata().CueSetVersion() !=
			"judicial-cases-cues-2026-07-30-4" {
		t.Fatalf(
			"SOT-ARCH-033: active=%q/%q next=%q/%q",
			active.Metadata().ProfileVersion(),
			active.Metadata().CueSetVersion(),
			profile.Metadata().ProfileVersion(),
			profile.Metadata().CueSetVersion(),
		)
	}
}

func TestRelationV2Profileは元Profileの内部状態を共有しない(
	t *testing.T,
) {
	t.Parallel()

	lawNames := mustEmbeddedLawNameLexicon(t)
	concepts := mustEmbeddedConceptLexicon(t)
	original, err := Load(
		relationV2ProfileJSON,
		embeddedCues,
		lawNames,
		concepts,
	)
	if err != nil {
		t.Fatalf("relation v2 profile を読み込めません: %v", err)
	}
	cloned, err := newCueTaskRelationV2Profile(original)
	if err != nil {
		t.Fatalf("relation v2 profile を準備できません: %v", err)
	}

	originalCueID := original.cues[0].CueID
	originalTerm := original.cues[0].Terms[0]
	originalDefinition := original.cueByID["task-search"]
	var conceptID string
	for current := range original.concepts {
		conceptID = current
		break
	}
	if conceptID == "" {
		t.Fatal("clone 検証に必要な concept がありません")
	}
	cloned.cues[0].CueID = ""
	cloned.cues[0].Terms[0] = ""
	delete(cloned.cueByID, "task-search")
	delete(cloned.concepts, conceptID)

	if original.cues[0].CueID != originalCueID ||
		original.cues[0].Terms[0] != originalTerm {
		t.Fatal("next profile の cue slice が元 profile と共有されています")
	}
	if original.cueByID["task-search"] != originalDefinition {
		t.Fatal("next profile の cue map が元 profile と共有されています")
	}
	if _, exists := original.concepts[conceptID]; !exists {
		t.Fatal("next profile の concept map が元 profile と共有されています")
	}
}

func TestRelationV2ProfileはDirectTaskだけを明示Taskにする(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name              string
		query             string
		wantCandidates    int
		wantRelationKinds []legalquery.CueTaskRelationKind
		wantExplicitTask  bool
	}{
		{
			name:           "裁判例検索の direct task",
			query:          "医療過誤の裁判例を検索してください。",
			wantCandidates: 1,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantExplicitTask: true,
		},
		{
			name:           "という文言に接続した bare mention",
			query:          "医療過誤の裁判例を検索してくださいという文言",
			wantCandidates: 0,
		},
		{
			name:           "引用句の resource mention",
			query:          "「裁判例」を含む条文を検索してください。",
			wantCandidates: 0,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			preprocessed, generation :=
				generateRelationV2Query(t, test.query)
			relations := preprocessed.CueTaskRelations()
			kinds := make(
				[]legalquery.CueTaskRelationKind,
				0,
				len(relations),
			)
			for _, relation := range relations {
				if relation.Subject().ProfileID() == profileID {
					kinds = append(kinds, relation.Kind())
				}
			}
			if !slices.Equal(kinds, test.wantRelationKinds) {
				t.Fatalf(
					"cue-relation-positive-direct-task: kinds = %#v、期待値は %#v",
					kinds,
					test.wantRelationKinds,
				)
			}
			candidates := generation.Candidates()
			if len(candidates) != test.wantCandidates {
				t.Fatalf(
					"positive-cue-bare-mention-rejected: candidates = %#v",
					candidates,
				)
			}
			for _, candidate := range candidates {
				if slices.Contains(
					candidate.EvidenceCodes(),
					legalquery.EvidenceExplicitTask,
				) != test.wantExplicitTask {
					t.Fatalf(
						"positive-cue-bare-mention-rejected: evidence = %#v",
						candidate.EvidenceCodes(),
					)
				}
			}
		})
	}
}

func mustRelationV2Profile(t *testing.T) *Profile {
	t.Helper()

	profile, err := Load(
		relationV2ProfileJSON,
		embeddedCues,
		mustEmbeddedLawNameLexicon(t),
		mustEmbeddedConceptLexicon(t),
	)
	if err != nil {
		t.Fatalf("relation v2 profile を読み込めません: %v", err)
	}
	profile, err = newCueTaskRelationV2Profile(profile)
	if err != nil {
		t.Fatalf("relation v2 profile を準備できません: %v", err)
	}
	return profile
}

func generateRelationV2Query(
	t *testing.T,
	query string,
) (legalquery.PreprocessResult, legalquery.CandidateGeneration) {
	t.Helper()

	profile := mustRelationV2Profile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("relation v2 preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("candidate generation input を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("candidate scope を構築できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("Generate() のエラー = %v", err)
	}
	return preprocessed, generation
}
