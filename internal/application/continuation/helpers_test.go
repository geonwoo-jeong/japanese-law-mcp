package continuation

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	goldenNowUnix              int64 = 1767225600
	goldenProviderID                 = "fixture-law-search-provider"
	goldenConditionFingerprint       = "BQeUoIa_eF7a213kAio-HtoShm2n7nAOTY16Q1Ki2Wc"
	goldenSlotFingerprint            = "zadxlbLu76a5B2obMqmB-WZliCIXtA6n4AR0oKY7gVg"
	goldenConfigFingerprint          = "oXKjfzseFA5fIaI8lihX-KaVZAgdE--PBdgOb1XGvtI"
	goldenEnvelopeMAC                = "FeaKZiCP6eYkb6_O-Gmfmqp4GcMWffxCzuLCXRbPZ0E"
)

var goldenNow = time.Unix(goldenNowUnix, 0).UTC()

type continuationFixture struct {
	manager    *Manager
	provider   model.ProviderDescriptor
	capability model.ProviderCapability
	condition  ConditionFingerprint
	config     ConfigFingerprint
}

func newContinuationFixture(t *testing.T) continuationFixture {
	t.Helper()

	return newContinuationFixtureWithKey(t, fixedContinuationKey())
}

func newContinuationFixtureWithKey(t *testing.T, key []byte) continuationFixture {
	t.Helper()

	capability := newTestCapability(t, "law.search", 1)
	provider := newTestProvider(t, goldenProviderID, "1.0.0", []model.ProviderCapability{capability})
	manager := newFixedManager(t, key, goldenNow)
	conditionObject := mustJSONObject(
		t,
		[]byte(`{"query":"民法","limit":20,"asOf":"2026-07-26"}`),
	)
	condition, err := manager.FingerprintCondition(conditionObject)
	if err != nil {
		t.Fatalf("SOT-IF-016: 検索条件 fingerprint の作成エラー = %v", err)
	}
	credential, err := NewCredential("apiKey", []byte("秘密"))
	if err != nil {
		t.Fatalf("SOT-IF-026: credential の作成エラー = %v", err)
	}
	config, err := manager.FingerprintConfigScope(ConfigScopeValues{
		Provider:       provider,
		Origin:         "https://fixture.example.jp",
		Dataset:        "laws",
		Tenant:         "n/a",
		Account:        "n/a",
		Proxy:          "n/a",
		SemanticConfig: mustJSONObject(t, []byte(`{}`)),
		AllowedSlots:   []string{"apiKey"},
		Credentials:    []Credential{credential},
	})
	if err != nil {
		t.Fatalf("SOT-IF-026: 設定 scope fingerprint の作成エラー = %v", err)
	}

	return continuationFixture{
		manager:    manager,
		provider:   provider,
		capability: capability,
		condition:  condition,
		config:     config,
	}
}

func newFixedManager(t *testing.T, key []byte, now time.Time) *Manager {
	t.Helper()

	manager, err := newManager(bytes.NewReader(key), func() time.Time {
		return now
	})
	if err != nil {
		t.Fatalf("SOT-IF-016: newManager() のエラー = %v", err)
	}
	return manager
}

func fixedContinuationKey() []byte {
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index)
	}
	return key
}

func alternateContinuationKey() []byte {
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(255 - index)
	}
	return key
}

func newTestCapability(t *testing.T, id string, majorVersion int) model.ProviderCapability {
	t.Helper()

	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: majorVersion,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-013: テスト用 capability の作成エラー = %v", err)
	}
	return capability
}

func newTestProvider(
	t *testing.T,
	providerID string,
	adapterContractVersion string,
	capabilities []model.ProviderCapability,
) model.ProviderDescriptor {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "fixture-law-source",
		Name:       "継続トークン検証用情報源",
		Publisher:  "Japanese Law MCP Test",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://fixture.example.jp/docs",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: テスト用 source の作成エラー = %v", err)
	}
	verifiedAt, err := model.NewDate("2026-07-26")
	if err != nil {
		t.Fatalf("SOT-MODEL-009: テスト用日付の作成エラー = %v", err)
	}
	provider, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 source,
		AdapterContractVersion: adapterContractVersion,
		UpstreamSpecVersion:    "2",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("SOT-IF-014: テスト用 provider の作成エラー = %v", err)
	}
	return provider
}

func mustJSONValue(t *testing.T, raw []byte) JSONValue {
	t.Helper()

	value, err := NewJSONValue(raw)
	if err != nil {
		t.Fatalf("SOT-IF-016: JSON value の作成エラー = %v", err)
	}
	return value
}

func mustJSONObject(t *testing.T, raw []byte) JSONObject {
	t.Helper()

	value, err := NewJSONObject(raw)
	if err != nil {
		t.Fatalf("SOT-IF-016: JSON object の作成エラー = %v", err)
	}
	return value
}

func goldenIssueInput(t *testing.T, fixture continuationFixture) IssueInput {
	t.Helper()

	snapshot := mustJSONValue(t, []byte(`"2026-07-26T00:00:00Z"`))
	sortMarker := mustJSONValue(t, []byte(`["lawId","asc"]`))
	return IssueInput{
		Provider:   fixture.provider,
		Capability: fixture.capability,
		Limit:      20,
		Condition:  fixture.condition,
		Config:     fixture.config,
		Position:   mustJSONValue(t, []byte(`{"cursor":"next-20"}`)),
		Snapshot:   &snapshot,
		Sort:       &sortMarker,
		TTL:        15 * time.Minute,
		MaxTTL:     time.Hour,
	}
}

func goldenVerifyInput(fixture continuationFixture, token string) VerifyInput {
	return VerifyInput{
		Token:      token,
		Provider:   fixture.provider,
		Capability: fixture.capability,
		Limit:      20,
		Condition:  fixture.condition,
		Config:     fixture.config,
		MaxTTL:     time.Hour,
	}
}

func decodeTokenPayload(t *testing.T, token string) ([]byte, map[string]json.RawMessage) {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("SOT-IF-016: token の部分数 = %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil {
		t.Fatalf("SOT-IF-016: payload の base64url decode エラー = %v", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("SOT-IF-016: payload の JSON decode エラー = %v", err)
	}
	return payload, object
}

func rawJSONString(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("SOT-IF-016: JSON string の decode エラー = %v", err)
	}
	return value
}

func signContinuationPayload(key []byte, payload []byte) string {
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	prefix := "v1." + encodedPayload
	mac := hmacForDomain(key, "continuation-token-v1", []byte(prefix))
	return prefix + "." + base64.RawURLEncoding.EncodeToString(mac)
}

func hmacForDomain(key []byte, domain string, values ...[]byte) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	for _, value := range values {
		_, _ = hash.Write(value)
	}
	return hash.Sum(nil)
}

func canonicalPayloadBytes(t *testing.T, values map[string]any) []byte {
	t.Helper()

	payload, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("SOT-IF-016: テスト用 payload の encode エラー = %v", err)
	}
	return payload
}

func goldenPayloadValues() map[string]any {
	return map[string]any{
		"providerId":             goldenProviderID,
		"capabilityId":           "law.search",
		"majorVersion":           1,
		"limit":                  20,
		"adapterContractVersion": "1.0.0",
		"position":               map[string]any{"cursor": "next-20"},
		"issuedAt":               goldenNowUnix,
		"expiresAt":              goldenNowUnix + 900,
		"conditionFingerprint":   goldenConditionFingerprint,
		"configFingerprint":      goldenConfigFingerprint,
		"snapshot":               "2026-07-26T00:00:00Z",
		"sort":                   []any{"lawId", "asc"},
	}
}
