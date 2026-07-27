package legalquery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLogicalInputConstructorsCreateSevenProviderIndependentVariants(t *testing.T) {
	t.Parallel()

	asOf := mustDate(t, "2025-04-01")
	lawRef := newLegalQuerySourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"law-1",
	)
	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	location := mustLawArticleLocation(t)

	lawSearch, err := NewLawSearchIntentV1(LawSearchIntentV1Values{
		Query: "  行政手続法  ",
		AsOf:  &asOf,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: LawSearchIntentV1 を作成できません: %v", err)
	}
	if lawSearch.Query() != "行政手続法" {
		t.Fatalf("SOT-MODEL-022: Query() = %q", lawSearch.Query())
	}
	assertOptionalDate(t, lawSearch.AsOf, "2025-04-01")

	allTerms := []string{" 永住許可 "}
	contentSearch, err := NewLawContentSearchIntentV1(LawContentSearchIntentV1Values{
		AllTerms:     allTerms,
		AnyTerms:     []string{"在留資格"},
		ExcludeTerms: []string{},
		AsOf:         &asOf,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: LawContentSearchIntentV1 を作成できません: %v", err)
	}
	allTerms[0] = "変更済み"
	if got := contentSearch.AllTerms(); len(got) != 1 || got[0] != "永住許可" {
		t.Fatalf("SOT-MODEL-022: AllTerms() = %#v", got)
	}
	returnedTerms := contentSearch.AllTerms()
	returnedTerms[0] = "変更済み"
	if contentSearch.AllTerms()[0] != "永住許可" {
		t.Fatal("SOT-MODEL-022: getter から logical input を変更できました")
	}
	if got := contentSearch.ExcludeTerms(); got == nil || len(got) != 0 {
		t.Fatalf("SOT-MODEL-022: ExcludeTerms() = %#v", got)
	}

	lawReadByID, err := NewLawReadIntentV1(LawReadIntentV1Values{
		LawID:      "503AC0000000037",
		RevisionID: "503AC0000000037_20250601",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: lawId の LawReadIntentV1 を作成できません: %v", err)
	}
	assertOptionalString(t, lawReadByID.LawID, "503AC0000000037")
	assertOptionalString(t, lawReadByID.RevisionID, "503AC0000000037_20250601")
	if _, exists := lawReadByID.Ref(); exists {
		t.Fatal("SOT-MODEL-022: lawId 形が ref を持っています")
	}

	lawReadByRef, err := NewLawReadIntentV1(LawReadIntentV1Values{Ref: &lawRef})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: ref の LawReadIntentV1 を作成できません: %v", err)
	}
	if got, exists := lawReadByRef.Ref(); !exists || got != lawRef {
		t.Fatalf("SOT-MODEL-022: Ref() = %#v, %t", got, exists)
	}

	articleRead, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
		Ref:      &lawRef,
		Location: location,
		AsOf:     &asOf,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: LawArticleReadIntentV1 を作成できません: %v", err)
	}
	if got := articleRead.Location(); got != location {
		t.Fatalf("SOT-MODEL-022: Location() = %#v", got)
	}

	updateList, err := NewLawUpdateListIntentV1(
		LawUpdateListIntentV1Values{Date: mustDate(t, "2026-07-27")},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: LawUpdateListIntentV1 を作成できません: %v", err)
	}
	if updateList.Date().String() != "2026-07-27" {
		t.Fatalf("SOT-MODEL-022: Date() = %q", updateList.Date().String())
	}

	judicialSearch, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "\u3000永住許可\u3000"},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: JudicialDecisionSearchIntentV1 を作成できません: %v", err)
	}
	if judicialSearch.Query() != "永住許可" {
		t.Fatalf("SOT-MODEL-022: Query() = %q", judicialSearch.Query())
	}

	judicialRead, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: judicialRef},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: JudicialDecisionReadIntentV1 を作成できません: %v", err)
	}
	if judicialRead.Ref() != judicialRef {
		t.Fatalf("SOT-MODEL-022: Ref() = %#v", judicialRead.Ref())
	}

	kinds := []LogicalInputKind{
		lawSearch.InputKind(),
		contentSearch.InputKind(),
		lawReadByID.InputKind(),
		articleRead.InputKind(),
		updateList.InputKind(),
		judicialSearch.InputKind(),
		judicialRead.InputKind(),
	}
	wantKinds := []LogicalInputKind{
		InputKindLawSearch,
		InputKindLawContentSearch,
		InputKindLawRead,
		InputKindLawArticleRead,
		InputKindLawUpdates,
		InputKindJudicialDecisionSearch,
		InputKindJudicialDecisionRead,
	}
	for index := range wantKinds {
		if kinds[index] != wantKinds[index] {
			t.Fatalf("SOT-MODEL-022: kinds[%d] = %q", index, kinds[index])
		}
	}
}

