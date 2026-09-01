package lawv1

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func mustTestDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date %q を作成できません: %v", value, err)
	}
	return date
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	//nolint:gosec // SOT-IF-035: テスト専用の固定 fixtures ディレクトリだけを読み込む。
	value, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("fixture %q を読み込めません: %v", name, err)
	}
	return string(value)
}

func testResponse(
	status int,
	body string,
	headers map[string]string,
) *http.Response {
	header := make(http.Header)
	for name, value := range headers {
		header.Set(name, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func mustTestClient(
	t *testing.T,
	dependencies clientDependencies,
) updateListClient {
	t.Helper()
	client, err := newUpdateListClient(dependencies)
	if err != nil {
		t.Fatalf("test client を作成できません: %v", err)
	}
	return client
}

func noSleep(
	_ context.Context,
	_ time.Duration,
) error {
	return nil
}

func syntheticUpdateListResponse(date string, itemCount int) string {
	var body strings.Builder
	body.WriteString(
		`<?xml version="1.0" encoding="UTF-8"?>` +
			`<DataRoot><Result><Code>0</Code><Message/></Result>` +
			`<ApplData><Date>` + date + `</Date>`,
	)
	for range itemCount {
		body.WriteString(
			`<LawNameListInfo>` +
				`<LawTypeName>法律</LawTypeName>` +
				`<LawNo>令和八年法律第一号</LawNo>` +
				`<LawName>試験法令</LawName>` +
				`<LawNameKana>しけんほうれい</LawNameKana>` +
				`<OldLawName/>` +
				`<PromulgationDate>20260624</PromulgationDate>` +
				`<AmendName>試験改正</AmendName>` +
				`<AmendNo>令和八年法律第二号</AmendNo>` +
				`<AmendPromulgationDate>20260624</AmendPromulgationDate>` +
				`<EnforcementDate>20260624</EnforcementDate>` +
				`<EnforcementComment>施行済み</EnforcementComment>` +
				`<LawId>506AC0000000001</LawId>` +
				`<LawUrl>https://elaws.e-gov.go.jp/document?lawid=506AC0000000001</LawUrl>` +
				`<EnforcementFlg>0</EnforcementFlg>` +
				`<AuthFlg>0</AuthFlg>` +
				`</LawNameListInfo>`,
		)
	}
	body.WriteString(`</ApplData></DataRoot>`)
	return body.String()
}

func countXMLStructureUnits(t *testing.T, body string) int {
	t.Helper()

	count := 0
	decoder := xml.NewDecoder(strings.NewReader(body))
	for {
		token, err := decoder.RawToken()
		if errors.Is(err, io.EOF) {
			return count
		}
		if err != nil {
			t.Fatalf("合成 XML の構造単位を数えられません: %v", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			count += 1 + len(value.Attr)
		case xml.CharData:
			if strings.TrimSpace(string(value)) != "" {
				count++
			}
		case xml.Comment, xml.ProcInst:
			count++
		}
	}
}

func assertSourceErrorCode(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error = %v、SourceError ではありません", err)
	}
	if sourceError.Code() != want {
		t.Fatalf("code = %q、期待値は %q です", sourceError.Code(), want)
	}
}
