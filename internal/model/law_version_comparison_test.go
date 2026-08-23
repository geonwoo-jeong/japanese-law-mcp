package model_test

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawVersionComparisonKeepsCountsSourcesAndJSONShape(t *testing.T) {
	t.Parallel()

	beforeArticle := newVersionArticle(t, versionArticleValues{
		revisionID:   "revision-before",
		article:      "1",
		chapter:      "1",
		text:         "第一条 旧本文",
		articleTitle: "第一条",
	})
	afterArticle := newVersionArticle(t, versionArticleValues{
		revisionID:     "revision-after",
		article:        "1",
		chapter:        "2",
		text:           "第一条 新本文",
		articleTitle:   "第一条",
		articleCaption: "（目的）",
	})
	reasons := []model.LawVersionChangeReason{
		model.LawVersionChangeReasonLocation,
		model.LawVersionChangeReasonText,
	}
	modified, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind:    model.LawVersionChangeKindModified,
		ChangeReasons: reasons,
		Before:        &beforeArticle,
		After:         &afterArticle,
	})
	if err != nil {
		t.Fatalf("変更項目を構築できません: %v", err)
	}
	emptyTextArticle := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "2",
		text:       "",
		order:      2,
	})
	added, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind: model.LawVersionChangeKindAdded,
		After:      &emptyTextArticle,
	})
	if err != nil {
		t.Fatalf("空文字を持つ追加項目を構築できません: %v", err)
	}

	items := []model.LawVersionChange{modified, added}
	comparison, err := model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             newVersionSnapshot(t, "revision-before", "2024-01-01"),
		After:              newVersionSnapshot(t, "revision-after", "2025-01-01"),
		BeforeArticleCount: 2,
		AfterArticleCount:  3,
		AddedCount:         1,
		RemovedCount:       0,
		ModifiedCount:      1,
		UnchangedCount:     1,
		TotalCount:         2,
		Items:              items,
	})
	if err != nil {
		t.Fatalf("比較結果を構築できません: %v", err)
	}

	reasons[0] = model.LawVersionChangeReasonStructure
	items[0] = model.LawVersionChange{}
	gotItems := comparison.Items()
	if gotItems[0].ChangeReasons()[0] != model.LawVersionChangeReasonLocation {
		t.Fatal("constructor 入力の変更が比較結果へ反映されました")
	}
	gotItems[0] = model.LawVersionChange{}
	if comparison.Items()[0].ChangeKind() != model.LawVersionChangeKindModified {
		t.Fatal("getter が内部 items を公開しました")
	}

	payload, err := json.Marshal(comparison)
	if err != nil {
		t.Fatalf("JSON へ変換できません: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("JSON を確認できません: %v", err)
	}
	if decoded["scope"] != "main_and_original_supplementary_articles" {
		t.Fatalf("scope = %#v", decoded["scope"])
	}
	if decoded["beforeArticleCount"] != float64(2) || decoded["unchangedCount"] != float64(1) {
		t.Fatalf("件数 = %#v", decoded)
	}
	decodedItems := decoded["items"].([]any)
	addedJSON := decodedItems[1].(map[string]any)
	afterJSON := addedJSON["after"].(map[string]any)
	if text, exists := afterJSON["text"]; !exists || text != "" {
		t.Fatalf("空文字の text が必須項目として保持されていません: %#v", afterJSON)
	}
	if _, exists := addedJSON["changeReasons"]; exists {
		t.Fatalf("added に changeReasons が出力されました: %#v", addedJSON)
	}
}

