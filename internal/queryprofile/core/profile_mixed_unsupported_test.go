package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは翻訳または法的助言と混在する弱い取得候補を保持する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		signal legalquery.CandidateGenerationSignal
	}{
		{
			name:   "翻訳",
			query:  "労働時間を検索して英訳してください。",
			signal: legalquery.CandidateSignalUnsupportedTranslation,
		},
		{
			name:   "法的助言",
			query:  "労働時間を検索して法的に判断してください。",
			signal: legalquery.CandidateSignalUnsupportedLegalAdvice,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, nil)
			if !slices.Contains(generation.Signals(), test.signal) {
				t.Fatalf(
					"SOT-MODEL-026: 対象外信号 = %#v, want %q",
					generation.Signals(),
					test.signal,
				)
			}

			candidates := generation.Candidates()
			if len(candidates) != 1 {
				t.Fatalf(
					"SOT-MODEL-026: 混在要求の candidates = %#v, want 1 件",
					candidates,
				)
			}
			evidence := candidates[0].EvidenceCodes()
			if !slices.Contains(evidence, legalquery.EvidenceExplicitTask) {
				t.Fatalf(
					"SOT-MODEL-026: explicit task evidence がありません: %#v",
					evidence,
				)
			}
			if !slices.Contains(evidence, legalquery.EvidenceMorphologicalContext) &&
				!slices.Contains(evidence, legalquery.EvidenceGeneralTerm) {
				t.Fatalf(
					"SOT-MODEL-026: 弱い検索語 evidence がありません: %#v",
					evidence,
				)
			}
			if slices.Contains(evidence, legalquery.EvidenceOfficialIdentifier) ||
				slices.Contains(evidence, legalquery.EvidenceStructuredReference) ||
				slices.Contains(evidence, legalquery.EvidenceExplicitResource) ||
				slices.Contains(evidence, legalquery.EvidenceOfficialAlias) ||
				slices.Contains(evidence, legalquery.EvidenceLegalConcept) {
				t.Fatalf(
					"SOT-MODEL-026: 強い取得根拠を持つ試験になっています: %#v",
					evidence,
				)
			}
		})
	}
}

func TestProfileは対象外要求に従属する取得候補を保持しない(
	t *testing.T,
) {
	t.Parallel()

	for _, query := range []string{
		"賃貸借契約の解除は適法ですか。私が今すぐ解除してよいか判断してください。",
		"国会会議録を検索し、民法改正の立法理由を追跡してください。",
		"一般ウェブ記事と法律事務所ブログを検索して要約してください。",
	} {
		generation := generateQuery(t, query, nil)
		if len(generation.Candidates()) != 0 {
			t.Errorf(
				"SOT-MODEL-026: 対象外要求 %q の candidates = %#v, want 0 件",
				query,
				generation.Candidates(),
			)
		}
	}
}
