package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは複数法令の明示条文読取りを一候補へ統合する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法第709条と刑法第199条を読んで",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("steps = %#v", steps)
	}
	want := []struct {
		lawID         string
		articleNumber string
	}{
		{lawID: "129AC0000000089", articleNumber: "709"},
		{lawID: "140AC0000000045", articleNumber: "199"},
	}
	for index, expected := range want {
		input, ok := steps[index].LogicalInput().(legalquery.LawArticleReadIntentV1)
		if !ok {
			t.Fatalf(
				"steps[%d].logicalInput = %T",
				index,
				steps[index].LogicalInput(),
			)
		}
		lawID, exists := input.LawID()
		if !exists ||
			lawID != expected.lawID ||
			input.Location().ArticleNumber() != expected.articleNumber {
			t.Fatalf("steps[%d] = %#v", index, input)
		}
	}
}

func TestProfileは別節の検索対象を共有読取りへ統合しない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法第709条を検索して、刑法第199条を読んで",
		nil,
	)
	for _, candidate := range generation.Candidates() {
		articleReads := 0
		for _, step := range candidate.Steps() {
			if step.InputKind() == legalquery.InputKindLawArticleRead {
				articleReads++
			}
		}
		if articleReads > 1 {
			t.Fatalf(
				"別節の対象を一つの読取り候補へ統合しました: %#v",
				candidate.Steps(),
			)
		}
	}
}
