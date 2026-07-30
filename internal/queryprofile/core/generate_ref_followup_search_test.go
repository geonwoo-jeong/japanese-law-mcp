package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは法令Ref読取り後の法令名と条文検索を原文順に保つ(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この法令参照の本文と第90条を読み、民法と成年後見の条文も検索してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("SOT-ARCH-025: candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 4 {
		t.Fatalf("SOT-MODEL-022: steps = %#v", steps)
	}
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead,
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
	}
	gotKinds := make([]legalquery.LogicalInputKind, 0, len(steps))
	for _, step := range steps {
		gotKinds = append(gotKinds, step.InputKind())
	}
	if !slices.Equal(gotKinds, wantKinds) {
		t.Fatalf(
			"SOT-ARCH-025: step 順 = %#v, want %#v",
			gotKinds,
			wantKinds,
		)
	}

	read, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
	readRef, readRefExists := read.Ref()
	if !ok || !readRefExists || readRef != ref {
		t.Fatalf("法令読取り入力 = %#v", steps[0].LogicalInput())
	}
	article, ok := steps[1].LogicalInput().(legalquery.LawArticleReadIntentV1)
	articleRef, articleRefExists := article.Ref()
	if !ok ||
		!articleRefExists ||
		articleRef != ref ||
		article.Location().ArticleNumber() != "90" {
		t.Fatalf("条文読取り入力 = %#v", steps[1].LogicalInput())
	}
	search, ok := steps[2].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || search.Query() != "民法" {
		t.Fatalf("法令検索入力 = %#v", steps[2].LogicalInput())
	}
	content, ok := steps[3].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(content.AllTerms(), []string{"成年後見"}) ||
		len(content.AnyTerms()) != 0 ||
		len(content.ExcludeTerms()) != 0 {
		t.Fatalf("条文検索入力 = %#v", steps[3].LogicalInput())
	}
}

func TestProfileは条文検索の法令範囲を別の法令名検索へ拡張しない(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この参照を読み、民法の成年後見の条文を検索してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("SOT-ARCH-025: candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawRead ||
		steps[1].InputKind() != legalquery.InputKindLawContentSearch {
		t.Fatalf(
			"SOT-ARCH-025: 法令範囲を独立検索へ拡張した steps = %#v",
			steps,
		)
	}
}
