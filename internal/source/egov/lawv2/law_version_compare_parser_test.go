package lawv2

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParseLawVersionDocumentSelectsOnlyAdoptedScope(t *testing.T) {
	t.Parallel()

	resource := mustLawVersionDocumentResource(t, "revision-before", `<Law>
		<LawBody>
			<MainProvision>
				<Article Num="1"><ArticleTitle> 第一条 </ArticleTitle>
					<Paragraph Num="1"><Sentence>永住&#x0A;　許可</Sentence></Paragraph>
				</Article>
				<Chapter Num="2"><Article Num="2">
					<ArticleCaption>（目的）</ArticleCaption><Paragraph Num="1"/>
				</Article></Chapter>
				<Paragraph Num="9"><Article Num="90"><Sentence>条外</Sentence></Article></Paragraph>
				<Article Num="3"><Paragraph Num="1"><QuoteStruct>
					<Article Num="91"><Sentence>引用</Sentence></Article>
				</QuoteStruct><Sentence>本文</Sentence></Paragraph></Article>
			</MainProvision>
			<SupplProvision><Article Num="1"><Sentence>原始附則</Sentence></Article></SupplProvision>
			<SupplProvision AmendLawNum=""><Article Num="9"><Sentence>改正附則</Sentence></Article></SupplProvision>
		</LawBody>
	</Law>`)

	parsed, err := parseLawVersionDocument(
		context.Background(),
		resource,
		defaultLawVersionCompareLimits(),
	)
	if err != nil {
		t.Fatalf("parseLawVersionDocument() のエラー = %v", err)
	}
	wantOrder := []string{"main\x001", "main\x002", "main\x003", "supplementary\x001"}
	if len(parsed.order) != len(wantOrder) {
		t.Fatalf("対象条 = %v", parsed.order)
	}
	for index := range wantOrder {
		if parsed.order[index] != wantOrder[index] {
			t.Fatalf("order[%d] = %q, want %q", index, parsed.order[index], wantOrder[index])
		}
	}
	first := parsed.articles["main\x001"].article
	if first.Text() != "第一条 永住 許可" {
		t.Fatalf("正規化 text = %q", first.Text())
	}
	if title, exists := first.ArticleTitle(); !exists || title != "第一条" {
		t.Fatalf("articleTitle = %q, %t", title, exists)
	}
	second := parsed.articles["main\x002"].article
	if chapter, exists := second.Location().ChapterNumber(); !exists || chapter != "2" {
		t.Fatalf("chapterNumber = %q, %t", chapter, exists)
	}
	third := parsed.articles["main\x003"].article
	if third.Text() != "引用 本文" {
		t.Fatalf("nested text = %q", third.Text())
	}
	if _, exists := parsed.articles["main\x0090"]; exists {
		t.Fatal("条外 Paragraph 内の Article を対象にしました")
	}
	if _, exists := parsed.articles["main\x0091"]; exists {
		t.Fatal("対象 Article 内の nested Article を独立対象にしました")
	}
	if _, exists := parsed.articles["supplementary\x009"]; exists {
		t.Fatal("AmendLawNum 属性を持つ改正附則を対象にしました")
	}
	location, _ := first.Citation().Location()
	if location != "main:article=1" {
		t.Fatalf("citation.location = %q", location)
	}
}

func TestCompareLawVersionDocumentsClassifiesLocationStructureAndWhitespace(t *testing.T) {
	t.Parallel()

	limits := defaultLawVersionCompareLimits()
	before := mustParsedLawVersionDocument(t, "revision-before", `<Law><LawBody><MainProvision>
		<Chapter Num="1"><Article Num="1"><Sentence>同じ本文</Sentence></Article></Chapter>
		<Article Num="2"><Paragraph Num="1"><Sentence A="1" B="2">同じ構造文字</Sentence></Paragraph></Article>
		<Article Num="3"><Sentence>空白  だけ</Sentence></Article>
	</MainProvision></LawBody></Law>`, limits)
	after := mustParsedLawVersionDocument(t, "revision-after", `<Law><LawBody><MainProvision>
		<Chapter Num="2"><Article Num="1"><Sentence>同じ本文</Sentence></Article></Chapter>
		<Article Num="2"><Paragraph Num="1"><ParagraphSentence><Sentence B="2" A="1">同じ構造文字</Sentence></ParagraphSentence></Paragraph></Article>
		<Article Num="3"><Sentence>空白&#x0A;　だけ</Sentence></Article>
	</MainProvision></LawBody></Law>`, limits)

	comparison, err := compareLawVersionDocuments(context.Background(), before, after, limits)
	if err != nil {
		t.Fatalf("compareLawVersionDocuments() のエラー = %v", err)
	}
	if comparison.ModifiedCount() != 2 || comparison.UnchangedCount() != 1 || comparison.TotalCount() != 2 {
		t.Fatalf("比較件数 = modified:%d unchanged:%d total:%d",
			comparison.ModifiedCount(), comparison.UnchangedCount(), comparison.TotalCount())
	}
	items := comparison.Items()
	if got := items[0].ChangeReasons(); len(got) != 1 || got[0] != model.LawVersionChangeReasonLocation {
		t.Fatalf("位置変更理由 = %v", got)
	}
	if got := items[1].ChangeReasons(); len(got) != 1 || got[0] != model.LawVersionChangeReasonStructure {
		t.Fatalf("構造変更理由 = %v", got)
	}
}

