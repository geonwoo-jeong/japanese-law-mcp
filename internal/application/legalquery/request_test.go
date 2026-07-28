package legalquery_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if legalquery.DefaultLimitPerAttempt != 10 ||
		legalquery.MaxLimitPerAttempt != 20 ||
		legalquery.MaxQueryBytes != 2048 {
		t.Fatal("SOT-IF-051: 統合照会の入力上限が契約と一致しない")
	}
}

func TestNewRequestNormalizesAndCopiesInput(t *testing.T) {
	t.Parallel()

	ref := newSourceResourceRef(t, "e-gov-law-api-v2", "e-gov-law-api-v2", "law", "law-1")
	originalRef := ref
	limit := 12
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query:           "\u3000\u00a0  民  法  \u00a0",
		Ref:             &ref,
		LimitPerAttempt: &limit,
	})
	if err != nil {
		t.Fatalf("SOT-IF-051: NewRequest() のエラー = %v", err)
	}

	ref = newSourceResourceRef(t, "courts-hanrei-html", "courts-hanrei", "judicial-decision", "95570/detail2")
	limit = 1

	if request.Query() != "民  法" {
		t.Fatalf("SOT-IF-051: Query() = %q", request.Query())
	}
	gotRef, exists := request.Ref()
	if !exists || gotRef != originalRef {
		t.Fatalf("SOT-MODEL-016/SOT-IF-051: Ref() = %#v, %t", gotRef, exists)
	}
	if request.LimitPerAttempt() != 12 {
		t.Fatalf("SOT-IF-051: LimitPerAttempt() = %d", request.LimitPerAttempt())
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-051: Validate() のエラー = %v", err)
	}
}

func TestNewRequestAppliesDefaults(t *testing.T) {
	t.Parallel()

	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: "民法を検索"})
	if err != nil {
		t.Fatalf("SOT-IF-051: NewRequest() のエラー = %v", err)
	}
	if request.LimitPerAttempt() != legalquery.DefaultLimitPerAttempt {
		t.Fatalf("SOT-IF-051: 既定 limitPerAttempt = %d", request.LimitPerAttempt())
	}
	if ref, exists := request.Ref(); exists || ref != (model.SourceResourceRef{}) {
		t.Fatalf("SOT-IF-051: 省略した Ref() = %#v, %t", ref, exists)
	}
}

func TestNewRequestAcceptsQueryByteBoundary(t *testing.T) {
	t.Parallel()

	query := strings.Repeat("法", 682) + "ab"
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf("SOT-IF-051: 2048 byte の query を拒否した: %v", err)
	}
	if len(request.Query()) != legalquery.MaxQueryBytes {
		t.Fatalf("SOT-IF-051: query の byte 数 = %d", len(request.Query()))
	}
}

func TestNewRequestRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"空文字":            "",
		"Unicode 空白だけ":   "\u3000 \u00a0",
		"内部タブ":           "民\t法",
		"内部改行":           "民\n法",
		"内部 NUL":         "民\x00法",
		"DEL":            "民\x7f法",
		"先頭の ASCII 制御文字": "\n民法",
		"末尾の ASCII 制御文字": "民法\n",
		"不正な UTF-8":      string([]byte{'a', 0xff}),
		"2048 byte 超":    strings.Repeat("法", 683),
	}
	for name, query := range tests {
		name, query := name, query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewRequest(
				legalquery.RequestValues{Query: query},
			); err == nil {
				t.Fatalf("SOT-IF-051: 不正な query %q を受理した", query)
			} else {
				assertArgumentError(t, err, "query")
			}
		})
	}
}

func TestNewRequestRejectsInvalidLimitPerAttempt(t *testing.T) {
	t.Parallel()

	for _, value := range []int{-1, 0, legalquery.MaxLimitPerAttempt + 1} {
		value := value
		t.Run("limitPerAttempt", func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewRequest(legalquery.RequestValues{
				Query:           "民法を検索",
				LimitPerAttempt: &value,
			}); err == nil {
				t.Fatalf("SOT-IF-051: limitPerAttempt=%d を受理した", value)
			} else {
				assertArgumentError(t, err, "limitPerAttempt")
			}
		})
	}
}

func TestNewRequestRejectsInvalidReference(t *testing.T) {
	t.Parallel()

	tests := map[string]model.SourceResourceRef{
		"構造が不正": {},
		"対象外の resourceType": newSourceResourceRef(
			t,
			"e-gov-law-api-v1",
			"e-gov-law-api-v1",
			"law-update-list",
			"2026-07-27",
		),
	}
	for name, ref := range tests {
		name, ref := name, ref
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewRequest(legalquery.RequestValues{
				Query: "この資源を読む",
				Ref:   &ref,
			}); err == nil {
				t.Fatal("SOT-MODEL-016/SOT-IF-051: 不正な ref を受理した")
			} else {
				assertArgumentError(t, err, "ref")
			}
		})
	}
}

func TestNewRequestAcceptsJudicialReferenceBeforePackDecision(t *testing.T) {
	t.Parallel()

	ref := newSourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "この裁判例を読む",
		Ref:   &ref,
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-019/SOT-IF-051: pack 判定前の ref を拒否した: %v", err)
	}
	if got, exists := request.Ref(); !exists || got != ref {
		t.Fatalf("SOT-MODEL-016: Ref() = %#v, %t", got, exists)
	}
}

func TestNewRequestKeepsLanguageClassificationForPlanner(t *testing.T) {
	t.Parallel()

	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "permanent residence",
	})
	if err != nil {
		t.Fatalf("SOT-IF-051: 対象外判定前の query を入力エラーにした: %v", err)
	}
	if request.Query() != "permanent residence" {
		t.Fatalf("SOT-IF-051: Query() = %q", request.Query())
	}
}

func TestRequestRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var request legalquery.Request
	if err := json.Unmarshal([]byte(`{"query":"民法を検索"}`), &request); err == nil {
		t.Fatal("SOT-ENG-025/SOT-IF-051: Request を JSON から直接復元できた")
	}
	if err := request.Validate(); err == nil {
		t.Fatal("SOT-IF-051: Request のゼロ値を受理した")
	} else {
		assertArgumentError(t, err, "query")
	}
}

func assertArgumentError(t *testing.T, err error, field string) {
	t.Helper()

	var argumentError legalquery.ArgumentError
	if !errors.As(err, &argumentError) {
		t.Fatalf("SOT-IF-027/SOT-IF-051: ArgumentError ではありません: %T %v", err, err)
	}
	if argumentError.Code() != model.ErrorCodeInvalidArgument {
		t.Fatalf("SOT-IF-027: Code() = %q", argumentError.Code())
	}
	if argumentError.Field() != field {
		t.Fatalf("SOT-IF-027/SOT-IF-051: Field() = %q、期待値 = %q", argumentError.Field(), field)
	}
	if argumentError.Reason() == "" {
		t.Fatal("SOT-IF-027: Reason() が空です")
	}
}

func newSourceResourceRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}
