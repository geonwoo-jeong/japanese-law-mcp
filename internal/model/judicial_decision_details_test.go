package model_test

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialDecisionDetailsCopiesValuesAndMarshalsJSON(t *testing.T) {
	t.Parallel()

	reporterCitation := "民集第79巻2号123頁"
	lowerCourtName := "東京高等裁判所"
	lowerCourtCaseNumber := "令和5年（ネ）第100号"
	lowerCourtDecisionDate := newDate(t, "2024-04-01")
	holdingText := "第一の判示事項\n第二の判示事項"
	summaryText := "裁判要旨の原文"
	referencedProvisionsText := "民法709条\n民事訴訟法312条"
	summary, err := model.NewJudicialDecisionSummary(
		validJudicialDecisionSummaryValues(t),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-020: JudicialDecisionSummary を作成できない: %v", err)
	}

	got, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:                  summary,
		ReporterCitation:         &reporterCitation,
		LowerCourtName:           &lowerCourtName,
		LowerCourtCaseNumber:     &lowerCourtCaseNumber,
		LowerCourtDecisionDate:   &lowerCourtDecisionDate,
		HoldingText:              &holdingText,
		SummaryText:              &summaryText,
		ReferencedProvisionsText: &referencedProvisionsText,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-021: NewJudicialDecisionDetails() のエラー = %v", err)
	}

	reporterCitation = "変更後"
	lowerCourtName = "変更後"
	lowerCourtCaseNumber = "変更後"
	lowerCourtDecisionDate = newDate(t, "2030-01-01")
	holdingText = "変更後"
	summaryText = "変更後"
	referencedProvisionsText = "変更後"

	if got.Summary().DecisionID() != "95570" {
		t.Fatalf("SOT-MODEL-021: Summary() = %#v", got.Summary())
	}
	assertJudicialDetailsOptionalString(
		t,
		"ReporterCitation",
		got.ReporterCitation,
		"民集第79巻2号123頁",
	)
	assertJudicialDetailsOptionalString(
		t,
		"LowerCourtName",
		got.LowerCourtName,
		"東京高等裁判所",
	)
	assertJudicialDetailsOptionalString(
		t,
		"LowerCourtCaseNumber",
		got.LowerCourtCaseNumber,
		"令和5年（ネ）第100号",
	)
	assertJudicialDetailsOptionalString(
		t,
		"HoldingText",
		got.HoldingText,
		"第一の判示事項\n第二の判示事項",
	)
	assertJudicialDetailsOptionalString(
		t,
		"SummaryText",
		got.SummaryText,
		"裁判要旨の原文",
	)
	assertJudicialDetailsOptionalString(
		t,
		"ReferencedProvisionsText",
		got.ReferencedProvisionsText,
		"民法709条\n民事訴訟法312条",
	)
	if date, exists := got.LowerCourtDecisionDate(); !exists ||
		date.String() != "2024-04-01" {
		t.Fatalf(
			"SOT-MODEL-021: LowerCourtDecisionDate() = %q, %t",
			date.String(),
			exists,
		)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-021: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/021: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/021: JSON を再解析できない: %v", err)
	}
	if object["reporterCitation"] != "民集第79巻2号123頁" ||
		object["lowerCourtName"] != "東京高等裁判所" ||
		object["lowerCourtCaseNumber"] != "令和5年（ネ）第100号" ||
		object["lowerCourtDecisionDate"] != "2024-04-01" ||
		object["holdingText"] != "第一の判示事項\n第二の判示事項" ||
		object["summaryText"] != "裁判要旨の原文" ||
		object["referencedProvisionsText"] != "民法709条\n民事訴訟法312条" {
		t.Fatalf("SOT-MODEL-009/021: JSON = %#v", object)
	}
	summaryJSON, exists := object["summary"].(map[string]any)
	if !exists || summaryJSON["decisionId"] != "95570" {
		t.Fatalf("SOT-MODEL-009/021: summary JSON = %#v", object["summary"])
	}
}

func TestJudicialDecisionDetailsOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	summary, err := model.NewJudicialDecisionSummary(
		validJudicialDecisionSummaryValues(t),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-020: JudicialDecisionSummary を作成できない: %v", err)
	}
	got, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary: summary,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-021: NewJudicialDecisionDetails() のエラー = %v", err)
	}

	for name, getter := range map[string]func() (string, bool){
		"reporterCitation":         got.ReporterCitation,
		"lowerCourtName":           got.LowerCourtName,
		"lowerCourtCaseNumber":     got.LowerCourtCaseNumber,
		"holdingText":              got.HoldingText,
		"summaryText":              got.SummaryText,
		"referencedProvisionsText": got.ReferencedProvisionsText,
	} {
		value, exists := getter()
		if exists || value != "" {
			t.Fatalf("SOT-MODEL-021: %s = %q, %t", name, value, exists)
		}
	}
	if date, exists := got.LowerCourtDecisionDate(); exists || !date.IsZero() {
		t.Fatalf(
			"SOT-MODEL-021: LowerCourtDecisionDate() = %q, %t",
			date.String(),
			exists,
		)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/021: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/021: JSON を再解析できない: %v", err)
	}
	for _, key := range []string{
		"reporterCitation",
		"lowerCourtName",
		"lowerCourtCaseNumber",
		"lowerCourtDecisionDate",
		"holdingText",
		"summaryText",
		"referencedProvisionsText",
	} {
		if _, exists := object[key]; exists {
			t.Fatalf("SOT-MODEL-009/021: 欠落した %s が省略されていない: %s", key, encoded)
		}
	}
}

func TestJudicialDecisionDetailsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	empty := ""
	zeroDate := model.Date{}
	summary, err := model.NewJudicialDecisionSummary(
		validJudicialDecisionSummaryValues(t),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-020: JudicialDecisionSummary を作成できない: %v", err)
	}
	valid := model.JudicialDecisionDetailsValues{Summary: summary}
	tests := map[string]model.JudicialDecisionDetailsValues{
		"summary のゼロ値": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.Summary = model.JudicialDecisionSummary{}
			},
		),
		"lowerCourtDecisionDate のゼロ値": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.LowerCourtDecisionDate = &zeroDate
			},
		),
		"空の reporterCitation": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.ReporterCitation = &empty
			},
		),
		"空の lowerCourtName": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.LowerCourtName = &empty
			},
		),
		"空の lowerCourtCaseNumber": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.LowerCourtCaseNumber = &empty
			},
		),
		"空の holdingText": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.HoldingText = &empty
			},
		),
		"空の summaryText": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.SummaryText = &empty
			},
		),
		"空の referencedProvisionsText": withJudicialDecisionDetailsChange(
			valid,
			func(values *model.JudicialDecisionDetailsValues) {
				values.ReferencedProvisionsText = &empty
			},
		),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewJudicialDecisionDetails(values); err == nil {
				t.Fatalf(
					"SOT-MODEL-021: NewJudicialDecisionDetails(%#v) が成功した",
					values,
				)
			}
		})
	}
}

func TestJudicialDecisionDetailsRejectsZeroValueAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.JudicialDecisionDetails{}); err == nil {
		t.Fatal("SOT-MODEL-009/021: JudicialDecisionDetails のゼロ値を JSON に変換できた")
	}

	var details model.JudicialDecisionDetails
	if err := json.Unmarshal([]byte(`{"summary":{"decisionId":"95570"}}`), &details); err == nil {
		t.Fatal("SOT-MODEL-021: JudicialDecisionDetails を JSON から直接復元できた")
	}
}

func assertJudicialDetailsOptionalString(
	t *testing.T,
	name string,
	getter func() (string, bool),
	want string,
) {
	t.Helper()

	value, exists := getter()
	if !exists || value != want {
		t.Fatalf("SOT-MODEL-021: %s() = %q, %t", name, value, exists)
	}
}

func withJudicialDecisionDetailsChange(
	values model.JudicialDecisionDetailsValues,
	change func(*model.JudicialDecisionDetailsValues),
) model.JudicialDecisionDetailsValues {
	change(&values)
	return values
}