func TestParseLawVersionDocumentRejectsDuplicateIdentityAndUnsafeXML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		code    model.SourceErrorCode
	}{
		"条同一性の重複": {
			content: `<Law><LawBody><MainProvision><Article Num="1"/><Article Num="1"/></MainProvision></LawBody></Law>`,
			code:    model.SourceErrorCodeInvalidSourceResponse,
		},
		"DTD": {
			content: `<!DOCTYPE Law [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><Law><LawBody/></Law>`,
			code:    model.SourceErrorCodeUnsafeSourceContent,
		},
		"未知 namespace": {
			content: `<Law xmlns="urn:unknown"><LawBody/></Law>`,
			code:    model.SourceErrorCodeUnsafeSourceContent,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseLawVersionDocument(
				context.Background(),
				mustLawVersionDocumentResource(t, "revision-before", test.content),
				defaultLawVersionCompareLimits(),
			)
			assertLawVersionCompareSourceError(t, err, test.code)
		})
	}
}

func TestLawVersionCompareBudgetsFailClosedWithSmallFixtures(t *testing.T) {
	t.Parallel()

	limits := defaultLawVersionCompareLimits()
	limits.articlesPerVersion = 1
	_, err := parseLawVersionDocument(
		context.Background(),
		mustLawVersionDocumentResource(t, "revision-before", `<Law><LawBody><MainProvision>
			<Article Num="1"/><Article Num="2"/>
		</MainProvision></LawBody></Law>`),
		limits,
	)
	assertLawVersionCompareSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)

	limits = defaultLawVersionCompareLimits()
	before := mustParsedLawVersionDocument(t, "revision-before", `<Law><LawBody><MainProvision>
		<Article Num="1"><Sentence>旧</Sentence></Article>
		<Article Num="2"><Sentence>旧</Sentence></Article>
	</MainProvision></LawBody></Law>`, limits)
	after := mustParsedLawVersionDocument(t, "revision-after", `<Law><LawBody><MainProvision>
		<Article Num="1"><Sentence>新</Sentence></Article>
		<Article Num="2"><Sentence>新</Sentence></Article>
	</MainProvision></LawBody></Law>`, limits)
	limits.changes = 1
	_, err = compareLawVersionDocuments(context.Background(), before, after, limits)
	assertLawVersionCompareSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)

	comparison, err := compareLawVersionDocuments(
		context.Background(),
		before,
		after,
		defaultLawVersionCompareLimits(),
	)
	if err != nil {
		t.Fatalf("結果サイズ試験用の比較を構築できません: %v", err)
	}
	limits = defaultLawVersionCompareLimits()
	limits.resultBytes = 1
	err = validateLawVersionCompareResultBudget(comparison, limits)
	assertLawVersionCompareSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)

	limits = defaultLawVersionCompareLimits()
	limits.combinedTextBytes = 1
	err = validateLawVersionCompareTextBudget(before, after, limits)
	assertLawVersionCompareSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)
}

func mustParsedLawVersionDocument(
	t *testing.T,
	revisionID string,
	content string,
	limits lawVersionCompareLimits,
) parsedLawVersionDocument {
	t.Helper()
	parsed, err := parseLawVersionDocument(
		context.Background(),
		mustLawVersionDocumentResource(t, revisionID, content),
		limits,
	)
	if err != nil {
		t.Fatalf("比較用 Law XML を解析できません: %v", err)
	}
	return parsed
}

func mustLawVersionDocumentResource(
	t *testing.T,
	revisionID string,
	content string,
) model.SourcedResource[model.LawDocumentRepresentation] {
	t.Helper()
	if !strings.HasPrefix(revisionID, "322CO0000000016_") {
		revisionID = "322CO0000000016_" + revisionID
	}
	resource, err := mapLawDocument(lawDocumentResponse{
		law: lawSearchLaw{
			lawID:      "322CO0000000016",
			revisionID: revisionID,
			title:      "試験法",
		},
		content: content,
	}, nil, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("比較用法令表現を構築できません: %v", err)
	}
	return resource
}

func assertLawVersionCompareSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error type = %T, want model.SourceError: %v", err, err)
	}
	if sourceError.Code() != want {
		t.Fatalf("source error code = %q, want %q", sourceError.Code(), want)
	}
}
