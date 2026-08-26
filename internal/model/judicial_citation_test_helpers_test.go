package model_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func judicialCitationMethods(values []string) []model.JudicialCitationMethod {
	methods := make([]model.JudicialCitationMethod, len(values))
	for index, value := range values {
		methods[index] = model.JudicialCitationMethod(value)
	}
	return methods
}

func mustIncomingCitationCoverage(t *testing.T) model.JudicialCitationCoverage {
	t.Helper()
	limit, attempted, completed := 10, 1, 1
	incoming, err := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:            model.JudicialCitationDirectionStatusComplete,
			Methods:           []model.JudicialCitationMethod{model.JudicialCitationMethodOfficialCaseSearch},
			Limit:             &limit,
			AttemptedSearches: &attempted,
			CompletedSearches: &completed,
		},
	)
	if err != nil {
		t.Fatalf("incoming coverage を作成できません: %v", err)
	}
	coverage, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: model.JudicialCitationRequestedDirectionIncoming,
		Outgoing:           mustDirectionCoverage(t, "not_requested", []string{}),
		Incoming:           incoming,
	})
	if err != nil {
		t.Fatalf("coverage を作成できません: %v", err)
	}
	return coverage
}
