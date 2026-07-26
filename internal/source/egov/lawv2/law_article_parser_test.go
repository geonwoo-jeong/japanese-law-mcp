package lawv2

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestParseLawArticleResponseSelectsRawArticleAndParagraph(t *testing.T) {
	t.Parallel()

	body := readLawArticleFixture(t)
	tests := []struct {
		name      string
		location  model.LawArticleLocation
		startText string
		endText   string
		contains  string
	}{
		{
			name:      "本則の条",
			location:  mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
			startText: `<Article Num="1" Keep="article-spacing">`,
			endText:   `</Article>`,
			contains:  `第二項&amp;原文保持`,
		},
		{
			name:      "許可された階層内の条",
			location:  mustLawArticleLocation(t, model.LawArticleProvisionMain, "38_3_2", 0),
			startText: `<Article Num="38_3_2">`,
			endText:   `</Article>`,
			contains:  `枝番号`,
		},
		{
			name:      "条直下の項",
			location:  mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 2),
			startText: `<Paragraph Num = "2" Keep="paragraph-spacing">`,
			endText:   `</Paragraph>`,
			contains:  `第二項&amp;原文保持`,
		},
		{
			name:      "原始附則",
			location:  mustLawArticleLocation(t, model.LawArticleProvisionSupplementary, "1", 0),
			startText: `<Article Num="1">`,
			endText:   `</Article>`,
			contains:  `原始附則`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response, err := parseLawArticleResponse(
				context.Background(),
				body,
				test.location,
			)
			if err != nil {
				t.Fatalf("SOT-IF-012: parseLawArticleResponse() のエラー = %v", err)
			}
			if response.law.lawID != "322CO0000000016" ||
				response.law.revisionID != "322CO0000000016_20240401_506CO0000000161" ||
				response.location != test.location {
				t.Fatalf("SOT-IF-011/012: response = %#v", response)
			}
			if !strings.HasPrefix(response.content, test.startText) ||
				!strings.HasSuffix(response.content, test.endText) ||
				!strings.Contains(response.content, test.contains) {
				t.Fatalf("SOT-IF-012: content = %q", response.content)
			}
		})
	}
}

func TestParseLawArticleResponseExcludesIneligibleStructures(t *testing.T) {
	t.Parallel()

	for name, location := range map[string]model.LawArticleLocation{
		"表・引用・別 Article・Paragraph 内": mustLawArticleLocation(
			t,
			model.LawArticleProvisionMain,
			"99",
			0,
		),
		"改正法令附則": mustLawArticleLocation(
			t,
			model.LawArticleProvisionSupplementary,
			"9",
			0,
		),
	} {
		name, location := name, location
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLawArticleResponse(
				context.Background(),
				readLawArticleFixture(t),
				location,
			)
			if !errors.Is(err, lawarticleread.ErrNotFound) {
				t.Fatalf("SOT-IF-012/025: error = %v", err)
			}
		})
	}
}

func TestParseLawArticleResponseDistinguishesMissingAndAmbiguous(t *testing.T) {
	t.Parallel()

	base := string(readLawArticleFixture(t))
	tests := []struct {
		name     string
		body     string
		location model.LawArticleLocation
		want     error
	}{
		{
			name:     "条がない",
			body:     base,
			location: mustLawArticleLocation(t, model.LawArticleProvisionMain, "404", 0),
			want:     lawarticleread.ErrNotFound,
		},
		{
			name: "条が複数",
			body: strings.Replace(
				base,
				`<Article Num="38_3_2">`,
				`<Article Num="38_3_2"/><Article Num="38_3_2">`,
				1,
			),
			location: mustLawArticleLocation(t, model.LawArticleProvisionMain, "38_3_2", 0),
			want:     lawarticleread.ErrAmbiguousLocation,
		},
		{
			name:     "項がない",
			body:     base,
			location: mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 404),
			want:     lawarticleread.ErrNotFound,
		},
		{
			name: "項が複数",
			body: strings.Replace(
				base,
				`<Paragraph Num="1">`,
				`<Paragraph Num="01"/><Paragraph Num="1">`,
				1,
			),
			location: mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 1),
			want:     lawarticleread.ErrAmbiguousLocation,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLawArticleResponse(
				context.Background(),
				[]byte(test.body),
				test.location,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("SOT-IF-012/025: error = %v、期待値 = %v", err, test.want)
			}
		})
	}
}

