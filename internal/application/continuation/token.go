package continuation

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	continuationTokenVersion      = "v1"
	maximumContinuationTokenBytes = 4096
)

// IssueInput は、同じ条件の続きを再開するために token へ束縛する値を表す。
type IssueInput struct {
	Provider   model.ProviderDescriptor
	Capability model.ProviderCapability
	Limit      int
	Condition  ConditionFingerprint
	Config     ConfigFingerprint
	Position   JSONValue
	Snapshot   *JSONValue
	Sort       *JSONValue
	TTL        time.Duration
	MaxTTL     time.Duration
}

// VerifyInput は、token と現在の binding、条件および設定を照合する値を表す。
type VerifyInput struct {
	Token      string
	Provider   model.ProviderDescriptor
	Capability model.ProviderCapability
	Limit      int
	Condition  ConditionFingerprint
	Config     ConfigFingerprint
	MaxTTL     time.Duration
}

// Cursor は、検証済みの provider 固有継続位置と marker を保持する。
type Cursor struct {
	position  JSONValue
	snapshot  *JSONValue
	sort      *JSONValue
	issuedAt  time.Time
	expiresAt time.Time
}

// Position は、provider が解釈する取得位置を返す。
func (c Cursor) Position() JSONValue {
	return cloneJSONValue(c.position)
}

// Snapshot は、snapshot marker と有無を返す。
func (c Cursor) Snapshot() (JSONValue, bool) {
	if c.snapshot == nil {
		return JSONValue{}, false
	}
	return cloneJSONValue(*c.snapshot), true
}

// Sort は、決定的な並び順 marker と有無を返す。
func (c Cursor) Sort() (JSONValue, bool) {
	if c.sort == nil {
		return JSONValue{}, false
	}
	return cloneJSONValue(*c.sort), true
}

// IssuedAt は、token の発行時刻を UTC で返す。
func (c Cursor) IssuedAt() time.Time {
	return c.issuedAt
}

// ExpiresAt は、token の有効期限を UTC で返す。
func (c Cursor) ExpiresAt() time.Time {
	return c.expiresAt
}

// String は、継続位置と marker を公開しない表現を返す。
func (Cursor) String() string {
	return "continuation.Cursor"
}

// GoString は、継続位置と marker を公開しない Go 構文表現を返す。
func (Cursor) GoString() string {
	return "continuation.Cursor"
}

