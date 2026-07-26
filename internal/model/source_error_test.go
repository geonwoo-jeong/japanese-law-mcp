package model_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type sourceErrorExpectation struct {
	code      model.SourceErrorCode
	message   string
	retryable bool
}

type testSourceOperation string

const (
	testSourceProviderID                             = "e-gov-law-api-v2"
	testSourceOperationLaws      testSourceOperation = "GET /laws"
	testSourceOperationKeyword   testSourceOperation = "GET /keyword"
	testSourceOperationLawData   testSourceOperation = "GET /law_data/{law_id_or_num_or_revision_id}"
	testSourceOperationParseLaws testSourceOperation = "parse-law-response"
)

func (testSourceOperation) SourceOperationProviderID() string {
	return testSourceProviderID
}

func (o testSourceOperation) SourceOperationName() string {
	return string(o)
}

func (o testSourceOperation) ValidateSourceOperation() error {
	switch o {
	case testSourceOperationLaws,
		testSourceOperationKeyword,
		testSourceOperationLawData,
		testSourceOperationParseLaws:
		return nil
	default:
		return fmt.Errorf("試験用 provider に operation が定義されていません")
	}
}

type mappedSourceOperation uint8

const mappedSourceOperationLaws mappedSourceOperation = 1

func (mappedSourceOperation) SourceOperationProviderID() string {
	return testSourceProviderID
}

func (o mappedSourceOperation) SourceOperationName() string {
	switch o {
	case mappedSourceOperationLaws:
		return "GET /laws"
	default:
		return ""
	}
}

func (o mappedSourceOperation) ValidateSourceOperation() error {
	if o != mappedSourceOperationLaws {
		return fmt.Errorf("試験用 provider に operation が定義されていません")
	}
	return nil
}

type otherProviderSourceOperation string

func (otherProviderSourceOperation) SourceOperationProviderID() string {
	return "other-law-api"
}

func (o otherProviderSourceOperation) SourceOperationName() string {
	return string(o)
}

func (o otherProviderSourceOperation) ValidateSourceOperation() error {
	if o != otherProviderSourceOperation("GET /laws") {
		return fmt.Errorf("別の試験用 provider に operation が定義されていません")
	}
	return nil
}

type permissiveSourceOperation string

func (permissiveSourceOperation) SourceOperationProviderID() string {
	return testSourceProviderID
}

func (o permissiveSourceOperation) SourceOperationName() string {
	return string(o)
}

func (permissiveSourceOperation) ValidateSourceOperation() error {
	return nil
}

type structSourceOperation struct{}

func (structSourceOperation) SourceOperationProviderID() string {
	return testSourceProviderID
}

func (structSourceOperation) SourceOperationName() string {
	return "GET /laws"
}

func (structSourceOperation) ValidateSourceOperation() error {
	return nil
}

func TestSourceErrorCodes(t *testing.T) {
	t.Parallel()

	for _, expectation := range sourceErrorExpectations() {
		expectation := expectation
		t.Run(string(expectation.code), func(t *testing.T) {
			t.Parallel()

			got, err := model.NewSourceError(validSourceErrorValues(t, expectation.code))
			if err != nil {
				t.Fatalf("SOT-IF-017: NewSourceError() のエラー = %v", err)
			}
			if got.Code() != expectation.code {
				t.Fatalf("SOT-IF-017: Code() = %q", got.Code())
			}
			if got.Message() != expectation.message {
				t.Fatalf("SOT-IF-017: Message() = %q", got.Message())
			}
			if got.Error() != expectation.message {
				t.Fatalf("SOT-ENG-003/SOT-IF-017: Error() = %q", got.Error())
			}
			if !containsJapanese(got.Message()) {
				t.Fatalf("SOT-IF-017: message が日本語ではない: %q", got.Message())
			}
			if got.ProviderID() != "e-gov-law-api-v2" {
				t.Fatalf("SOT-IF-017: ProviderID() = %q", got.ProviderID())
			}
			if got.SourceID() != "e-gov-law-api-v2" {
				t.Fatalf("SOT-IF-017: SourceID() = %q", got.SourceID())
			}
			if got.CapabilityID() != "law.search" {
				t.Fatalf("SOT-IF-017: CapabilityID() = %q", got.CapabilityID())
			}
			if got.Operation() != "GET /laws" {
				t.Fatalf("SOT-IF-017: Operation() = %q", got.Operation())
			}
			if got.Retryable() != expectation.retryable {
				t.Fatalf("SOT-IF-017: Retryable() = %t", got.Retryable())
			}
			if retryAfter, exists := got.RetryAfter(); exists || retryAfter != "" {
				t.Fatalf(
					"SOT-IF-017: RetryAfter() = %q, %t",
					retryAfter,
					exists,
				)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("SOT-IF-017: Validate() のエラー = %v", err)
			}
		})
	}
}

func TestSourceErrorPreservesExplicitRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       model.SourceErrorCode
		retryAfter string
	}{
		{
			name:       "rate_limited の秒数",
			code:       model.SourceErrorCodeRateLimited,
			retryAfter: "120",
		},
		{
			name:       "source_unavailable の HTTP-date",
			code:       model.SourceErrorCodeSourceUnavailable,
			retryAfter: "Wed, 21 Oct 2015 07:28:00 GMT",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			values := validSourceErrorValues(t, test.code)
			values.RetryAfter = test.retryAfter
			got, err := model.NewSourceError(values)
			if err != nil {
				t.Fatalf("SOT-IF-017/SOT-IF-027: NewSourceError() のエラー = %v", err)
			}
			retryAfter, exists := got.RetryAfter()
			if !exists || retryAfter != test.retryAfter {
				t.Fatalf(
					"SOT-IF-017/SOT-IF-027: RetryAfter() = %q, %t",
					retryAfter,
					exists,
				)
			}
		})
	}
}

func TestSourceErrorAcceptsDefinedOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation model.SourceOperation
		want      string
	}{
		{
			name:      "法令一覧",
			operation: testSourceOperationLaws,
			want:      "GET /laws",
		},
		{
			name:      "キーワード検索",
			operation: testSourceOperationKeyword,
			want:      "GET /keyword",
		},
		{
			name:      "法令本文",
			operation: testSourceOperationLawData,
			want:      "GET /law_data/{law_id_or_num_or_revision_id}",
		},
		{
			name:      "解析",
			operation: testSourceOperationParseLaws,
			want:      "parse-law-response",
		},
		{
			name:      "内部列挙値と公開名が異なる operation",
			operation: mappedSourceOperationLaws,
			want:      "GET /laws",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			values := validSourceErrorValues(
				t,
				model.SourceErrorCodeInvalidSourceResponse,
			)
			values.Operation = test.operation
			got, err := model.NewSourceError(values)
			if err != nil {
				t.Fatalf("SOT-IF-004/SOT-IF-017: 定義済み operation を拒否した: %v", err)
			}
			if got.Operation() != test.want {
				t.Fatalf("SOT-IF-017: Operation() = %q", got.Operation())
			}
		})
	}
}

func TestSourceErrorRejectsRetryAfterForOtherCodes(t *testing.T) {
	t.Parallel()

	for _, expectation := range sourceErrorExpectations() {
		expectation := expectation
		if expectation.code == model.SourceErrorCodeRateLimited ||
			expectation.code == model.SourceErrorCodeSourceUnavailable {
			continue
		}
		t.Run(string(expectation.code), func(t *testing.T) {
			t.Parallel()

			values := validSourceErrorValues(t, expectation.code)
			values.RetryAfter = "120"
			if _, err := model.NewSourceError(values); err == nil {
				t.Fatal("SOT-IF-017/SOT-IF-027: 許可されない retryAfter を受理した")
			}
		})
	}
}

func TestSourceErrorRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*model.SourceErrorValues){
		"未知の code": func(values *model.SourceErrorValues) {
			values.Code = model.SourceErrorCode("temporary_failure")
		},
		"provider descriptor の欠落": func(values *model.SourceErrorValues) {
			values.Provider = model.ProviderDescriptor{}
		},
		"capability の欠落": func(values *model.SourceErrorValues) {
			values.Capability = model.ProviderCapability{}
		},
		"operation の欠落": func(values *model.SourceErrorValues) {
			values.Operation = nil
		},
		"operation が空白だけ": func(values *model.SourceErrorValues) {
			values.Operation = permissiveSourceOperation(" ")
		},
		"operation の無効な UTF-8": func(values *model.SourceErrorValues) {
			values.Operation = permissiveSourceOperation(string([]byte{0xff}))
		},
		"operation の改行": func(values *model.SourceErrorValues) {
			values.Operation = permissiveSourceOperation("GET /laws\nX-Input: secret")
		},
		"operation の具体化された path": func(values *model.SourceErrorValues) {
			values.Operation = testSourceOperation("GET /law_data/12345")
		},
		"operation の書込み method": func(values *model.SourceErrorValues) {
			values.Operation = testSourceOperation("POST /submit")
		},
		"operation の未定義名": func(values *model.SourceErrorValues) {
			values.Operation = testSourceOperation("download-secret")
		},
		"operation の provider が不一致": func(values *model.SourceErrorValues) {
			values.Operation = otherProviderSourceOperation("GET /laws")
		},
		"operation が列挙値型ではない": func(values *model.SourceErrorValues) {
			values.Operation = structSourceOperation{}
		},
		"retryAfter の無効な UTF-8": func(values *model.SourceErrorValues) {
			values.RetryAfter = string([]byte{0xff})
		},
		"retryAfter の制御文字": func(values *model.SourceErrorValues) {
			values.RetryAfter = "120\nX-Input: secret"
		},
	}
	for name, change := range tests {
		name := name
		change := change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := validSourceErrorValues(t, model.SourceErrorCodeRateLimited)
			change(&values)
			if _, err := model.NewSourceError(values); err == nil {
				t.Fatalf("SOT-IF-017: 不正な値を受理した: %#v", values)
			}
		})
	}
}

func TestSourceErrorRejectsCapabilityDeclarationMismatch(t *testing.T) {
	t.Parallel()

	t.Run("unsupported_capability なのに宣言済み", func(t *testing.T) {
		t.Parallel()

		values := validSourceErrorValues(t, model.SourceErrorCodeUnsupportedCapability)
		values.Provider = newSourceErrorProviderDescriptor(
			t,
			[]model.ProviderCapability{values.Capability},
		)
		if _, err := model.NewSourceError(values); err == nil {
			t.Fatal("SOT-IF-015/SOT-IF-017: 宣言済み能力を unsupported_capability とした")
		}
	})

	t.Run("unsupported_capability 以外なのに未宣言", func(t *testing.T) {
		t.Parallel()

		values := validSourceErrorValues(
			t,
			model.SourceErrorCodeInvalidSourceResponse,
		)
		values.Provider = newSourceErrorProviderDescriptor(
			t,
			[]model.ProviderCapability{},
		)
		if _, err := model.NewSourceError(values); err == nil {
			t.Fatal("SOT-IF-015/SOT-IF-017: 未宣言能力の情報源エラーを受理した")
		}
	})

	t.Run("宣言と能力属性が不一致", func(t *testing.T) {
		t.Parallel()

		values := validSourceErrorValues(
			t,
			model.SourceErrorCodeInvalidSourceResponse,
		)
		experimental, err := model.NewProviderCapability(
			model.ProviderCapabilityValues{
				ID:           values.Capability.ID(),
				MajorVersion: values.Capability.MajorVersion(),
				Level:        values.Capability.Level(),
				Stability:    model.CapabilityStabilityExperimental,
			},
		)
		if err != nil {
			t.Fatalf("SOT-MODEL-013: NewProviderCapability() のエラー = %v", err)
		}
		values.Capability = experimental
		if _, err := model.NewSourceError(values); err == nil {
			t.Fatal("SOT-IF-017: descriptor と異なる能力属性を受理した")
		}
	})
}

func TestSourceErrorCanBeDetectedAfterWrapping(t *testing.T) {
	t.Parallel()

	sourceError, err := model.NewSourceError(
		validSourceErrorValues(t, model.SourceErrorCodeSourceTimeout),
	)
	if err != nil {
		t.Fatalf("SOT-IF-017: NewSourceError() のエラー = %v", err)
	}
	wrapped := fmt.Errorf("情報源操作を完了できません: %w", sourceError)

	var got model.SourceError
	if !errors.As(wrapped, &got) {
		t.Fatalf("SOT-ENG-003: errors.As() が SourceError を検出できない: %v", wrapped)
	}
	if got.Code() != sourceError.Code() ||
		got.ProviderID() != sourceError.ProviderID() ||
		got.CapabilityID() != sourceError.CapabilityID() {
		t.Fatalf("SOT-ENG-003: errors.As() の結果 = %#v", got)
	}
}

