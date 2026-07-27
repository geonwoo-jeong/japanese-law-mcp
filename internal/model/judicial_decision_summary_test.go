package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialDecisionSummaryCopiesValuesAndMarshalsJSON(t *testing.T) {
	t.Parallel()

	caseName := "損害賠償請求事件"
	branchName := "東京支部"
	divisionName := "第一小法廷"
	decisionType := "判決"
	outcome := "破棄差戻"
	documents := []model.JudicialDocumentLink{
		newJudicialDocumentLink(
			t,
			model.JudicialDocumentKindFullText,
			"全文",
			"https://www.courts.go.jp/app/files/hanrei_jp/570/095570_hanrei.pdf",
		),
		newJudicialDocumentLink(
			t,
			model.JudicialDocumentKindAttachment,
			"別紙",
			"https://www.courts.go.jp/app/files/hanrei_jp/570/095570_option1.pdf",
		),
	}
	source := newJudicialInformationSource(t)

	got, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "95570",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6年（受）第1号",
		CaseName:            &caseName,
		DecisionDate:        newDate(t, "2025-03-03"),
		CourtName:           "最高裁判所",
		BranchName:          &branchName,
		DivisionName:        &divisionName,
		DecisionType:        &decisionType,
		Outcome:             &outcome,
		DetailURL:           "https://www.courts.go.jp/hanrei/95570/detail2/index.html",
		Documents:           documents,
		Source:              source,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-020: NewJudicialDecisionSummary() のエラー = %v", err)
	}

	caseName = "変更後"
	branchName = "変更後"
	divisionName = "変更後"
	decisionType = "変更後"
	outcome = "変更後"
	documents[0] = newJudicialDocumentLink(
		t,
		model.JudicialDocumentKindSummary,
		"変更後",
		"https://www.courts.go.jp/app/files/changed.pdf",
	)

	if got.DecisionID() != "95570" ||
		got.PublicationCategory() != model.JudicialPublicationCategorySupremeCourt ||
		got.SourceCategoryLabel() != "最高裁判例" ||
		got.CaseNumber() != "令和6年（受）第1号" ||
		got.DecisionDate().String() != "2025-03-03" ||
		got.CourtName() != "最高裁判所" ||
		got.DetailURL() != "https://www.courts.go.jp/hanrei/95570/detail2/index.html" ||
		got.Source() != source {
		t.Fatalf("SOT-MODEL-020: JudicialDecisionSummary = %#v", got)
	}
	assertJudicialOptionalString(t, "CaseName", got.CaseName, "損害賠償請求事件")
	assertJudicialOptionalString(t, "BranchName", got.BranchName, "東京支部")
	assertJudicialOptionalString(t, "DivisionName", got.DivisionName, "第一小法廷")
	assertJudicialOptionalString(t, "DecisionType", got.DecisionType, "判決")
	assertJudicialOptionalString(t, "Outcome", got.Outcome, "破棄差戻")

	firstRead := got.Documents()
	if len(firstRead) != 2 || firstRead[0].Label() != "全文" {
		t.Fatalf("SOT-MODEL-020: Documents() = %#v", firstRead)
	}
	firstRead[0] = newJudicialDocumentLink(
		t,
		model.JudicialDocumentKindSummary,
		"外部変更",
		"https://www.courts.go.jp/app/files/external-change.pdf",
	)
	if got.Documents()[0].Label() != "全文" {
		t.Fatal("SOT-MODEL-020: Documents() が内部配列を公開した")
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-020: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/020: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/020: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"decisionId":          "95570",
		"publicationCategory": "supreme_court",
		"sourceCategoryLabel": "最高裁判例",
		"caseNumber":          "令和6年（受）第1号",
		"caseName":            "損害賠償請求事件",
		"decisionDate":        "2025-03-03",
		"courtName":           "最高裁判所",
		"branchName":          "東京支部",
		"divisionName":        "第一小法廷",
		"decisionType":        "判決",
		"outcome":             "破棄差戻",
		"detailUrl":           "https://www.courts.go.jp/hanrei/95570/detail2/index.html",
		"documents": []any{
			map[string]any{
				"kind":      "full_text",
				"label":     "全文",
				"mediaType": "application/pdf",
				"url": "https://www.courts.go.jp/app/files/hanrei_jp/570/" +
					"095570_hanrei.pdf",
			},
			map[string]any{
				"kind":      "attachment",
				"label":     "別紙",
				"mediaType": "application/pdf",
				"url": "https://www.courts.go.jp/app/files/hanrei_jp/570/" +
					"095570_option1.pdf",
			},
		},
		"source": map[string]any{
			"id":         "courts-hanrei",
			"name":       "裁判例検索",
			"publisher":  "最高裁判所",
			"authority":  "official",
			"serviceUrl": "https://www.courts.go.jp/hanrei/search1/index.html",
		},
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/020: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestJudicialDecisionSummaryOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	got, err := model.NewJudicialDecisionSummary(
		validJudicialDecisionSummaryValues(t),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-020: NewJudicialDecisionSummary() のエラー = %v", err)
	}

	for name, getter := range map[string]func() (string, bool){
		"caseName":     got.CaseName,
		"branchName":   got.BranchName,
		"divisionName": got.DivisionName,
		"decisionType": got.DecisionType,
		"outcome":      got.Outcome,
	} {
		value, exists := getter()
		if exists || value != "" {
			t.Fatalf("SOT-MODEL-020: %s = %q, %t", name, value, exists)
		}
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/020: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/020: JSON を再解析できない: %v", err)
	}
	for _, key := range []string{
		"caseName",
		"branchName",
		"divisionName",
		"decisionType",
		"outcome",
	} {
		if _, exists := object[key]; exists {
			t.Fatalf("SOT-MODEL-009/020: 欠落した %s が省略されていない: %s", key, encoded)
		}
	}
	documents, exists := object["documents"].([]any)
	if !exists || len(documents) != 0 {
		t.Fatalf("SOT-MODEL-020: documents は空配列でなければならない: %#v", object)
	}
}