// Issue は、検証済みの値から process 固有の継続トークンを発行する。
func (m *Manager) Issue(input IssueInput) (string, error) {
	if !m.valid() || !validIssueInput(m, input) {
		return "", errInvalidIssueInput
	}

	issuedAt := m.now().Unix()
	ttlSeconds := int64(input.TTL / time.Second)
	if issuedAt > math.MaxInt64-ttlSeconds {
		return "", errInvalidIssueInput
	}

	payload, err := encodePayload(input, issuedAt, issuedAt+ttlSeconds)
	if err != nil {
		return "", errInvalidIssueInput
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signed := continuationTokenVersion + "." + encodedPayload
	mac := m.domainHMAC("continuation-token-v1", []byte(signed))
	token := signed + "." + base64.RawURLEncoding.EncodeToString(mac[:])
	if len(token) > maximumContinuationTokenBytes {
		return "", errTokenTooLarge
	}
	return token, nil
}

// Verify は、token の署名、形式、期限および現在の再開条件を検証する。
func (m *Manager) Verify(input VerifyInput) (Cursor, error) {
	signed, encodedPayload, encodedMAC, ok := splitToken(input.Token)
	if !ok {
		return Cursor{}, ErrInvalidToken
	}
	if !m.valid() || !validVerifyInput(m, input) {
		return Cursor{}, errInvalidVerifyInput
	}
	if !input.Config.boundToProvider(input.Provider.ProviderID()) {
		return Cursor{}, ErrInvalidToken
	}

	mac, err := base64.RawURLEncoding.Strict().DecodeString(encodedMAC)
	if err != nil || len(mac) != fingerprintBytes {
		return Cursor{}, ErrInvalidToken
	}
	expectedMAC := m.domainHMAC("continuation-token-v1", []byte(signed))
	if !hmac.Equal(mac, expectedMAC[:]) {
		return Cursor{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.Strict().DecodeString(encodedPayload)
	if err != nil {
		return Cursor{}, ErrInvalidToken
	}
	canonical, err := canonicalizeJSON(payloadBytes)
	if err != nil || !hmac.Equal(payloadBytes, canonical) {
		return Cursor{}, ErrInvalidToken
	}

	payload, err := decodePayload(payloadBytes)
	if err != nil || !payloadMatches(input, payload) {
		return Cursor{}, ErrInvalidToken
	}
	now := m.now().Unix()
	if !validLifetime(payload.issuedAt, payload.expiresAt, now, input.MaxTTL) {
		return Cursor{}, ErrInvalidToken
	}
	return cursorFromPayload(payload), nil
}

func splitToken(token string) (string, string, string, bool) {
	if len(token) > maximumContinuationTokenBytes || !utf8.ValidString(token) {
		return "", "", "", false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 ||
		parts[0] != continuationTokenVersion ||
		parts[1] == "" ||
		parts[2] == "" ||
		!isRawBase64URL(parts[1]) ||
		!isRawBase64URL(parts[2]) {
		return "", "", "", false
	}
	return parts[0] + "." + parts[1], parts[1], parts[2], true
}

func isRawBase64URL(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '-' ||
			character == '_' {
			continue
		}
		return false
	}
	return true
}

func validIssueInput(manager *Manager, input IssueInput) bool {
	if !validBinding(input.Provider, input.Capability) ||
		input.Limit < 1 ||
		!input.Condition.validFor(manager) ||
		!input.Config.validFor(manager) ||
		!input.Config.boundToProvider(input.Provider.ProviderID()) ||
		!input.Position.valid() ||
		!validOptionalValue(input.Snapshot) ||
		!validOptionalValue(input.Sort) {
		return false
	}
	return validDuration(input.TTL) &&
		validDuration(input.MaxTTL) &&
		input.TTL <= input.MaxTTL
}

func validVerifyInput(manager *Manager, input VerifyInput) bool {
	return validBinding(input.Provider, input.Capability) &&
		input.Limit >= 1 &&
		input.Condition.validFor(manager) &&
		input.Config.validFor(manager) &&
		validDuration(input.MaxTTL)
}

func validBinding(
	provider model.ProviderDescriptor,
	capability model.ProviderCapability,
) bool {
	if err := provider.Validate(); err != nil {
		return false
	}
	if err := capability.Validate(); err != nil {
		return false
	}
	for _, declared := range provider.Capabilities() {
		if sameCapability(declared, capability) {
			return true
		}
	}
	return false
}

func sameCapability(left, right model.ProviderCapability) bool {
	return left.ID() == right.ID() &&
		left.MajorVersion() == right.MajorVersion()
}

func validOptionalValue(value *JSONValue) bool {
	return value == nil || value.valid()
}

func validDuration(duration time.Duration) bool {
	return duration > 0 && duration%time.Second == 0
}

func validLifetime(
	issuedAt int64,
	expiresAt int64,
	now int64,
	maxTTL time.Duration,
) bool {
	if issuedAt > now || expiresAt <= now || expiresAt <= issuedAt {
		return false
	}
	maxSeconds := int64(maxTTL / time.Second)
	if issuedAt > math.MaxInt64-maxSeconds {
		return true
	}
	return expiresAt <= issuedAt+maxSeconds
}

func cloneJSONValue(value JSONValue) JSONValue {
	return JSONValue{canonical: value.Bytes()}
}

func cursorFromPayload(payload decodedPayload) Cursor {
	position := cloneJSONValue(payload.position)
	return Cursor{
		position:  position,
		snapshot:  cloneOptionalJSONValue(payload.snapshot),
		sort:      cloneOptionalJSONValue(payload.sort),
		issuedAt:  time.Unix(payload.issuedAt, 0).UTC(),
		expiresAt: time.Unix(payload.expiresAt, 0).UTC(),
	}
}

func cloneOptionalJSONValue(value *JSONValue) *JSONValue {
	if value == nil {
		return nil
	}
	cloned := cloneJSONValue(*value)
	return &cloned
}

func encodePayload(input IssueInput, issuedAt, expiresAt int64) ([]byte, error) {
	position := json.RawMessage(input.Position.Bytes())
	candidate := tokenPayloadForEncoding{
		ProviderID:             input.Provider.ProviderID(),
		CapabilityID:           input.Capability.ID(),
		MajorVersion:           input.Capability.MajorVersion(),
		Limit:                  input.Limit,
		AdapterContractVersion: input.Provider.AdapterContractVersion(),
		Position:               position,
		IssuedAt:               issuedAt,
		ExpiresAt:              expiresAt,
		ConditionFingerprint:   encodeFingerprint(input.Condition.digest),
		ConfigFingerprint:      encodeFingerprint(input.Config.digest),
		Snapshot:               optionalRawMessage(input.Snapshot),
		Sort:                   optionalRawMessage(input.Sort),
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return nil, err
	}
	return canonicalizeJSON(encoded)
}

func optionalRawMessage(value *JSONValue) *json.RawMessage {
	if value == nil {
		return nil
	}
	raw := json.RawMessage(value.Bytes())
	return &raw
}

func encodeFingerprint(fingerprint [fingerprintBytes]byte) string {
	return base64.RawURLEncoding.EncodeToString(fingerprint[:])
}

type tokenPayloadForEncoding struct {
	ProviderID             string           `json:"providerId"`
	CapabilityID           string           `json:"capabilityId"`
	MajorVersion           int              `json:"majorVersion"`
	Limit                  int              `json:"limit"`
	AdapterContractVersion string           `json:"adapterContractVersion"`
	Position               json.RawMessage  `json:"position"`
	IssuedAt               int64            `json:"issuedAt"`
	ExpiresAt              int64            `json:"expiresAt"`
	ConditionFingerprint   string           `json:"conditionFingerprint"`
	ConfigFingerprint      string           `json:"configFingerprint"`
	Snapshot               *json.RawMessage `json:"snapshot,omitempty"`
	Sort                   *json.RawMessage `json:"sort,omitempty"`
}
