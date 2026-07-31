package profileevidence_test

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const mappingLifetimeVerificationID = "profile-private-evidence-mapping-lifetime"

func TestProfilePrivateEvidenceMappingLifetime(t *testing.T) {
	t.Run("入力と戻り値を深く複製する", func(t *testing.T) {
		values := singleStepValues(t)
		mapping, err := profileevidence.NewMapping(values)
		if err != nil {
			t.Fatalf("%s: mapping を作成できません: %v", mappingLifetimeVerificationID, err)
		}

		replacement := mustSpan(t, 20, 24)
		*values.Facts[1].Span = replacement
		values.Facts[1].FactID = "changed-fact"
		values.Drafts[0].DraftID = "changed-draft"
		values.Drafts[0].Steps[0].StepID = "changed-step"
		values.Drafts[0].Steps[0].Evidence[0].FactID = "changed-evidence"

		evidence, err := mapping.StepEvidence("draft-one", "step-one")
		if err != nil {
			t.Fatalf("%s: step evidence を取得できません: %v", mappingLifetimeVerificationID, err)
		}
		if mapping.ProfileID() != "core" || len(evidence) != 2 {
			t.Fatalf("%s: mapping の内容が一致しません", mappingLifetimeVerificationID)
		}
		if evidence[0].Layer() != profileevidence.LayerBoundary ||
			evidence[0].Code() != legalquery.EvidenceOfficialIdentifier ||
			!evidence[0].IndependentPositive() ||
			evidence[0].ClusterSpan() {
			t.Fatalf("%s: boundary evidence が一致しません", mappingLifetimeVerificationID)
		}
		if _, exists := evidence[0].Span(); exists {
			t.Fatalf("%s: 入力 ref が span を持ちました", mappingLifetimeVerificationID)
		}
		if evidence[1].Layer() != profileevidence.LayerTargetAnchor ||
			evidence[1].Code() != legalquery.EvidenceOfficialAlias ||
			evidence[1].IndependentPositive() ||
			!evidence[1].ClusterSpan() {
			t.Fatalf("%s: target evidence が一致しません", mappingLifetimeVerificationID)
		}
		span, exists := evidence[1].Span()
		if !exists || span != mustSpan(t, 4, 10) {
			t.Fatalf("%s: constructor 後に span が変更されました", mappingLifetimeVerificationID)
		}

		evidence[0] = profileevidence.Evidence{}
		again, err := mapping.StepEvidence("draft-one", "step-one")
		if err != nil || len(again) != 2 ||
			again[0].FactID() != "input-ref" ||
			again[1].FactID() != "law-name" {
			t.Fatalf("%s: 戻り値から内部状態が変更されました: %v", mappingLifetimeVerificationID, err)
		}
	})

	t.Run("同じ局所IDを別の評価へ保持しない", func(t *testing.T) {
		firstValues := singleStepValues(t)
		first, err := profileevidence.NewMapping(firstValues)
		if err != nil {
			t.Fatalf("%s: 最初の mapping を作成できません: %v", mappingLifetimeVerificationID, err)
		}

		secondValues := singleStepValues(t)
		secondValues.ProfileID = "judicial-cases"
		second, err := profileevidence.NewMapping(secondValues)
		if err != nil {
			t.Fatalf("%s: 二回目の mapping を作成できません: %v", mappingLifetimeVerificationID, err)
		}
		if first.ProfileID() != "core" || second.ProfileID() != "judicial-cases" {
			t.Fatalf("%s: profile 間で mapping が混在しました", mappingLifetimeVerificationID)
		}
	})

	t.Run("並行するrequestが状態を共有しない", func(t *testing.T) {
		const workers = 8

		var wait sync.WaitGroup
		errors := make(chan error, workers)
		for worker := range workers {
			wait.Add(1)
			go func() {
				defer wait.Done()

				values := singleStepValues(t)
				values.ProfileID = fmt.Sprintf("profile-%d", worker+1)
				mapping, err := profileevidence.NewMapping(values)
				if err != nil {
					errors <- err
					return
				}
				key, eligible, err := mapping.ClusterKey("draft-one")
				if err != nil {
					errors <- err
					return
				}
				if !eligible || key.Canonical() != "1:4:10" {
					errors <- fmt.Errorf("cluster key が一致しません: %q", key.Canonical())
				}
			}()
		}
		wait.Wait()
		close(errors)

		for err := range errors {
			t.Fatalf("%s: 並行評価が失敗しました: %v", mappingLifetimeVerificationID, err)
		}
	})

	t.Run("長寿命modelに一時対応を追加しない", func(t *testing.T) {
		values := []reflect.Type{
			reflect.TypeOf(legalquery.CandidateGenerationInput{}),
			reflect.TypeOf(legalquery.LegalQueryCandidate{}),
			reflect.TypeOf(legalquery.CandidateGeneration{}),
			reflect.TypeOf(legalquery.QueryProfileSetResult{}),
		}
		for _, value := range values {
			for index := range value.NumField() {
				name := strings.ToLower(value.Field(index).Name)
				if strings.Contains(name, "evidencemapping") ||
					strings.Contains(name, "evidencespan") ||
					strings.Contains(name, "clusterkey") ||
					strings.Contains(name, "topicordinal") {
					t.Fatalf(
						"%s: %s に一時 field %q があります",
						mappingLifetimeVerificationID,
						value.Name(),
						value.Field(index).Name,
					)
				}
			}
		}
	})
}
