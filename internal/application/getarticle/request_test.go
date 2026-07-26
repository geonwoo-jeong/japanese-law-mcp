package getarticle

import (
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestNewRequestNormalizesPublicInput(t *testing.T) {
	t.Parallel()

	paragraph := 2
	asOf, err := model.NewDate("2026-07-26")
	if err != nil {
		t.Fatalf("日付を作成できません: %v", err)
	}
	provision := model.LawArticleProvisionSupplementary
	request, err := NewRequest(RequestValues{
		LawID:     " law-1 ",
		Provision: &provision,
		Article:   "38_3",
		Paragraph: &paragraph,
		AsOf:      &asOf,
	})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	if request.LawID() != "law-1" {
		t.Fatalf("lawId = %q", request.LawID())
	}
	gotAsOf, exists := request.AsOf()
	if !exists || gotAsOf.String() != "2026-07-26" {
		t.Fatalf("asOf = %q, %t", gotAsOf.String(), exists)
	}
	location := request.Location()
	if location.Provision() != model.LawArticleProvisionSupplementary ||
		location.ArticleNumber() != "38_3" {
		t.Fatalf("location = %#v", location)
	}
	gotParagraph, exists := location.ParagraphNumber()
	if !exists || gotParagraph != 2 {
		t.Fatalf("paragraph = %d, %t", gotParagraph, exists)
	}
}

func TestNewRequestDefaultsProvisionToMain(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{LawID: "law-1", Article: "1"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	if request.Location().Provision() != model.LawArticleProvisionMain {
		t.Fatalf("provision = %q", request.Location().Provision())
	}
	if _, exists := request.Location().ParagraphNumber(); exists {
		t.Fatal("省略した paragraph が設定されています")
	}
}

func TestNewRequestRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	emptyProvision := model.LawArticleProvision("")
	invalidProvision := model.LawArticleProvision("amending")
	zero := 0
	oldDate, err := model.NewDate("2017-03-31")
	if err != nil {
		t.Fatalf("日付を作成できません: %v", err)
	}
	tests := map[string]RequestValues{
		"空の lawId":             {LawID: " ", Article: "1"},
		"空の provision":         {LawID: "law-1", Provision: &emptyProvision, Article: "1"},
		"未定義の provision":       {LawID: "law-1", Provision: &invalidProvision, Article: "1"},
		"非正規形の article":        {LawID: "law-1", Article: "01"},
		"0 の paragraph":        {LawID: "law-1", Article: "1", Paragraph: &zero},
		"収録開始日前の asOf":         {LawID: "law-1", Article: "1", AsOf: &oldDate},
		"article の ASCII 制御文字": {LawID: "law-1", Article: "1\n"},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewRequest(values); err == nil {
				t.Fatalf("NewRequest(%#v) が成功しました", values)
			}
		})
	}
}

func TestRequestRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var request Request
	if err := json.Unmarshal(
		[]byte(`{"lawId":"law-1","article":"1"}`),
		&request,
	); err == nil {
		t.Fatal("Request を JSON から直接復元できました")
	}
}
