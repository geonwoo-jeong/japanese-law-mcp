package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestProfileは読点列挙した法令本文と複数条を個別読取りにする(
	t *testing.T,
) {
	t.Parallel()

	ref := mustCoreCompositionLawRef(t)
	generation := generateQuery(
		t,
		"この法令参照の本文、第90条、第95条を順に読んでください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("SOT-SCN-009: candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 3 ||
		steps[0].InputKind() != legalquery.InputKindLawRead ||
		steps[1].InputKind() != legalquery.InputKindLawArticleRead ||
		steps[2].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf("SOT-SCN-009: steps = %#v", steps)
	}
	read, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
	readRef, hasReadRef := read.Ref()
	if !ok || !hasReadRef || readRef != ref {
		t.Fatalf("法令本文読取り入力 = %#v", steps[0].LogicalInput())
	}
	for index, articleNumber := range []string{"90", "95"} {
		article, articleOK := steps[index+1].LogicalInput().(legalquery.LawArticleReadIntentV1)
		if !articleOK ||
			article.Location().ArticleNumber() != articleNumber {
			t.Fatalf(
				"条文読取り入力[%d] = %#v",
				index,
				steps[index+1].LogicalInput(),
			)
		}
	}
}
