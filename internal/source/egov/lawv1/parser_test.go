package lawv1

import (
	"context"
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestParserはXML構造と資源予算を検証する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		code model.SourceErrorCode
	}{
		{
			name: "未知要素",
			body: `<DataRoot><Result><Code>0</Code><Message/></Result><ApplData><Date>20230201</Date><Unknown/></ApplData></DataRoot>`,
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "directive",
			body: `<!DOCTYPE DataRoot><DataRoot><Result><Code>0</Code><Message/></Result><ApplData><Date>20230201</Date></ApplData></DataRoot>`,
			code: model.SourceErrorCodeUnsafeSourceContent,
		},
		{
			name: "depth",
			body: `<DataRoot><Result><Code>0</Code><Message/></Result>` +
				`<ApplData><Date>20230201</Date><LawNameListInfo>` +
				`<LawName><TooDeep/></LawName>` +
				`</LawNameListInfo></ApplData></DataRoot>`,
			code: model.SourceErrorCodeUnsafeSourceContent,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseResponse(context.Background(), []byte(test.body))
			assertSourceErrorCode(t, err, test.code)
		})
	}
}

func TestParserはXMLnode上限の直前と超過を検証する(t *testing.T) {
	t.Parallel()

	exact := minimalResponseWithComments(maximumXMLNodes - 8)
	if _, err := parseResponse(
		context.Background(),
		[]byte(exact),
	); err != nil {
		t.Fatalf("node 数が上限と等しい XML を拒否しました: %v", err)
	}
	over := minimalResponseWithComments(maximumXMLNodes - 7)
	_, err := parseResponse(context.Background(), []byte(over))
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func TestXMLnode数は全構造単位を一つのcounterで数える(t *testing.T) {
	t.Parallel()

	state := xmlParserState{}
	tokens := []xml.Token{
		xml.StartElement{
			Name: xml.Name{Local: "DataRoot"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "attribute"}, Value: "value"},
				{Name: xml.Name{Space: "xmlns", Local: "p"}, Value: "urn:test"},
			},
		},
		xml.CharData("text"),
		xml.CharData(" \n"),
		xml.Comment("comment"),
		xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0"`)},
	}
	for _, token := range tokens {
		if err := state.countNode(token); err != nil {
			t.Fatalf("countNode() error = %v", err)
		}
	}
	if state.nodeCount != 6 {
		t.Fatalf("nodeCount = %d、期待値は 6 です", state.nodeCount)
	}
}

func TestParserは不完全重複namespaceを契約変更にする(t *testing.T) {
	t.Parallel()

	tests := []string{
		`<DataRoot>`,
		`<DataRoot>text</DataRoot>`,
		`<DataRoot xmlns="urn:changed"><Result><Code>0</Code><Message/></Result><ApplData><Date>20230201</Date></ApplData></DataRoot>`,
		`<DataRoot changed="true"><Result><Code>0</Code><Message/></Result><ApplData><Date>20230201</Date></ApplData></DataRoot>`,
		`<DataRoot><Result><Code>0</Code><Code>0</Code><Message/></Result><ApplData><Date>20230201</Date></ApplData></DataRoot>`,
		`<DataRoot><Result><Code>0</Code></Result><ApplData><Date>20230201</Date></ApplData></DataRoot>`,
	}
	for _, body := range tests {
		_, err := parseResponse(context.Background(), []byte(body))
		assertSourceErrorCode(t, err, model.SourceErrorCodeSourceContractChanged)
	}
}

func TestParserは取消と不正UTF8を扱う(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parseResponse(ctx, []byte("<DataRoot/>"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	_, err = parseResponse(context.Background(), []byte{0xff})
	assertSourceErrorCode(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestParserは内部解析期限を処理上限にする(t *testing.T) {
	t.Parallel()

	_, err := parseResponseWithTimeout(
		context.Background(),
		[]byte(`<DataRoot/>`),
		0,
	)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceProcessingLimit)
}

func minimalResponseWithComments(count int) string {
	var body strings.Builder
	body.WriteString(
		`<DataRoot><Result><Code>0</Code><Message/></Result>` +
			`<ApplData><Date>20230201</Date>`,
	)
	for range count {
		body.WriteString(`<!--x-->`)
	}
	body.WriteString(`</ApplData></DataRoot>`)
	return body.String()
}
