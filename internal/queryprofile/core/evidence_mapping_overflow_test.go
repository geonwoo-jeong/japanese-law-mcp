package core

import (
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCoreEvidenceMappingはRawDraft上限より先に同値縮約する(t *testing.T) {
	const verificationID = "core-evidence-mapping-post-retention-limit"

	fixture := mustCoreEvidenceFixture(
		t,
		"民法（法令ID 129AC0000000089）という法令を読んでください。",
		nil,
	)
	template := mustCoreSingleKindDraft(
		t,
		fixture.drafts,
		legalquery.InputKindLawRead,
	)
	rawDrafts := make([]candidateDraft, maximumGeneratedCandidates+1)
	for index := range rawDrafts {
		rawDrafts[index] = cloneCoreEvidenceTestDraft(template)
	}

	profile := mustCoreEvidenceProfile(t)
	materialize := func() (
		[]legalquery.LegalQueryCandidate,
		[][]int,
		bool,
		error,
	) {
		scope, err := legalquery.NewCandidateIDScope(1)
		if err != nil {
			return nil, nil, false, err
		}
		return profile.materializeCoreEvidenceCandidates(
			fixture.input,
			fixture.cues,
			rawDrafts,
			scope,
		)
	}

	first, firstStarts, firstForced, err := materialize()
	if err != nil {
		t.Fatalf(
			"%s: %d 件の raw draft を縮約前の共通上限で拒否しました: %v",
			verificationID,
			len(rawDrafts),
			err,
		)
	}
	if len(first) != 1 || len(first) > maximumGeneratedCandidates || firstForced {
		t.Fatalf(
			"%s: 同値縮約後の candidates=%d forced=%t、期待値は 1 件かつ強制明確化なしです",
			verificationID,
			len(first),
			firstForced,
		)
	}

	repeated, repeatedStarts, repeatedForced, err := materialize()
	if err != nil {
		t.Fatalf("%s: 再実行に失敗しました: %v", verificationID, err)
	}
	if !reflect.DeepEqual(first, repeated) ||
		!reflect.DeepEqual(firstStarts, repeatedStarts) ||
		firstForced != repeatedForced {
		t.Fatalf(
			"%s: 同じ raw draft から決定的な結果を作れませんでした",
			verificationID,
		)
	}
}
