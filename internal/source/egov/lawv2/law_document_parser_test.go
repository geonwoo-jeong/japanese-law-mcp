package lawv2

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestParseLawDocumentResponseExtractsMetadataAndExactLawElement(t *testing.T) {
	t.Parallel()

	body := readLawDocumentFixture(t)
	response, err := parseLawDocumentResponse(context.Background(), body)
	if err != nil {
		t.Fatalf("SOT-IF-011: parseLawDocumentResponse() のエラー = %v", err)
	}
	if response.law.lawID != "322CO0000000016" ||
		response.law.revisionID !=
			"322CO0000000016_20240401_506CO0000000161" ||
		response.law.title != "地方自治法施行令" ||
		response.law.lawNumber != "昭和二十二年政令第十六号" ||
		response.law.promulgationDate != "1947-05-03" ||
		response.law.revisionEffectiveDate != "2024-04-01" {
		t.Fatalf("SOT-IF-009/011: law metadata = %#v", response.law)
	}
	start := bytes.Index(body, []byte("<Law "))
	endMarker := []byte("</Law>")
	end := bytes.Index(body[start:], endMarker)
	if start < 0 || end < 0 {
		t.Fatal("試験 fixture に Law 要素がない")
	}
	want := string(body[start : start+end+len(endMarker)])
	if response.content != want {
		t.Fatalf(
			"SOT-IF-011: 抽出した Law が変化した\n取得値: %q\n期待値: %q",
			response.content,
			want,
		)
	}
	if !strings.Contains(response.content, "&amp;") {
		t.Fatal("SOT-IF-011: XML entity 表現が書き換えられた")
	}
}

func TestParseLawDocumentResponseRejectsUnsafeXML(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"DTD": `<?xml version="1.0"?>
			<!DOCTYPE law_data_response [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
			<law_data_response><law_info/><revision_info/><law_full_text>
			<Law>&xxe;</Law></law_full_text></law_data_response>`,
		"外部 entity 参照": `<law_data_response><law_info/><revision_info/>
			<law_full_text><Law>&external;</Law></law_full_text></law_data_response>`,
		"不正な UTF-8": string([]byte{
			'<', 'l', 'a', 'w', '_', 'd', 'a', 't', 'a', '_',
			'r', 'e', 's', 'p', 'o', 'n', 's', 'e', '>', 0xff,
		}),
	}
	for name, body := range tests {
		body := body
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLawDocumentResponse(
				context.Background(),
				[]byte(body),
			)
			assertLawDocumentSourceError(
				t,
				err,
				model.SourceErrorCodeUnsafeSourceContent,
			)
		})
	}
}

func TestParseLawDocumentResponseRejectsStructuralViolations(t *testing.T) {
	t.Parallel()

	validLawInfo := `<law_info><law_id>322CO0000000016</law_id></law_info>`
	validRevision := `<revision_info>
		<law_revision_id>322CO0000000016_20240401_506CO0000000161</law_revision_id>
		<law_title>地方自治法施行令</law_title>
	</revision_info>`
	tests := []struct {
		name string
		body string
		code model.SourceErrorCode
	}{
		{
			name: "law_info 欠落",
			body: `<law_data_response>` + validRevision +
				`<law_full_text><Law/></law_full_text></law_data_response>`,
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "Law が二件",
			body: `<law_data_response>` + validLawInfo + validRevision +
				`<law_full_text><Law/><Law/></law_full_text></law_data_response>`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "Law が law_full_text の外",
			body: `<law_data_response>` + validLawInfo + validRevision +
				`<Law/><law_full_text/></law_data_response>`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "mapped field が重複",
			body: `<law_data_response><law_info>
				<law_id>322CO0000000016</law_id>
				<law_id>322CO0000000016</law_id>
				</law_info>` + validRevision +
				`<law_full_text><Law/></law_full_text></law_data_response>`,
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLawDocumentResponse(
				context.Background(),
				[]byte(test.body),
			)
			assertLawDocumentSourceError(t, err, test.code)
		})
	}
}

func TestParseLawDocumentResponseAppliesXMLBudgets(t *testing.T) {
	t.Parallel()

	t.Run("parser input byte 上限", func(t *testing.T) {
		t.Parallel()

		_, err := parseLawDocumentResponse(
			context.Background(),
			bytes.Repeat([]byte{' '}, lawDocumentParserInputBytes+1),
		)
		assertLawDocumentSourceError(
			t,
			err,
			model.SourceErrorCodeSourceResponseTooLarge,
		)
	})

	t.Run("element 数", func(t *testing.T) {
		t.Parallel()

		body := `<law_data_response><law_info><law_id>x</law_id></law_info>` +
			`<revision_info><law_revision_id>x_y</law_revision_id>` +
			`<law_title>x</law_title></revision_info><law_full_text><Law>` +
			strings.Repeat("<X/>", lawDocumentXMLElements) +
			`</Law></law_full_text></law_data_response>`
		_, err := parseLawDocumentResponse(
			context.Background(),
			[]byte(body),
		)
		assertLawDocumentSourceError(
			t,
			err,
			model.SourceErrorCodeSourceResponseTooLarge,
		)
	})

	t.Run("XML depth", func(t *testing.T) {
		t.Parallel()

		body := `<law_data_response><law_info><law_id>x</law_id></law_info>` +
			`<revision_info><law_revision_id>x_y</law_revision_id>` +
			`<law_title>x</law_title></revision_info><law_full_text><Law>` +
			strings.Repeat("<X>", lawDocumentXMLDepth) +
			strings.Repeat("</X>", lawDocumentXMLDepth) +
			`</Law></law_full_text></law_data_response>`
		_, err := parseLawDocumentResponse(
			context.Background(),
			[]byte(body),
		)
		assertLawDocumentSourceError(
			t,
			err,
			model.SourceErrorCodeUnsafeSourceContent,
		)
	})
}

func TestParseLawDocumentResponsePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parseLawDocumentResponse(ctx, readLawDocumentFixture(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-IF-015: error = %v、期待値 = context.Canceled", err)
	}
}

func readLawDocumentFixture(t *testing.T) []byte {
	t.Helper()

	body, err := os.ReadFile("fixtures/law-document-normal.xml")
	if err != nil {
		t.Fatalf("fixture を読み込めません: %v", err)
	}
	return body
}

func assertLawDocumentSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()

	if err == nil {
		t.Fatalf("SOT-IF-011: error = nil、期待値 = %q", want)
	}
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SOT-IF-011: error type = %T、期待値 = model.SourceError", err)
	}
	if sourceError.Code() != want {
		t.Fatalf("SOT-IF-011: code = %q、期待値 = %q", sourceError.Code(), want)
	}
}
