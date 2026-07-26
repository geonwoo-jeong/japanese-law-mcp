package continuation

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestFingerprintGoldenVectorsUseExactDomainsAndConfigScope(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016/SOT-IF-026: golden token の発行エラー = %v", err)
	}
	_, payload := decodeTokenPayload(t, token)
	gotCondition := rawJSONString(t, payload["conditionFingerprint"])
	gotConfig := rawJSONString(t, payload["configFingerprint"])

	conditionCanonical := []byte(`{"asOf":"2026-07-26","limit":20,"query":"民法"}`)
	conditionMAC := hmacForDomain(
		fixedContinuationKey(),
		"continuation-condition-v1",
		conditionCanonical,
	)
	if got := base64.RawURLEncoding.EncodeToString(conditionMAC); got != goldenConditionFingerprint {
		t.Fatalf("SOT-IF-016: 独立計算した condition fingerprint = %s", got)
	}
	if gotCondition != goldenConditionFingerprint {
		t.Fatalf(
			"SOT-IF-016: condition fingerprint = %s、期待値 = %s",
			gotCondition,
			goldenConditionFingerprint,
		)
	}

	credentialInput := []byte(
		"e-gov-law-api-v2\x00apiKey\x00秘密",
	)
	credentialMAC := hmacForDomain(
		fixedContinuationKey(),
		"credential-scope-v1",
		credentialInput,
	)
	if got := base64.RawURLEncoding.EncodeToString(credentialMAC); got != goldenSlotFingerprint {
		t.Fatalf("SOT-IF-026: 独立計算した credential fingerprint = %s", got)
	}
	configCanonical := []byte(
		`{"account":"n/a","credentialSlots":{"apiKey":"` +
			goldenSlotFingerprint +
			`"},"dataset":"laws","origin":"https://laws.e-gov.go.jp",` +
			`"providerId":"e-gov-law-api-v2","proxy":"n/a","semanticConfig":{},"tenant":"n/a"}`,
	)
	configMAC := hmacForDomain(
		fixedContinuationKey(),
		"provider-config-scope-v1",
		configCanonical,
	)
	if got := base64.RawURLEncoding.EncodeToString(configMAC); got != goldenConfigFingerprint {
		t.Fatalf("SOT-IF-026: 独立計算した config fingerprint = %s", got)
	}
	if gotConfig != goldenConfigFingerprint {
		t.Fatalf(
			"SOT-IF-026: config fingerprint = %s、期待値 = %s",
			gotConfig,
			goldenConfigFingerprint,
		)
	}
}

func TestConditionFingerprintUsesCanonicalJSON(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	first, err := fixture.manager.FingerprintCondition(
		mustJSONObject(t, []byte(`{"query":"民法","limit":20,"asOf":"2026-07-26"}`)),
	)
	if err != nil {
		t.Fatalf("SOT-IF-016: 一つ目の fingerprint 作成エラー = %v", err)
	}
	second, err := fixture.manager.FingerprintCondition(
		mustJSONObject(t, []byte(`{"asOf":"2026-07-26","limit":2E1,"query":"\u6c11\u6cd5"}`)),
	)
	if err != nil {
		t.Fatalf("SOT-IF-016: 二つ目の fingerprint 作成エラー = %v", err)
	}

	firstCondition, _ := fingerprintFields(t, fixture, first, fixture.config)
	secondCondition, _ := fingerprintFields(t, fixture, second, fixture.config)
	if firstCondition != secondCondition || firstCondition != goldenConditionFingerprint {
		t.Fatalf(
			"SOT-IF-016: canonical な同一条件の fingerprint = %q, %q",
			firstCondition,
			secondCondition,
		)
	}
}

func TestCredentialAndConfigScopeAreImmutableAndOrderIndependent(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	secret := []byte("秘密")
	apiKey, err := NewCredential("apiKey", secret)
	if err != nil {
		t.Fatalf("SOT-IF-026: apiKey credential の作成エラー = %v", err)
	}
	secondary, err := NewCredential("secondary", []byte("別の秘密"))
	if err != nil {
		t.Fatalf("SOT-IF-026: secondary credential の作成エラー = %v", err)
	}
	for index := range secret {
		secret[index] = 'x'
	}
	base := ConfigScopeValues{
		Provider:       fixture.provider,
		Origin:         "https://laws.e-gov.go.jp",
		Dataset:        "laws",
		Tenant:         "n/a",
		Account:        "n/a",
		Proxy:          "n/a",
		SemanticConfig: mustJSONObject(t, []byte(`{}`)),
	}
	firstScope := base
	firstScope.Credentials = []Credential{secondary, apiKey}
	first, err := fixture.manager.FingerprintConfigScope(firstScope)
	if err != nil {
		t.Fatalf("SOT-IF-026: 一つ目の config fingerprint 作成エラー = %v", err)
	}
	firstScope.Credentials[0] = apiKey
	secondScope := base
	secondScope.Credentials = []Credential{apiKey, secondary}
	second, err := fixture.manager.FingerprintConfigScope(secondScope)
	if err != nil {
		t.Fatalf("SOT-IF-026: 二つ目の config fingerprint 作成エラー = %v", err)
	}

	_, firstConfig := fingerprintFields(t, fixture, fixture.condition, first)
	_, secondConfig := fingerprintFields(t, fixture, fixture.condition, second)
	if firstConfig != secondConfig {
		t.Fatalf(
			"SOT-IF-026: credential の順序で config fingerprint が変化した: %q, %q",
			firstConfig,
			secondConfig,
		)
	}
}