func TestParseLawArticleResponseRejectsUnsafeXMLAndBudgets(t *testing.T) {
	location := mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0)
	for name, body := range map[string][]byte{
		"DTD": []byte(`<!DOCTYPE law_data_response [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
			<law_data_response><law_info/><revision_info/><law_full_text><Law>&xxe;</Law></law_full_text></law_data_response>`),
		"外部 entity": []byte(`<law_data_response><law_info/><revision_info/><law_full_text>
			<Law>&external;</Law></law_full_text></law_data_response>`),
		"不正な UTF-8": {
			'<', 'l', 'a', 'w', '_', 'd', 'a', 't', 'a', '_',
			'r', 'e', 's', 'p', 'o', 'n', 's', 'e', '>', 0xff,
		},
	} {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLawArticleResponse(context.Background(), body, location)
			assertLawArticleSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
		})
	}

	t.Run("parser input byte 上限", func(t *testing.T) {
		t.Parallel()

		_, err := parseLawArticleResponse(
			context.Background(),
			bytes.Repeat([]byte{' '}, lawDocumentParserInputBytes+1),
			location,
		)
		assertLawArticleSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})

	t.Run("XML depth", func(t *testing.T) {
		t.Parallel()

		body := `<law_data_response><law_info><law_id>x</law_id></law_info>` +
			`<revision_info><law_revision_id>x_y</law_revision_id><law_title>x</law_title></revision_info>` +
			`<law_full_text><Law>` + strings.Repeat("<X>", lawDocumentXMLDepth) +
			strings.Repeat("</X>", lawDocumentXMLDepth) + `</Law></law_full_text></law_data_response>`
		_, err := parseLawArticleResponse(context.Background(), []byte(body), location)
		assertLawArticleSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
	})

	t.Run("element 数", func(t *testing.T) {
		t.Parallel()

		body := `<law_data_response><law_info><law_id>x</law_id></law_info>` +
			`<revision_info><law_revision_id>x_y</law_revision_id><law_title>x</law_title></revision_info>` +
			`<law_full_text><Law>` + strings.Repeat("<X/>", lawDocumentXMLElements) +
			`</Law></law_full_text></law_data_response>`
		_, err := parseLawArticleResponse(context.Background(), []byte(body), location)
		assertLawArticleSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})
}

func TestParseLawArticleResponsePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parseLawArticleResponse(
		ctx,
		readLawArticleFixture(t),
		mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-IF-015: error = %v、期待値 = context.Canceled", err)
	}
}

func readLawArticleFixture(t *testing.T) []byte {
	t.Helper()

	body, err := os.ReadFile("fixtures/law-article-normal.xml")
	if err != nil {
		t.Fatalf("fixture を読み込めません: %v", err)
	}
	return body
}

func mustLawArticleLocation(
	t *testing.T,
	provision model.LawArticleProvision,
	article string,
	paragraph int,
) model.LawArticleLocation {
	t.Helper()

	values := model.LawArticleLocationValues{
		Provision:     provision,
		ArticleNumber: article,
	}
	if paragraph != 0 {
		values.ParagraphNumber = &paragraph
	}
	location, err := model.NewLawArticleLocation(values)
	if err != nil {
		t.Fatalf("LawArticleLocation を作成できません: %v", err)
	}
	return location
}

func assertLawArticleSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()

	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SOT-IF-012: error type = %T、期待値 = model.SourceError", err)
	}
	if sourceError.Code() != want ||
		sourceError.CapabilityID() != "law.article.read" ||
		sourceError.Operation() != string(operationLawData) {
		t.Fatalf("SOT-IF-012/017: source error = %#v", sourceError)
	}
}
