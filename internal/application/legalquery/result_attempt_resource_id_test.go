package legalquery

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLegalQueryJudicialAttemptPreservesProviderOwnedResourceID(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	const providerOwnedID = "vendor:decision/95570/revision-current"
	item := attemptTestSourced(
		t,
		fixtures.judicialInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "courts-hanrei-html",
			ResourceType: "judicial-decision",
			ResourceID:   providerOwnedID,
		},
		fixtures.judicialSummary.Data(),
	)
	attempt, err := NewLegalQueryJudicialSearchAttempt(
		LegalQueryJudicialSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step: planTestStep(
				t,
				"裁判例検索",
				"step-provider-owned-id",
			),
			Page: resultTestPage(
				t,
				1,
				false,
				1,
				model.TotalRelationExact,
			),
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{
				item,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-016/SOT-IF-041: provider 所有の resourceId を拒否しました: %v", err)
	}
	if got := attempt.Items()[0].Ref().Key().ResourceID(); got != providerOwnedID {
		t.Fatalf("SOT-MODEL-024: resourceId = %q", got)
	}
}
