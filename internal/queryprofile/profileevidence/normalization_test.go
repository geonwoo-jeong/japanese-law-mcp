package profileevidence_test

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func TestProfileEvidenceStepNormalization(t *testing.T) {
	t.Run("同じgroupだけを閉じた優越表で正規化する", func(t *testing.T) {
		mapping := mustNormalizationMapping(t, []profileevidence.EvidenceValues{
			normalizationEvidence("identifier", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialIdentifier, "target-one"),
			normalizationEvidence("alias", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias, "target-one"),
			normalizationEvidence("concept", profileevidence.LayerSemanticExpansion, legalquery.EvidenceLegalConcept, "target-one"),
			normalizationEvidence("morph", profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext, "target-one"),
			normalizationEvidence("general", profileevidence.LayerTargetAnchor, legalquery.EvidenceGeneralTerm, "target-one"),
			normalizationEvidence("structured", profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference, "target-one"),
			normalizationEvidence("task", profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask, ""),
		})
		values, err := mapping.NormalizedStepEvidence("draft-one", "step-one")
		if err != nil {
			t.Fatalf("正規化済み根拠を取得できません: %v", err)
		}
		if got := sortedEvidenceCodes(values); !slices.Equal(got, []legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceStructuredReference,
		}) {
			t.Fatalf("閉じた優越表の結果 = %v", got)
		}
	})

	t.Run("別groupとgroupなしの根拠を保持する", func(t *testing.T) {
		mapping := mustNormalizationMapping(t, []profileevidence.EvidenceValues{
			normalizationEvidence("identifier", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialIdentifier, "law-target"),
			normalizationEvidence("structured", profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference, "law-target"),
			normalizationEvidence("concept", profileevidence.LayerSemanticExpansion, legalquery.EvidenceLegalConcept, "content-target"),
			normalizationEvidence("alias", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias, "content-target"),
			normalizationEvidence("general", profileevidence.LayerTargetAnchor, legalquery.EvidenceGeneralTerm, ""),
		})
		values, err := mapping.NormalizedStepEvidence("draft-one", "step-one")
		if err != nil {
			t.Fatalf("正規化済み根拠を取得できません: %v", err)
		}
		if got := sortedEvidenceCodes(values); !slices.Equal(got, []legalquery.EvidenceCode{
			legalquery.EvidenceGeneralTerm,
			legalquery.EvidenceLegalConcept,
			legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceStructuredReference,
		}) {
			t.Fatalf("別groupの根拠を除去しました: %v", got)
		}
	})

	t.Run("閉じた優越表の各下位codeだけを除去する", func(t *testing.T) {
		tests := []struct {
			name     string
			evidence []profileevidence.EvidenceValues
			want     []legalquery.EvidenceCode
		}{
			{
				name: "official_alias",
				evidence: []profileevidence.EvidenceValues{
					normalizationEvidence("alias", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias, "target-one"),
					normalizationEvidence("morph", profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext, "target-one"),
					normalizationEvidence("general", profileevidence.LayerSemanticExpansion, legalquery.EvidenceGeneralTerm, "target-one"),
				},
				want: []legalquery.EvidenceCode{legalquery.EvidenceOfficialAlias},
			},
			{
				name: "legal_concept",
				evidence: []profileevidence.EvidenceValues{
					normalizationEvidence("concept", profileevidence.LayerSemanticExpansion, legalquery.EvidenceLegalConcept, "target-one"),
					normalizationEvidence("morph", profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext, "target-one"),
					normalizationEvidence("general", profileevidence.LayerSemanticExpansion, legalquery.EvidenceGeneralTerm, "target-one"),
				},
				want: []legalquery.EvidenceCode{legalquery.EvidenceLegalConcept},
			},
			{
				name: "morphological_context",
				evidence: []profileevidence.EvidenceValues{
					normalizationEvidence("morph", profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext, "target-one"),
					normalizationEvidence("general", profileevidence.LayerSemanticExpansion, legalquery.EvidenceGeneralTerm, "target-one"),
				},
				want: []legalquery.EvidenceCode{legalquery.EvidenceMorphologicalContext},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				mapping := mustNormalizationMapping(t, test.evidence)
				values, err := mapping.NormalizedStepEvidence(
					"draft-one",
					"step-one",
				)
				if err != nil {
					t.Fatalf("正規化済み根拠を取得できません: %v", err)
				}
				if got := sortedEvidenceCodes(values); !slices.Equal(got, test.want) {
					t.Fatalf("閉じた優越表の結果 = %v、期待値は %v", got, test.want)
				}
			})
		}
	})

	t.Run("同じfactの複数groupを拒否する", func(t *testing.T) {
		span := normalizationSpan(t, 0, 4)
		_, err := profileevidence.NewMapping(profileevidence.MappingValues{
			ProfileID: "core",
			Facts: []profileevidence.FactValues{{
				FactID: "alias",
				Span:   &span,
			}},
			Drafts: []profileevidence.DraftValues{{
				DraftID: "draft-one",
				Steps: []profileevidence.StepValues{{
					StepID:               "step-one",
					SourceOrdinal:        1,
					TopicOrdinal:         1,
					StepMeaningSignature: "law-search",
					Evidence: []profileevidence.EvidenceValues{
						normalizationEvidence("alias", profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias, "target-one"),
						normalizationEvidence("alias", profileevidence.LayerTargetAnchor, legalquery.EvidenceUniqueTypoCorrection, "target-two"),
					},
				}},
			}},
		})
		if err == nil {
			t.Fatal("同じfactの複数 normalizationGroup を受理しました")
		}
	})

	t.Run("一件だけのgroupを拒否する", func(t *testing.T) {
		span := normalizationSpan(t, 0, 4)
		_, err := profileevidence.NewMapping(profileevidence.MappingValues{
			ProfileID: "core",
			Facts: []profileevidence.FactValues{{
				FactID: "alias",
				Span:   &span,
			}},
			Drafts: []profileevidence.DraftValues{{
				DraftID: "draft-one",
				Steps: []profileevidence.StepValues{{
					StepID:               "step-one",
					SourceOrdinal:        1,
					TopicOrdinal:         1,
					StepMeaningSignature: "law-search",
					Evidence: []profileevidence.EvidenceValues{
						normalizationEvidence(
							"alias",
							profileevidence.LayerTargetAnchor,
							legalquery.EvidenceOfficialAlias,
							"target-one",
						),
					},
				}},
			}},
		})
		if err == nil {
			t.Fatal("一件だけの normalizationGroup を受理しました")
		}
	})
}

