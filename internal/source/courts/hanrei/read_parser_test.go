package hanrei

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html"
)

func TestParseReadResponseRejectsUnsafeAndOversizedInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body []byte
		code model.SourceErrorCode
	}{
		{
			name: "invalid UTF-8",
			body: []byte{0xff, 0xfe},
			code: model.SourceErrorCodeUnsafeSourceContent,
		},
		{
			name: "decompressed body",
			body: bytes.Repeat([]byte("x"), maximumReadDecompressedBytes+1),
			code: model.SourceErrorCodeSourceResponseTooLarge,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseReadResponse(context.Background(), testCase.body)
			assertReadSourceError(t, err, testCase.code)
		})
	}
}

func TestDecodeReadCategoryLabelUsesReadCapabilityOnCancellation(t *testing.T) {
	t.Parallel()
	document, err := html.Parse(strings.NewReader(
		`<main><div><h4>最高裁判所</h4></div></main>`,
	))
	if err != nil {
		t.Fatal(err)
	}
	var main *html.Node
	var find func(*html.Node)
	find = func(node *html.Node) {
		if main != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "main" {
			main = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			find(child)
		}
	}
	find(document)
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	cancel()
	_, err = decodeReadCategoryLabel(ctx, main)
	assertReadSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)
}

func TestParseReadResponseContractErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		code model.SourceErrorCode
	}{
		{
			"missing title",
			`<html><head><title>別ページ</title></head><body><main id="main-contents"></main></body></html>`,
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"missing main",
			`<html><head><title>裁判例結果詳細</title></head><body></body></html>`,
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"missing dl",
			`<html><head><title>裁判例結果詳細</title></head><body><main id="main-contents"><h4>高等裁判所</h4></main></body></html>`,
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"duplicate dt",
			`<html><head><title>裁判例結果詳細</title></head><body><main id="main-contents"><h4>高等裁判所</h4><dl><dt>事件番号</dt><dt>事件名</dt><dd><p>a</p></dd></dl></main></body></html>`,
			model.SourceErrorCodeInvalidSourceResponse,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseReadResponse(context.Background(), []byte(testCase.body))
			assertReadSourceError(t, err, testCase.code)
		})
	}
}
