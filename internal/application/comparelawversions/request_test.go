package comparelawversions_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
)

func TestPublicRequestNormalizesLawIDAndKeepsSelectors(t *testing.T) {
	t.Parallel()

	before, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: "revision-before",
	})
	if err != nil {
		t.Fatalf("before selector を構築できません: %v", err)
	}
	after, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: "revision-after",
	})
	if err != nil {
		t.Fatalf("after selector を構築できません: %v", err)
	}
	request, err := comparelawversions.NewRequest(comparelawversions.RequestValues{
		LawID:  " law-1 ",
		Before: before,
		After:  after,
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	if request.LawID() != "law-1" {
		t.Fatalf("lawId = %q", request.LawID())
	}
	if revisionID, exists := request.After().RevisionID(); !exists || revisionID != "revision-after" {
		t.Fatalf("after = %#v", request.After())
	}

	if _, err := comparelawversions.NewRequest(comparelawversions.RequestValues{
		LawID:  strings.Repeat("a", 257),
		Before: before,
		After:  after,
	}); err == nil {
		t.Fatal("256 byte を超える lawId を受理しました")
	}
}
