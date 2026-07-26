package continuation

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestVerifyRejectsMalformedEnvelopeAndBase64URL(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	tests := map[string]string{
		"空":                "",
		"version だけ":       "v1",
		"二部分":              "v1.e30",
		"四部分":              "v1.e30.e30.extra",
		"未知の version":      "v2.e30.e30",
		"空の payload":       "v1..e30",
		"空の MAC":           "v1.e30.",
		"padding 付き":       "v1.e30=.e30=",
		"標準 base64 の記号":    "v1.+/8.e30",
		"空白":               "v1.e 30.e30",
		"MAC の長さ不正":        "v1.e30.YQ",
		"UTF-8 ではない token": string([]byte{'v', '1', '.', 0xff, '.', 'x'}),
		"4096 byte の不正値":   strings.Repeat("x", 4096),
		"4097 byte の超過値":   strings.Repeat("x", 4097),
	}
	for name, token := range tests {
		token := token
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := goldenVerifyInput(fixture, token)
			_, err := fixture.manager.Verify(input)
			assertInvalidTokenError(t, err, token, "民法", "秘密")
		})
	}
}

func TestVerifyRejectsLineBreaksInsideRawBase64URLSegments(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016: token の発行エラー = %v", err)
	}
	parts := strings.Split(token, ".")

	payloadWithLineBreak := parts[1][:16] + "\n" + parts[1][16:]
	signed := "v1." + payloadWithLineBreak
	mac := hmacForDomain(
		fixedContinuationKey(),
		"continuation-token-v1",
		[]byte(signed),
	)
	correctlySignedPayload := signed + "." +
		base64.RawURLEncoding.EncodeToString(mac)

	tests := map[string]string{
		"payload 内の LF": correctlySignedPayload,
		"MAC 内の LF": parts[0] + "." + parts[1] + "." +
			parts[2][:16] + "\n" + parts[2][16:],
		"MAC 内の CR": parts[0] + "." + parts[1] + "." +
			parts[2][:16] + "\r" + parts[2][16:],
	}
	for name, candidate := range tests {
		candidate := candidate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := fixture.manager.Verify(goldenVerifyInput(fixture, candidate))
			assertInvalidTokenError(t, err, candidate)
		})
	}
}

func TestVerifyAcceptsLargestValidTokenBelow4096AndRejectsNextOversizeToken(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	var largest string
	low, high := 0, 5000
	for low <= high {
		middle := low + (high-low)/2
		values := goldenPayloadValues()
		values["position"] = strings.Repeat("x", middle)
		token := signContinuationPayload(
			fixedContinuationKey(),
			canonicalPayloadBytes(t, values),
		)
		if len(token) <= 4096 {
			largest = token
			low = middle + 1
		} else {
			high = middle - 1
		}
	}
	if len(largest) < 4093 || len(largest) > 4096 {
		t.Fatalf("SOT-IF-016: 境界直下の token 長 = %d", len(largest))
	}
	if _, err := fixture.manager.Verify(goldenVerifyInput(fixture, largest)); err != nil {
		t.Fatalf("SOT-IF-016: 境界直下の有効な token の検証エラー = %v", err)
	}

	values := goldenPayloadValues()
	values["position"] = strings.Repeat("x", low)
	oversize := signContinuationPayload(
		fixedContinuationKey(),
		canonicalPayloadBytes(t, values),
	)
	if len(oversize) <= 4096 {
		t.Fatalf("SOT-IF-016: 境界超過用 token 長 = %d", len(oversize))
	}
	_, err := fixture.manager.Verify(goldenVerifyInput(fixture, oversize))
	assertInvalidTokenError(t, err, oversize)
}

func TestVerifyRejectsPayloadAndMACTampering(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016: token の発行エラー = %v", err)
	}
	parts := strings.Split(token, ".")
	tamperedPayload := parts[1]
	if tamperedPayload[0] == 'A' {
		tamperedPayload = "B" + tamperedPayload[1:]
	} else {
		tamperedPayload = "A" + tamperedPayload[1:]
	}
	tamperedMAC := parts[2]
	if tamperedMAC[len(tamperedMAC)-1] == 'A' {
		tamperedMAC = tamperedMAC[:len(tamperedMAC)-1] + "B"
	} else {
		tamperedMAC = tamperedMAC[:len(tamperedMAC)-1] + "A"
	}
	tests := map[string]string{
		"payload": parts[0] + "." + tamperedPayload + "." + parts[2],
		"MAC":     parts[0] + "." + parts[1] + "." + tamperedMAC,
	}
	for name, candidate := range tests {
		candidate := candidate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := fixture.manager.Verify(goldenVerifyInput(fixture, candidate))
			assertInvalidTokenError(t, err, candidate)
		})
	}
}

