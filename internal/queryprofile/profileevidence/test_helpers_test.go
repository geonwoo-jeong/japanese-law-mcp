package profileevidence_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func mustSpan(t testing.TB, startByte int, endByte int) legalquery.QuerySpan {
	t.Helper()

	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf("query span を作成できません: %v", err)
	}
	return span
}

func singleStepValues(t testing.TB) profileevidence.MappingValues {
	t.Helper()

	targetSpan := mustSpan(t, 4, 10)
	return profileevidence.MappingValues{
		ProfileID: "core",
		Facts: []profileevidence.FactValues{
			{
				FactID: "input-ref",
			},
			{
				FactID: "law-name",
				Span:   &targetSpan,
			},
		},
		Drafts: []profileevidence.DraftValues{
			{
				DraftID: "draft-one",
				Steps: []profileevidence.StepValues{
					{
						StepID:               "step-one",
						SourceOrdinal:        1,
						TopicOrdinal:         1,
						StepMeaningSignature: "law-search",
						Evidence: []profileevidence.EvidenceValues{
							{
								FactID:              "input-ref",
								Layer:               profileevidence.LayerBoundary,
								Code:                legalquery.EvidenceOfficialIdentifier,
								IndependentPositive: true,
							},
							{
								FactID:      "law-name",
								Layer:       profileevidence.LayerTargetAnchor,
								Code:        legalquery.EvidenceOfficialAlias,
								ClusterSpan: true,
							},
						},
					},
				},
			},
		},
	}
}
