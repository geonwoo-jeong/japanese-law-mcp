package judicialcases

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const judicialEvidenceRawDraftOverflowID = "judicial-evidence-raw-draft-overflow"

func TestJudicialEvidenceMappingはRawDraft上限より先に同値縮約する(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)
	input, cues, draft := judicialEvidenceOverflowFixture(t, profile)
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf(
			"%s: candidate scope を作成できません: %v",
			judicialEvidenceRawDraftOverflowID,
			err,
		)
	}

	var baseline []materializedCandidate
	for _, count := range []int{16, 17, 64} {
		t.Run(fmt.Sprintf("同値raw-draft-%d件", count), func(t *testing.T) {
			drafts := repeatedJudicialEvidenceDrafts(draft, count)
			before := cloneJudicialEvidenceDrafts(drafts)
			first, forced, materializeErr :=
				profile.materializeJudicialEvidenceCandidateRecords(
					input,
					cues,
					drafts,
					scope,
				)
			if materializeErr != nil {
				t.Fatalf(
					"%s: raw draft %d 件のエラー = %v",
					judicialEvidenceRawDraftOverflowID,
					count,
					materializeErr,
				)
			}
			second, forcedAgain, materializeErr :=
				profile.materializeJudicialEvidenceCandidateRecords(
					input,
					cues,
					drafts,
					scope,
				)
			if materializeErr != nil {
				t.Fatalf(
					"%s: raw draft %d 件の再実行エラー = %v",
					judicialEvidenceRawDraftOverflowID,
					count,
					materializeErr,
				)
			}
			if forced || forcedAgain {
				t.Fatalf(
					"%s: raw draft %d 件で clarification を強制しました",
					judicialEvidenceRawDraftOverflowID,
					count,
				)
			}
			if len(first) != 1 || len(second) != 1 {
				t.Fatalf(
					"%s: raw draft %d 件の最終候補 = %d/%d、期待値は 1/1",
					judicialEvidenceRawDraftOverflowID,
					count,
					len(first),
					len(second),
				)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatalf(
					"%s: raw draft %d 件の再実行結果が一致しません",
					judicialEvidenceRawDraftOverflowID,
					count,
				)
			}
			if baseline == nil {
				baseline = first
			} else if !reflect.DeepEqual(baseline, first) ||
				!reflect.DeepEqual(
					baseline[0].candidate.EvidenceCodes(),
					first[0].candidate.EvidenceCodes(),
				) ||
				baseline[0].candidate.SemanticScore() !=
					first[0].candidate.SemanticScore() ||
				baseline[0].candidate.Confidence() !=
					first[0].candidate.Confidence() {
				t.Fatalf(
					"%s: raw draft 件数で candidate/evidence/score/confidence が変わりました",
					judicialEvidenceRawDraftOverflowID,
				)
			}
			if !reflect.DeepEqual(before, drafts) {
				t.Fatalf(
					"%s: raw draft %d 件の入力を変更しました",
					judicialEvidenceRawDraftOverflowID,
					count,
				)
			}
		})
	}

	t.Run("非同値の最終候補17件", func(t *testing.T) {
		prepared := make([]judicialEvidencePreparedDraft, 17)
		for index := range prepared {
			prepared[index] = judicialEvidencePreparedDraft{
				cluster:   fmt.Sprintf("cluster-%d", index+1),
				score:     100,
				signature: fmt.Sprintf("meaning-%d", index+1),
			}
		}
		_, _, retainErr := profile.retainJudicialEvidenceBranches(prepared)
		if retainErr == nil ||
			!slices.Contains([]string{retainErr.Error()},
				"judicial-cases profile の候補は 16 件以下でなければなりません") {
			t.Fatalf(
				"%s: 非同値の最終候補17件のエラー = %v",
				judicialEvidenceRawDraftOverflowID,
				retainErr,
			)
		}
	})
}

func judicialEvidenceOverflowFixture(
	t *testing.T,
	profile *Profile,
) (
	legalquery.CandidateGenerationInput,
	resolvedCues,
	candidateDraft,
) {
	t.Helper()

	preprocessed := preprocessJudicialEvidenceQuery(
		t,
		profile,
		"医療過誤の裁判例を検索してください。",
		nil,
		judicialEvidenceRawDraftOverflowID,
	)
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf(
			"%s: candidate input を作成できません: %v",
			judicialEvidenceRawDraftOverflowID,
			err,
		)
	}
	cues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf(
			"%s: cue を解決できません: %v",
			judicialEvidenceRawDraftOverflowID,
			err,
		)
	}
	cues, err = profile.resolveRelationV2Cues(input, cues)
	if err != nil {
		t.Fatalf(
			"%s: relation cue を解決できません: %v",
			judicialEvidenceRawDraftOverflowID,
			err,
		)
	}
	drafts, tooMany, ambiguous, err :=
		profile.buildJudicialEvidenceSearchDrafts(input, cues)
	if err != nil {
		t.Fatalf(
			"%s: search draft を作成できません: %v",
			judicialEvidenceRawDraftOverflowID,
			err,
		)
	}
	if tooMany || ambiguous || len(drafts) != 1 {
		t.Fatalf(
			"%s: fixture draft = %d、tooMany/ambiguous = %t/%t",
			judicialEvidenceRawDraftOverflowID,
			len(drafts),
			tooMany,
			ambiguous,
		)
	}
	return input, cues, drafts[0]
}

func repeatedJudicialEvidenceDrafts(
	value candidateDraft,
	count int,
) []candidateDraft {
	result := make([]candidateDraft, 0, count)
	for range count {
		result = append(result, cloneJudicialDraft(value))
	}
	return result
}

func cloneJudicialEvidenceDrafts(values []candidateDraft) []candidateDraft {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		result = append(result, cloneJudicialDraft(value))
	}
	return result
}