func TestCredentialValidationAndSafeFormatting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		slot   string
		secret []byte
	}{
		"空の slot":           {slot: "", secret: []byte("秘密")},
		"空の secret":         {slot: "apiKey", secret: []byte{}},
		"UTF-8 ではない secret": {slot: "apiKey", secret: []byte{0xff}},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			credential, err := NewCredential(test.slot, test.secret)
			if err == nil {
				t.Fatalf("SOT-IF-026: 不正な credential を受理した: %#v", credential)
			}
			if strings.Contains(err.Error(), "秘密") {
				t.Fatalf("SOT-IF-026: credential のエラーが秘密値を含む: %v", err)
			}
		})
	}

	credential, err := NewCredential("apiKey", []byte("絶対に公開しない秘密"))
	if err != nil {
		t.Fatalf("SOT-IF-026: credential の作成エラー = %v", err)
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
		formatted := fmt.Sprintf(format, credential)
		if strings.Contains(formatted, "絶対に公開しない秘密") {
			t.Fatalf("SOT-IF-026: credential の formatting が秘密値を公開した: %s", formatted)
		}
	}
}

func TestFingerprintRejectsIncompleteConditionAndConfigScope(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	if _, err := fixture.manager.FingerprintCondition(JSONObject{}); err == nil {
		t.Fatalf("SOT-IF-016: zero value の条件 object を受理した")
	}

	valid := ConfigScopeValues{
		Provider:       fixture.provider,
		Origin:         "https://laws.e-gov.go.jp",
		Dataset:        "laws",
		Tenant:         "n/a",
		Account:        "n/a",
		Proxy:          "n/a",
		SemanticConfig: mustJSONObject(t, []byte(`{}`)),
	}
	tests := map[string]func(*ConfigScopeValues){
		"provider":       func(values *ConfigScopeValues) { values.Provider = model.ProviderDescriptor{} },
		"origin":         func(values *ConfigScopeValues) { values.Origin = "" },
		"dataset":        func(values *ConfigScopeValues) { values.Dataset = "" },
		"tenant":         func(values *ConfigScopeValues) { values.Tenant = "" },
		"account":        func(values *ConfigScopeValues) { values.Account = "" },
		"proxy":          func(values *ConfigScopeValues) { values.Proxy = "" },
		"semanticConfig": func(values *ConfigScopeValues) { values.SemanticConfig = JSONObject{} },
	}
	for name, change := range tests {
		change := change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := valid
			change(&values)
			if _, err := fixture.manager.FingerprintConfigScope(values); err == nil {
				t.Fatalf("SOT-IF-026: %s が不完全な config scope を受理した", name)
			}
		})
	}

	credential, err := NewCredential("apiKey", []byte("秘密"))
	if err != nil {
		t.Fatalf("SOT-IF-026: credential の作成エラー = %v", err)
	}
	valid.Credentials = []Credential{credential, credential}
	if _, err := fixture.manager.FingerprintConfigScope(valid); err == nil {
		t.Fatalf("SOT-IF-026: 重複した credential slot を受理した")
	}
}

func fingerprintFields(
	t *testing.T,
	fixture continuationFixture,
	condition ConditionFingerprint,
	config ConfigFingerprint,
) (string, string) {
	t.Helper()

	input := goldenIssueInput(t, fixture)
	input.Condition = condition
	input.Config = config
	input.Snapshot = nil
	input.Sort = nil
	input.TTL = time.Minute
	token, err := fixture.manager.Issue(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: fingerprint 確認用 token の発行エラー = %v", err)
	}
	_, payload := decodeTokenPayload(t, token)
	return rawJSONString(t, payload["conditionFingerprint"]),
		rawJSONString(t, payload["configFingerprint"])
}

func TestFingerprintTypesCannotBeSubstitutedByInvalidZeroValues(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	input := goldenIssueInput(t, fixture)
	input.Condition = ConditionFingerprint{}
	if _, err := fixture.manager.Issue(input); err == nil || errors.Is(err, ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: zero value condition fingerprint の発行結果 = %v", err)
	}
	input = goldenIssueInput(t, fixture)
	input.Config = ConfigFingerprint{}
	if _, err := fixture.manager.Issue(input); err == nil || errors.Is(err, ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: zero value config fingerprint の発行結果 = %v", err)
	}
}

func TestIssueRejectsConfigFingerprintBoundToAnotherProvider(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	otherProvider := newTestProvider(
		t,
		"other-provider",
		"1.0.0",
		[]model.ProviderCapability{fixture.capability},
	)
	input := goldenIssueInput(t, fixture)
	input.Provider = otherProvider

	token, err := fixture.manager.Issue(input)
	if err == nil || token != "" {
		t.Fatalf(
			"SOT-IF-016/SOT-IF-026: 別 provider の config fingerprint による発行結果 = %q, %v",
			token,
			err,
		)
	}
}
