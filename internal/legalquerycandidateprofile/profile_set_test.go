package legalquerycandidateprofile

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

func TestLoadは校正済み二Profileを固定順で構成する(t *testing.T) {
	t.Parallel()

	set, err := Load()
	if err != nil {
		t.Fatalf("SOT-ENG-038: 候補 profile set を構成できません: %v", err)
	}
	metadata := set.ProfileMetadata()
	if len(metadata) != 2 ||
		metadata[0].ProfileID() != "core" ||
		metadata[0].ProfileVersion() != "core-2026-07-31-38" ||
		metadata[1].ProfileID() != "judicial-cases" ||
		metadata[1].ProfileVersion() != "judicial-cases-2026-07-31-12" ||
		set.Profiles().RankingVersion() != "legal-query-ranking-2026-07-31-2" ||
		set.Profiles().ProfileVersion() != "profile-set-sha256-0b00c3409408684b825f3c0bdf1c874bdc99e5383564d8e6b66fe83d4e417a69" {
		t.Fatalf(
			"SOT-ENG-038: 候補 identity が固定値と一致しません: metadata=%#v profileSetVersion=%q",
			metadata,
			set.Profiles().ProfileVersion(),
		)
	}
}

func TestLoadはActiveProfileSetを変更しない(t *testing.T) {
	t.Parallel()

	candidate, err := Load()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: 候補 profile set を構成できません: %v", err)
	}
	active, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: active profile set を構成できません: %v", err)
	}
	if candidate.Profiles().ProfileVersion() == active.Profiles().ProfileVersion() ||
		active.Profiles().ProfileVersion() != "profile-set-sha256-be9ce1499a7b6708a162c4ae2f4da9a340ed2883d3bd3480b2ec21989d11bf8f" {
		t.Fatal("SOT-ARCH-033: 候補と active の profile set が分離されていません")
	}
}

func TestLoadは対象外意図をProfileSetへ保持する(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantSignal legalquery.CandidateGenerationSignal
	}{
		{
			name:       "法的助言",
			query:      "この契約に署名すべきか法的に判断してください。",
			wantSignal: legalquery.CandidateSignalUnsupportedLegalAdvice,
		},
		{
			name:       "対象外資源の横断検索",
			query:      "都道府県の未公開内部文書を横断検索してください。",
			wantSignal: legalquery.CandidateSignalUnsupportedTaskOrResource,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set, err := Load()
			if err != nil {
				t.Fatalf("候補 profile set を構成できません: %v", err)
			}
			request, err := legalquery.NewRequest(
				legalquery.RequestValues{Query: test.query},
			)
			if err != nil {
				t.Fatalf("request を構成できません: %v", err)
			}
			preprocessed, err := set.Preprocessor().Preprocess(
				context.Background(),
				request,
			)
			if err != nil {
				t.Fatalf("候補前処理に失敗しました: %v", err)
			}
			result, err := set.Profiles().Collect(preprocessed)
			if err != nil {
				t.Fatalf("候補集約に失敗しました: %v", err)
			}
			if !slices.Contains(result.Signals(), test.wantSignal) {
				t.Fatalf(
					"SOT-MODEL-030: signals=%#v, want %q; cueMentions=%#v relations=%#v",
					result.Signals(),
					test.wantSignal,
					preprocessed.CueMentions(),
					preprocessed.CueTaskRelations(),
				)
			}
			if len(result.RankedCandidates()) != 0 {
				t.Fatalf(
					"SOT-ENG-028: 対象外意図だけの候補=%#v",
					result.RankedCandidates(),
				)
			}
		})
	}
}

func TestLoadは丁寧形の明示法令検索を保持する(
	t *testing.T,
) {
	set, err := Load()
	if err != nil {
		t.Fatalf("候補 profile set を構成できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "制度について法情報を探したいです。",
	})
	if err != nil {
		t.Fatalf("request を構成できません: %v", err)
	}
	preprocessed, err := set.Preprocessor().Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("候補前処理に失敗しました: %v", err)
	}
	result, err := set.Profiles().Collect(preprocessed)
	if err != nil {
		t.Fatalf("候補集約に失敗しました: %v", err)
	}
	candidates := result.RankedCandidates()
	wantEvidence := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceMorphologicalContext,
	}
	if len(candidates) != 1 ||
		len(candidates[0].Steps()) != 1 ||
		candidates[0].Steps()[0].InputKind() != legalquery.InputKindLawSearch ||
		!slices.Equal(candidates[0].EvidenceCodes(), wantEvidence) {
		t.Fatalf(
			"SOT-MODEL-030: polite law search candidates=%#v; cueMentions=%#v relations=%#v queryTerms=%#v",
			candidates,
			preprocessed.CueMentions(),
			preprocessed.CueTaskRelations(),
			preprocessed.QueryTermMentions(),
		)
	}
}