func TestJudicialDecisionSummaryAcceptsPublicationCategories(t *testing.T) {
	t.Parallel()

	for _, category := range []model.JudicialPublicationCategory{
		model.JudicialPublicationCategorySupremeCourt,
		model.JudicialPublicationCategoryHighCourt,
		model.JudicialPublicationCategoryLowerCourt,
		model.JudicialPublicationCategoryAdministrative,
		model.JudicialPublicationCategoryLabor,
		model.JudicialPublicationCategoryIntellectualProperty,
	} {
		category := category
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()

			values := validJudicialDecisionSummaryValues(t)
			values.PublicationCategory = category
			if _, err := model.NewJudicialDecisionSummary(values); err != nil {
				t.Fatalf("SOT-MODEL-020: category %q を拒否した: %v", category, err)
			}
		})
	}
}

func TestJudicialDecisionSummaryRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	empty := ""
	zeroDate := model.Date{}
	valid := validJudicialDecisionSummaryValues(t)
	tests := map[string]model.JudicialDecisionSummaryValues{
		"decisionId の欠落": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DecisionID = ""
			},
		),
		"未知の publicationCategory": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.PublicationCategory = model.JudicialPublicationCategory("district")
			},
		),
		"sourceCategoryLabel の欠落": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.SourceCategoryLabel = ""
			},
		),
		"caseNumber の欠落": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.CaseNumber = ""
			},
		),
		"decisionDate のゼロ値": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DecisionDate = zeroDate
			},
		),
		"courtName の欠落": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.CourtName = ""
			},
		),
		"HTTP detailUrl": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DetailURL = "http://www.courts.go.jp/hanrei/95570/detail2/index.html"
			},
		),
		"別ホストの detailUrl": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DetailURL = "https://example.com/hanrei/95570/detail2/index.html"
			},
		),
		"documents の nil": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Documents = nil
			},
		),
		"documents のゼロ値": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Documents = []model.JudicialDocumentLink{{}}
			},
		),
		"source のゼロ値": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Source = model.InformationSource{}
			},
		),
		"補助情報源": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Source = newJudicialInformationSourceWithAuthority(
					t,
					model.AuthoritySupplementary,
				)
			},
		),
		"別サービスの source": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Source = newOtherOfficialInformationSource(t)
			},
		),
		"空の caseName": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.CaseName = &empty
			},
		),
		"空の branchName": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.BranchName = &empty
			},
		),
		"空の divisionName": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DivisionName = &empty
			},
		),
		"空の decisionType": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.DecisionType = &empty
			},
		),
		"空の outcome": withJudicialDecisionSummaryChange(
			valid,
			func(values *model.JudicialDecisionSummaryValues) {
				values.Outcome = &empty
			},
		),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewJudicialDecisionSummary(values); err == nil {
				t.Fatalf(
					"SOT-MODEL-020: NewJudicialDecisionSummary(%#v) が成功した",
					values,
				)
			}
		})
	}
}

func TestJudicialDecisionSummaryRejectsZeroValueAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.JudicialDecisionSummary{}); err == nil {
		t.Fatal("SOT-MODEL-009/020: JudicialDecisionSummary のゼロ値を JSON に変換できた")
	}

	var summary model.JudicialDecisionSummary
	if err := json.Unmarshal([]byte(`{"decisionId":"95570"}`), &summary); err == nil {
		t.Fatal("SOT-MODEL-020: JudicialDecisionSummary を JSON から直接復元できた")
	}
}

func validJudicialDecisionSummaryValues(t *testing.T) model.JudicialDecisionSummaryValues {
	t.Helper()

	return model.JudicialDecisionSummaryValues{
		DecisionID:          "95570",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6年（受）第1号",
		DecisionDate:        newDate(t, "2025-03-03"),
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/95570/detail2/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              newJudicialInformationSource(t),
	}
}

func newJudicialDocumentLink(
	t *testing.T,
	kind model.JudicialDocumentKind,
	label string,
	documentURL string,
) model.JudicialDocumentLink {
	t.Helper()

	got, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      kind,
		Label:     label,
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       documentURL,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-020: JudicialDocumentLink を作成できない: %v", err)
	}
	return got
}

func newJudicialInformationSource(t *testing.T) model.InformationSource {
	t.Helper()
	return newJudicialInformationSourceWithAuthority(t, model.AuthorityOfficial)
}

func newJudicialInformationSourceWithAuthority(
	t *testing.T,
	authority model.Authority,
) model.InformationSource {
	t.Helper()

	got, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  authority,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: InformationSource を作成できない: %v", err)
	}
	return got
}

func newOtherOfficialInformationSource(t *testing.T) model.InformationSource {
	t.Helper()

	got, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "other-official-source",
		Name:       "他の公式情報源",
		Publisher:  "他の機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.com/",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: InformationSource を作成できない: %v", err)
	}
	return got
}

func assertJudicialOptionalString(
	t *testing.T,
	name string,
	getter func() (string, bool),
	want string,
) {
	t.Helper()

	value, exists := getter()
	if !exists || value != want {
		t.Fatalf("SOT-MODEL-020: %s() = %q, %t", name, value, exists)
	}
}

func withJudicialDecisionSummaryChange(
	values model.JudicialDecisionSummaryValues,
	change func(*model.JudicialDecisionSummaryValues),
) model.JudicialDecisionSummaryValues {
	change(&values)
	return values
}