func TestVerifyRejectsCorrectMACPayloadWithUnknownDuplicateOrNoncanonicalJSON(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	canonical := canonicalPayloadBytes(t, goldenPayloadValues())
	unknown := goldenPayloadValues()
	unknown["unexpected"] = true
	caseVariant := goldenPayloadValues()
	delete(caseVariant, "providerId")
	caseVariant["ProviderId"] = "e-gov-law-api-v2"
	duplicate := append(
		[]byte(`{"adapterContractVersion":"1.0.0",`),
		canonical[1:]...,
	)
	escapedDuplicate := append(
		[]byte(`{"\u0061dapterContractVersion":"1.0.0",`),
		canonical[1:]...,
	)
	noncanonicalWhitespace := append([]byte("{ "), canonical[1:]...)
	noncanonicalNumber := []byte(strings.Replace(
		string(canonical),
		`"majorVersion":1`,
		`"majorVersion":1.0`,
		1,
	))
	tests := map[string][]byte{
		"未知の key":           canonicalPayloadBytes(t, unknown),
		"大文字小文字が異なる key":    canonicalPayloadBytes(t, caseVariant),
		"重複 key":            duplicate,
		"escape 後に重複する key": escapedDuplicate,
		"空白を含む非 canonical":  noncanonicalWhitespace,
		"非 canonical な数値":   noncanonicalNumber,
		"不正な JSON":          canonical[:len(canonical)-1],
	}
	for name, payload := range tests {
		payload := append([]byte(nil), payload...)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			token := signContinuationPayload(fixedContinuationKey(), payload)
			_, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
			assertInvalidTokenError(t, err, token)
		})
	}
}

func TestVerifyRejectsCorrectMACPayloadMissingEveryRequiredKey(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	required := []string{
		"providerId",
		"capabilityId",
		"majorVersion",
		"limit",
		"adapterContractVersion",
		"position",
		"issuedAt",
		"expiresAt",
		"conditionFingerprint",
		"configFingerprint",
	}
	for _, key := range required {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			values := goldenPayloadValues()
			delete(values, key)
			token := signContinuationPayload(
				fixedContinuationKey(),
				canonicalPayloadBytes(t, values),
			)
			_, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
			assertInvalidTokenError(t, err, token)
		})
	}
}

func TestVerifyRejectsCorrectMACPayloadWithWrongRequiredTypes(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	tests := map[string]struct {
		key   string
		value any
	}{
		"providerId":             {key: "providerId", value: 1},
		"capabilityId":           {key: "capabilityId", value: nil},
		"majorVersion の文字列":      {key: "majorVersion", value: "1"},
		"majorVersion の小数":       {key: "majorVersion", value: 1.5},
		"limit の文字列":             {key: "limit", value: "20"},
		"limit の小数":              {key: "limit", value: 20.5},
		"adapterContractVersion": {key: "adapterContractVersion", value: true},
		"issuedAt":               {key: "issuedAt", value: "1767225600"},
		"expiresAt":              {key: "expiresAt", value: nil},
		"conditionFingerprint":   {key: "conditionFingerprint", value: 1},
		"configFingerprint":      {key: "configFingerprint", value: []any{}},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := goldenPayloadValues()
			values[test.key] = test.value
			token := signContinuationPayload(
				fixedContinuationKey(),
				canonicalPayloadBytes(t, values),
			)
			_, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
			assertInvalidTokenError(t, err, token)
		})
	}
}

