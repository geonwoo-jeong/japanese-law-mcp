package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestContentResourceOwnershipは包含する同義Cueの最長Spanを使う(
	t *testing.T,
) {
	t.Parallel()

	short := mustCoreResourceMention(t, "条文", 0, len("条文"))
	long := mustCoreResourceMention(t, "条文検索", 0, len("条文検索"))
	got := collapseOverlappingResourceMentions(
		[]legalquery.CueMention{short, long},
	)
	if len(got) != 1 ||
		got[0].Span().StartByte() != 0 ||
		got[0].Span().EndByte() != len("条文検索") ||
		got[0].Surface() != "条文検索" {
		t.Fatalf(
			"SOT-ARCH-025: collapsed resource mentions = %#v",
			got,
		)
	}
}

func mustCoreResourceMention(
	t *testing.T,
	surface string,
	startByte int,
	endByte int,
) legalquery.CueMention {
	t.Helper()

	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf("試験用 resource span を構築できません: %v", err)
	}
	mention, err := legalquery.NewCueMention(
		legalquery.CueMentionValues{
			Span:      span,
			Surface:   surface,
			ProfileID: "core",
			CueID:     "resource-law-provision",
			MatchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
	)
	if err != nil {
		t.Fatalf("試験用 resource cue を構築できません: %v", err)
	}
	return mention
}