func TestSourceErrorCannotBeUsedAsJSONBoundaryType(t *testing.T) {
	t.Parallel()

	sourceError, err := model.NewSourceError(
		validSourceErrorValues(t, model.SourceErrorCodeInvalidSourceResponse),
	)
	if err != nil {
		t.Fatalf("SOT-IF-017: NewSourceError() のエラー = %v", err)
	}
	if _, err := json.Marshal(sourceError); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-027: SourceError を直接 JSON に変換できた")
	}

	var decoded model.SourceError
	if err := json.Unmarshal([]byte(`{"code":"invalid_source_response"}`), &decoded); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-027: SourceError を JSON から直接復元できた")
	}
}

func TestZeroSourceErrorIsInvalidAndSafe(t *testing.T) {
	t.Parallel()

	var sourceError model.SourceError
	if err := sourceError.Validate(); err == nil {
		t.Fatal("SOT-IF-017: SourceError のゼロ値を有効として扱った")
	}
	if sourceError.Error() == "" {
		t.Fatal("SOT-ENG-003: SourceError のゼロ値が空のエラー文を返した")
	}
	if strings.Contains(sourceError.Error(), "temporary_failure") {
		t.Fatalf("SOT-IF-017: SourceError のゼロ値が内部値を公開した: %q", sourceError.Error())
	}
}

func validSourceErrorValues(
	t *testing.T,
	code model.SourceErrorCode,
) model.SourceErrorValues {
	t.Helper()

	capability := newCapability(t, "law.search", 1, model.CapabilityLevelCore)
	capabilities := []model.ProviderCapability{capability}
	if code == model.SourceErrorCodeUnsupportedCapability {
		capabilities = []model.ProviderCapability{}
	}
	return model.SourceErrorValues{
		Code:       code,
		Provider:   newSourceErrorProviderDescriptor(t, capabilities),
		Capability: capability,
		Operation:  testSourceOperationLaws,
	}
}

func newSourceErrorProviderDescriptor(
	t *testing.T,
	capabilities []model.ProviderCapability,
) model.ProviderDescriptor {
	t.Helper()

	values := validDescriptorValues(t)
	values.Capabilities = capabilities
	descriptor, err := model.NewProviderDescriptor(values)
	if err != nil {
		t.Fatalf("SOT-IF-014: NewProviderDescriptor() のエラー = %v", err)
	}
	return descriptor
}

func sourceErrorExpectations() []sourceErrorExpectation {
	return []sourceErrorExpectation{
		{
			code:      model.SourceErrorCodeUnsupportedCapability,
			message:   "選択したプロバイダーは、この機能に対応していません。別の情報源または機能を選択してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeUnsupportedQuery,
			message:   "指定した条件は、選択した情報源の公式な対象範囲では処理できません。条件または情報源を変更してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeConfigurationRequired,
			message:   "情報源を利用するための設定が不足しています。実行環境の設定を確認してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeSourceAuthFailed,
			message:   "情報源の認証または利用権限を確認してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeRateLimited,
			message:   "情報源の呼出し頻度が制限されています。しばらく待ってから再試行してください。",
			retryable: true,
		},
		{
			code:      model.SourceErrorCodeSourceTimeout,
			message:   "情報源との通信が期限を超えました。しばらく待ってから再試行してください。",
			retryable: true,
		},
		{
			code:      model.SourceErrorCodeSourceUnavailable,
			message:   "情報源へ一時的に接続できません。しばらく待ってから再試行してください。",
			retryable: true,
		},
		{
			code:      model.SourceErrorCodeSourceBusy,
			message:   "このプロバイダーで実行中の処理が上限に達しています。完了を待ってから再試行してください。",
			retryable: true,
		},
		{
			code:      model.SourceErrorCodeSourceContractChanged,
			message:   "情報源の仕様が確認済みの契約と一致しません。実装または情報源の更新を確認してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeInvalidSourceResponse,
			message:   "情報源から契約を満たさない応答を受信しました。入力を変えず、情報源または実装の更新を確認してください。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeSourceResponseTooLarge,
			message:   "情報源の応答が安全に処理できる大きさの上限を超えました。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeSourceProcessingLimit,
			message:   "情報源の内容を安全な処理時間内に解析できませんでした。",
			retryable: false,
		},
		{
			code:      model.SourceErrorCodeUnsafeSourceContent,
			message:   "情報源の内容を安全に処理できませんでした。",
			retryable: false,
		},
	}
}

func containsJapanese(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.In(character, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}