func TestVerifyRejectsEveryExpectedBindingAndFingerprintMismatch(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016: token の発行エラー = %v", err)
	}
	documentCapability := newTestCapability(t, "law.document.read", 1)
	versionTwoCapability := newTestCapability(t, "law.search", 2)
	otherCondition, err := fixture.manager.FingerprintCondition(
		mustJSONObject(t, []byte(`{"asOf":"2026-07-26","limit":20,"query":"商法"}`)),
	)
	if err != nil {
		t.Fatalf("SOT-IF-016: 別条件 fingerprint の作成エラー = %v", err)
	}
	otherConfig, err := fixture.manager.FingerprintConfigScope(ConfigScopeValues{
		Provider:       fixture.provider,
		Origin:         "https://laws.e-gov.go.jp",
		Dataset:        "different-laws",
		Tenant:         "n/a",
		Account:        "n/a",
		Proxy:          "n/a",
		SemanticConfig: mustJSONObject(t, []byte(`{}`)),
	})
	if err != nil {
		t.Fatalf("SOT-IF-026: 別設定 fingerprint の作成エラー = %v", err)
	}

	tests := map[string]func(*VerifyInput){
		"providerId": func(input *VerifyInput) {
			input.Provider = newTestProvider(
				t,
				"other-provider",
				"1.0.0",
				[]model.ProviderCapability{fixture.capability},
			)
		},
		"capabilityId": func(input *VerifyInput) {
			input.Provider = newTestProvider(
				t,
				"e-gov-law-api-v2",
				"1.0.0",
				[]model.ProviderCapability{documentCapability},
			)
			input.Capability = documentCapability
		},
		"majorVersion": func(input *VerifyInput) {
			input.Provider = newTestProvider(
				t,
				"e-gov-law-api-v2",
				"1.0.0",
				[]model.ProviderCapability{versionTwoCapability},
			)
			input.Capability = versionTwoCapability
		},
		"limit": func(input *VerifyInput) {
			input.Limit = 19
		},
		"adapterContractVersion": func(input *VerifyInput) {
			input.Provider = newTestProvider(
				t,
				"e-gov-law-api-v2",
				"1.0.1",
				[]model.ProviderCapability{fixture.capability},
			)
		},
		"conditionFingerprint": func(input *VerifyInput) {
			input.Condition = otherCondition
		},
		"configFingerprint": func(input *VerifyInput) {
			input.Config = otherConfig
		},
	}
	for name, change := range tests {
		change := change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := goldenVerifyInput(fixture, token)
			change(&input)
			_, err := fixture.manager.Verify(input)
			assertInvalidTokenError(t, err, token)
		})
	}
}

func TestVerifyEnforcesIssuedExpiryAndMaximumLifetimeBoundaries(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	validAtBoundary := goldenPayloadValues()
	validAtBoundary["issuedAt"] = goldenNowUnix
	validAtBoundary["expiresAt"] = goldenNowUnix + 1
	validToken := signContinuationPayload(
		fixedContinuationKey(),
		canonicalPayloadBytes(t, validAtBoundary),
	)
	validInput := goldenVerifyInput(fixture, validToken)
	validInput.MaxTTL = time.Second
	if _, err := fixture.manager.Verify(validInput); err != nil {
		t.Fatalf("SOT-IF-016: 有効期限上限ちょうどの token の検証エラー = %v", err)
	}

	tests := map[string]struct {
		issuedAt  int64
		expiresAt int64
		maxTTL    time.Duration
	}{
		"未来の issuedAt": {
			issuedAt: goldenNowUnix + 1, expiresAt: goldenNowUnix + 61, maxTTL: time.Minute,
		},
		"現在時刻より前の expiresAt": {
			issuedAt: goldenNowUnix - 60, expiresAt: goldenNowUnix - 1, maxTTL: time.Minute,
		},
		"現在時刻と同じ expiresAt": {
			issuedAt: goldenNowUnix - 60, expiresAt: goldenNowUnix, maxTTL: time.Minute,
		},
		"issuedAt と同じ expiresAt": {
			issuedAt: goldenNowUnix, expiresAt: goldenNowUnix, maxTTL: time.Minute,
		},
		"issuedAt より前の expiresAt": {
			issuedAt: goldenNowUnix, expiresAt: goldenNowUnix - 1, maxTTL: time.Minute,
		},
		"有効期限上限の一秒超過": {
			issuedAt: goldenNowUnix, expiresAt: goldenNowUnix + 61, maxTTL: time.Minute,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := goldenPayloadValues()
			values["issuedAt"] = test.issuedAt
			values["expiresAt"] = test.expiresAt
			token := signContinuationPayload(
				fixedContinuationKey(),
				canonicalPayloadBytes(t, values),
			)
			input := goldenVerifyInput(fixture, token)
			input.MaxTTL = test.maxTTL
			_, err := fixture.manager.Verify(input)
			assertInvalidTokenError(t, err, token)
		})
	}
}

func TestInvalidTokenErrorsDoNotEchoTokenQueryOrSecret(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	malformedEnvelope := "v1.秘密のtoken.民法の検索語"
	_, err := fixture.manager.Verify(goldenVerifyInput(fixture, malformedEnvelope))
	assertInvalidTokenError(t, err, malformedEnvelope, "秘密のtoken", "民法の検索語")
}

func assertInvalidTokenError(t *testing.T, err error, forbidden ...string) {
	t.Helper()

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: errors.Is(err, ErrInvalidToken) = false、エラー = %v", err)
	}
	for _, value := range forbidden {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("SOT-IF-016: invalid token エラーが入力を公開した: %v", err)
		}
	}
}