func mustNormalizationMapping(
	t *testing.T,
	evidence []profileevidence.EvidenceValues,
) profileevidence.Mapping {
	t.Helper()
	factIDs := make(map[string]struct{})
	facts := make([]profileevidence.FactValues, 0, len(evidence))
	for index, value := range evidence {
		if _, exists := factIDs[value.FactID]; exists {
			continue
		}
		factIDs[value.FactID] = struct{}{}
		span := normalizationSpan(t, index*4, index*4+2)
		facts = append(facts, profileevidence.FactValues{
			FactID: value.FactID,
			Span:   &span,
		})
	}
	mapping, err := profileevidence.NewMapping(profileevidence.MappingValues{
		ProfileID: "core",
		Facts:     facts,
		Drafts: []profileevidence.DraftValues{{
			DraftID: "draft-one",
			Steps: []profileevidence.StepValues{{
				StepID:               "step-one",
				SourceOrdinal:        1,
				TopicOrdinal:         1,
				StepMeaningSignature: "law-search",
				Evidence:             evidence,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("正規化 fixture を構築できません: %v", err)
	}
	return mapping
}

func normalizationEvidence(
	factID string,
	layer profileevidence.Layer,
	code legalquery.EvidenceCode,
	group string,
) profileevidence.EvidenceValues {
	return profileevidence.EvidenceValues{
		FactID:              factID,
		Layer:               layer,
		Code:                code,
		IndependentPositive: true,
		ClusterSpan:         true,
		NormalizationGroup:  group,
	}
}

func normalizationSpan(t *testing.T, start int, end int) legalquery.QuerySpan {
	t.Helper()
	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: start,
		EndByte:   end,
	})
	if err != nil {
		t.Fatalf("正規化 fixture span を構築できません: %v", err)
	}
	return span
}

func sortedEvidenceCodes(values []profileevidence.Evidence) []legalquery.EvidenceCode {
	result := make([]legalquery.EvidenceCode, 0, len(values))
	for _, value := range values {
		result = append(result, value.Code())
	}
	slices.Sort(result)
	return result
}