func TestLogicalInputsRejectValuesThatCannotBeMaterialized(t *testing.T) {
	t.Parallel()

	asOf := mustDate(t, "2025-04-01")
	lawRef := newLegalQuerySourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"law-1",
	)
	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	versionedLawRef := newVersionedSourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"law-1",
		"revision-1",
	)
	versionedJudicialRef := newVersionedSourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
		"version-1",
	)
	location := mustLawArticleLocation(t)

	tests := map[string]func() error{
		"空の法令検索語": func() error {
			_, err := NewLawSearchIntentV1(LawSearchIntentV1Values{Query: "   "})
			return err
		},
		"上限超過の法令検索語": func() error {
			_, err := NewLawSearchIntentV1(
				LawSearchIntentV1Values{Query: strings.Repeat("a", 513)},
			)
			return err
		},
		"本文検索の空条件": func() error {
			_, err := NewLawContentSearchIntentV1(LawContentSearchIntentV1Values{})
			return err
		},
		"本文検索の演算子": func() error {
			_, err := NewLawContentSearchIntentV1(
				LawContentSearchIntentV1Values{AllTerms: []string{"永住*"}},
			)
			return err
		},
		"法令読取りの対象欠落": func() error {
			_, err := NewLawReadIntentV1(LawReadIntentV1Values{})
			return err
		},
		"法令読取りの ID と ref": func() error {
			_, err := NewLawReadIntentV1(LawReadIntentV1Values{
				LawID: "law-1",
				Ref:   &lawRef,
			})
			return err
		},
		"法令読取りの revision と asOf": func() error {
			_, err := NewLawReadIntentV1(LawReadIntentV1Values{
				LawID:      "law-1",
				RevisionID: "revision-1",
				AsOf:       &asOf,
			})
			return err
		},
		"法令読取りの ref と asOf": func() error {
			_, err := NewLawReadIntentV1(LawReadIntentV1Values{
				Ref:  &lawRef,
				AsOf: &asOf,
			})
			return err
		},
		"法令読取りに裁判例 ref": func() error {
			_, err := NewLawReadIntentV1(LawReadIntentV1Values{Ref: &judicialRef})
			return err
		},
		"条文読取りの対象欠落": func() error {
			_, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
				Location: location,
			})
			return err
		},
		"条文読取りの ID と ref": func() error {
			_, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
				LawID:    "law-1",
				Ref:      &lawRef,
				Location: location,
			})
			return err
		},
		"条文読取りの版付き ref と asOf": func() error {
			_, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
				Ref:      &versionedLawRef,
				Location: location,
				AsOf:     &asOf,
			})
			return err
		},
		"更新一覧のゼロ日付": func() error {
			_, err := NewLawUpdateListIntentV1(LawUpdateListIntentV1Values{})
			return err
		},
		"空の裁判例検索語": func() error {
			_, err := NewJudicialDecisionSearchIntentV1(
				JudicialDecisionSearchIntentV1Values{Query: "\u3000"},
			)
			return err
		},
		"裁判例読取りに法令 ref": func() error {
			_, err := NewJudicialDecisionReadIntentV1(
				JudicialDecisionReadIntentV1Values{Ref: lawRef},
			)
			return err
		},
		"裁判例読取りの版付き ref": func() error {
			_, err := NewJudicialDecisionReadIntentV1(
				JudicialDecisionReadIntentV1Values{Ref: versionedJudicialRef},
			)
			return err
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := test(); err == nil {
				t.Fatalf("SOT-MODEL-022: 不正な logical input を受理しました")
			}
		})
	}
}

func TestLogicalInputsRejectDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	targets := []any{
		&LawSearchIntentV1{},
		&LawContentSearchIntentV1{},
		&LawReadIntentV1{},
		&LawArticleReadIntentV1{},
		&LawUpdateListIntentV1{},
		&JudicialDecisionSearchIntentV1{},
		&JudicialDecisionReadIntentV1{},
	}
	for _, target := range targets {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-022: %T を JSON から直接復元できました", target)
		}
	}
}

func TestLogicalReadInputsDeferProviderSourceAdoptionToRegistry(t *testing.T) {
	t.Parallel()

	lawRef := newLegalQuerySourceResourceRef(
		t,
		"law-provider",
		"independent-law-source",
		"law",
		"law-1",
	)
	if _, err := NewLawReadIntentV1(LawReadIntentV1Values{Ref: &lawRef}); err != nil {
		t.Fatalf("SOT-ARCH-022: 構造的に有効な law ref を拒否しました: %v", err)
	}
	if _, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
		Ref:      &lawRef,
		Location: mustLawArticleLocation(t),
	}); err != nil {
		t.Fatalf("SOT-ARCH-022: 構造的に有効な article ref を拒否しました: %v", err)
	}

	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"judicial-provider",
		"independent-judicial-source",
		"judicial-decision",
		"decision-1",
	)
	if _, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: judicialRef},
	); err != nil {
		t.Fatalf("SOT-ARCH-022: 構造的に有効な judicial ref を拒否しました: %v", err)
	}
}

func assertOptionalDate(
	t *testing.T,
	getter func() (model.Date, bool),
	want string,
) {
	t.Helper()
	got, exists := getter()
	if !exists || got.String() != want {
		t.Fatalf("SOT-MODEL-022: optional date = %q, %t", got.String(), exists)
	}
}

func assertOptionalString(
	t *testing.T,
	getter func() (string, bool),
	want string,
) {
	t.Helper()
	got, exists := getter()
	if !exists || got != want {
		t.Fatalf("SOT-MODEL-022: optional string = %q, %t", got, exists)
	}
}

func mustDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用 Date を作成できません: %v", err)
	}
	return date
}

func mustLawArticleLocation(t *testing.T) model.LawArticleLocation {
	t.Helper()
	paragraph := 2
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       model.LawArticleProvisionMain,
		ArticleNumber:   "22",
		ParagraphNumber: &paragraph,
	})
	if err != nil {
		t.Fatalf("試験用 LawArticleLocation を作成できません: %v", err)
	}
	return location
}

func newVersionedSourceResourceRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}

func newLegalQuerySourceResourceRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}