func TestLawVersionChangeRejectsInconsistentIdentityAndReasons(t *testing.T) {
	t.Parallel()

	base := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-before",
		article:    "1",
		chapter:    "1",
		text:       "旧本文",
	})
	changedText := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "1",
		chapter:    "1",
		text:       "新本文",
	})
	moved := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "1",
		chapter:    "2",
		text:       "旧本文",
	})
	differentIdentity := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "2",
		chapter:    "1",
		text:       "旧本文",
	})
	structureAndTextChanged := newVersionArticle(t, versionArticleValues{
		revisionID:  "revision-after",
		article:     "1",
		chapter:     "1",
		text:        "新本文",
		fingerprint: "different-structure",
	})

	tests := map[string]model.LawVersionChangeValues{
		"added without after": {
			ChangeKind: model.LawVersionChangeKindAdded,
		},
		"modified with another identity": {
			ChangeKind:    model.LawVersionChangeKindModified,
			ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonStructure},
			Before:        &base,
			After:         &differentIdentity,
		},
		"modified without text reason": {
			ChangeKind:    model.LawVersionChangeKindModified,
			ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonStructure},
			Before:        &base,
			After:         &changedText,
		},
		"modified without location reason": {
			ChangeKind:    model.LawVersionChangeKindModified,
			ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonStructure},
			Before:        &base,
			After:         &moved,
		},
		"modified without structure reason": {
			ChangeKind:    model.LawVersionChangeKindModified,
			ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonText},
			Before:        &base,
			After:         &structureAndTextChanged,
		},
		"reasons in unstable order": {
			ChangeKind: model.LawVersionChangeKindModified,
			ChangeReasons: []model.LawVersionChangeReason{
				model.LawVersionChangeReasonText,
				model.LawVersionChangeReasonLocation,
			},
			Before: &base,
			After: versionArticlePointer(newVersionArticle(t, versionArticleValues{
				revisionID: "revision-after",
				article:    "1",
				chapter:    "2",
				text:       "新本文",
			})),
		},
	}
	for name, values := range tests {
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := model.NewLawVersionChange(values); err == nil {
				t.Fatal("不整合な変更項目を受理しました")
			}
		})
	}
}

func TestLawVersionChangeAcceptsStructureOnlyDifference(t *testing.T) {
	t.Parallel()

	before := newVersionArticle(t, versionArticleValues{
		revisionID:  "revision-before",
		article:     "1",
		text:        "同一本文",
		fingerprint: "structure-before",
	})
	after := newVersionArticle(t, versionArticleValues{
		revisionID:  "revision-after",
		article:     "1",
		text:        "同一本文",
		fingerprint: "structure-after",
	})

	change, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind:    model.LawVersionChangeKindModified,
		ChangeReasons: []model.LawVersionChangeReason{model.LawVersionChangeReasonStructure},
		Before:        &before,
		After:         &after,
	})
	if err != nil {
		t.Fatalf("構造だけが変わる modified を構築できません: %v", err)
	}
	if reasons := change.ChangeReasons(); len(reasons) != 1 || reasons[0] != model.LawVersionChangeReasonStructure {
		t.Fatalf("changeReasons = %#v", reasons)
	}
}

func TestLawVersionComparisonRejectsItemsOutsideDocumentOrder(t *testing.T) {
	t.Parallel()

	late := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "2",
		text:       "第二条",
		order:      2,
	})
	early := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-after",
		article:    "1",
		text:       "第一条",
		order:      1,
	})
	lateChange, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind: model.LawVersionChangeKindAdded,
		After:      &late,
	})
	if err != nil {
		t.Fatalf("後方の追加項目を構築できません: %v", err)
	}
	earlyChange, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind: model.LawVersionChangeKindAdded,
		After:      &early,
	})
	if err != nil {
		t.Fatalf("前方の追加項目を構築できません: %v", err)
	}

	_, err = model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             newVersionSnapshot(t, "revision-before", "2024-01-01"),
		After:              newVersionSnapshot(t, "revision-after", "2025-01-01"),
		BeforeArticleCount: 0,
		AfterArticleCount:  2,
		AddedCount:         2,
		TotalCount:         2,
		Items:              []model.LawVersionChange{lateChange, earlyChange},
	})
	if err == nil {
		t.Fatal("比較後版の文書順ではない items を受理しました")
	}
}

func TestLawVersionComparisonRejectsRemovedItemsOutsideBeforeDocumentOrder(t *testing.T) {
	t.Parallel()

	late := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-before",
		article:    "2",
		text:       "第二条",
		order:      2,
	})
	early := newVersionArticle(t, versionArticleValues{
		revisionID: "revision-before",
		article:    "1",
		text:       "第一条",
		order:      1,
	})
	lateChange, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind: model.LawVersionChangeKindRemoved,
		Before:     &late,
	})
	if err != nil {
		t.Fatalf("後方の削除項目を構築できません: %v", err)
	}
	earlyChange, err := model.NewLawVersionChange(model.LawVersionChangeValues{
		ChangeKind: model.LawVersionChangeKindRemoved,
		Before:     &early,
	})
	if err != nil {
		t.Fatalf("前方の削除項目を構築できません: %v", err)
	}

	_, err = model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             newVersionSnapshot(t, "revision-before", "2024-01-01"),
		After:              newVersionSnapshot(t, "revision-after", "2025-01-01"),
		BeforeArticleCount: 2,
		AfterArticleCount:  0,
		RemovedCount:       2,
		TotalCount:         2,
		Items:              []model.LawVersionChange{lateChange, earlyChange},
	})
	if err == nil {
		t.Fatal("比較前版の文書順ではない removed items を受理しました")
	}
}

func TestLawVersionComparisonRejectsCountAndCitationMismatch(t *testing.T) {
	t.Parallel()

	before := newVersionSnapshot(t, "revision-before", "2024-01-01")
	after := newVersionSnapshot(t, "revision-after", "2025-01-01")
	values := model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             before,
		After:              after,
		BeforeArticleCount: 1,
		AfterArticleCount:  1,
		UnchangedCount:     1,
		TotalCount:         1,
	}
	if _, err := model.NewLawVersionComparison(values); err == nil {
		t.Fatal("items と一致しない totalCount を受理しました")
	}

	wrongAfter := newVersionSnapshotForLaw(t, "law-2", "revision-after", "2025-01-01")
	values.TotalCount = 0
	values.After = wrongAfter
	if _, err := model.NewLawVersionComparison(values); err == nil {
		t.Fatal("別法令の snapshot を受理しました")
	}
}

type versionArticleValues struct {
	revisionID     string
	article        string
	chapter        string
	text           string
	articleTitle   string
	articleCaption string
	order          int
	fingerprint    string
}

func newVersionArticle(t *testing.T, values versionArticleValues) model.LawVersionArticle {
	t.Helper()
	if values.order == 0 {
		values.order = 1
	}
	if values.fingerprint == "" {
		values.fingerprint = "structure"
	}
	location, err := model.NewLawVersionArticleLocation(model.LawVersionArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: values.article,
		ChapterNumber: values.chapter,
	})
	if err != nil {
		t.Fatalf("位置を構築できません: %v", err)
	}
	article, err := model.NewLawVersionArticle(model.LawVersionArticleValues{
		Location:             location,
		ArticleTitle:         values.articleTitle,
		ArticleCaption:       values.articleCaption,
		Text:                 values.text,
		Citation:             newVersionCitation(t, "law-1", values.revisionID, values.article),
		DocumentOrder:        values.order,
		StructureFingerprint: values.fingerprint,
	})
	if err != nil {
		t.Fatalf("条を構築できません: %v", err)
	}
	return article
}

func versionArticlePointer(value model.LawVersionArticle) *model.LawVersionArticle {
	return &value
}

func newVersionSnapshot(t *testing.T, revisionID, asOf string) model.LawVersionSnapshot {
	t.Helper()
	return newVersionSnapshotForLaw(t, "law-1", revisionID, asOf)
}

func newVersionSnapshotForLaw(
	t *testing.T,
	lawID string,
	revisionID string,
	asOf string,
) model.LawVersionSnapshot {
	t.Helper()
	source := newVersionLegalSource(t)
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "行政手続法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("法令概要を構築できません: %v", err)
	}
	date, err := model.NewDate(asOf)
	if err != nil {
		t.Fatalf("日付を構築できません: %v", err)
	}
	snapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      law,
		AsOf:     &date,
		Citation: newVersionCitation(t, lawID, revisionID, ""),
	})
	if err != nil {
		t.Fatalf("snapshot を構築できません: %v", err)
	}
	return snapshot
}

func newVersionCitation(
	t *testing.T,
	lawID string,
	revisionID string,
	article string,
) model.Citation {
	t.Helper()
	location := ""
	if article != "" {
		location = "main:article=" + article
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     newVersionLegalSource(t),
		LawID:      lawID,
		RevisionID: revisionID,
		Location:   location,
		URL:        "https://laws.e-gov.go.jp/law/" + revisionID,
	})
	if err != nil {
		t.Fatalf("Citation を構築できません: %v", err)
	}
	return citation
}

func newVersionLegalSource(t *testing.T) model.LegalSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	})
	if err != nil {
		t.Fatalf("情報源を構築できません: %v", err)
	}
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("法令情報源を構築できません: %v", err)
	}
	return legalSource
}
