package continuation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const fingerprintBytes = sha256.Size

// ConditionFingerprint は、正規化済み検索条件の鍵付き fingerprint を保持する。
type ConditionFingerprint struct {
	digest      [fingerprintBytes]byte
	owner       *Manager
	initialized bool
}

// ConfigFingerprint は、provider configuration scope の鍵付き fingerprint を保持する。
type ConfigFingerprint struct {
	digest      [fingerprintBytes]byte
	owner       *Manager
	providerID  string
	initialized bool
}

// Credential は、一つの credential slot と秘密値をメモリー内だけに保持する。
type Credential struct {
	slot        string
	secret      []byte
	initialized bool
}

// ConfigScopeValues は、結果または継続位置の意味を変え得る有効設定を表す。
type ConfigScopeValues struct {
	Provider       model.ProviderDescriptor
	Origin         string
	Dataset        string
	Tenant         string
	Account        string
	Proxy          string
	SemanticConfig JSONObject
	AllowedSlots   []string
	Credentials    []Credential
}

// NewCredential は、共通の構造を検証して秘密値を複製した credential を返す。
func NewCredential(slot string, secret []byte) (Credential, error) {
	if slot == "" ||
		!utf8.ValidString(slot) ||
		strings.ContainsRune(slot, '\x00') ||
		len(secret) == 0 ||
		!utf8.Valid(secret) {
		return Credential{}, errInvalidCredential
	}
	return Credential{
		slot:        slot,
		secret:      append([]byte(nil), secret...),
		initialized: true,
	}, nil
}

// String は、credential slot と秘密値を公開しない表現を返す。
func (Credential) String() string {
	return "continuation.Credential"
}

// GoString は、credential slot と秘密値を公開しない Go 構文表現を返す。
func (Credential) GoString() string {
	return "continuation.Credential"
}

// String は、fingerprint を公開しない表現を返す。
func (ConditionFingerprint) String() string {
	return "continuation.ConditionFingerprint"
}

// GoString は、fingerprint を公開しない Go 構文表現を返す。
func (ConditionFingerprint) GoString() string {
	return "continuation.ConditionFingerprint"
}

// String は、fingerprint を公開しない表現を返す。
func (ConfigFingerprint) String() string {
	return "continuation.ConfigFingerprint"
}

// GoString は、fingerprint を公開しない Go 構文表現を返す。
func (ConfigFingerprint) GoString() string {
	return "continuation.ConfigFingerprint"
}

// FingerprintCondition は、canonical な検索条件 object の鍵付き fingerprint を返す。
func (m *Manager) FingerprintCondition(
	condition JSONObject,
) (ConditionFingerprint, error) {
	if !m.valid() || !condition.valid() {
		return ConditionFingerprint{}, errInvalidIssueInput
	}
	digest := m.domainHMAC("continuation-condition-v1", condition.value.canonical)
	return ConditionFingerprint{
		digest:      digest,
		owner:       m,
		initialized: true,
	}, nil
}

// FingerprintConfigScope は、八項目の provider configuration scope を fingerprint にする。
func (m *Manager) FingerprintConfigScope(
	values ConfigScopeValues,
) (ConfigFingerprint, error) {
	if !m.valid() || !validConfigScopeValues(values) {
		return ConfigFingerprint{}, errInvalidConfigScope
	}

	credentialSlots, ok := m.fingerprintCredentialSlots(
		values.Provider.ProviderID(),
		values.AllowedSlots,
		values.Credentials,
	)
	if !ok {
		return ConfigFingerprint{}, errInvalidConfigScope
	}

	candidate, err := json.Marshal(struct {
		Account         string            `json:"account"`
		CredentialSlots map[string]string `json:"credentialSlots"`
		Dataset         string            `json:"dataset"`
		Origin          string            `json:"origin"`
		ProviderID      string            `json:"providerId"`
		Proxy           string            `json:"proxy"`
		SemanticConfig  json.RawMessage   `json:"semanticConfig"`
		Tenant          string            `json:"tenant"`
	}{
		Account:         values.Account,
		CredentialSlots: credentialSlots,
		Dataset:         values.Dataset,
		Origin:          values.Origin,
		ProviderID:      values.Provider.ProviderID(),
		Proxy:           values.Proxy,
		SemanticConfig:  values.SemanticConfig.Bytes(),
		Tenant:          values.Tenant,
	})
	if err != nil {
		return ConfigFingerprint{}, errInvalidConfigScope
	}
	canonical, err := canonicalizeJSON(candidate)
	if err != nil {
		return ConfigFingerprint{}, errInvalidConfigScope
	}

	digest := m.domainHMAC("provider-config-scope-v1", canonical)
	return ConfigFingerprint{
		digest:      digest,
		owner:       m,
		providerID:  values.Provider.ProviderID(),
		initialized: true,
	}, nil
}

func validConfigScopeValues(values ConfigScopeValues) bool {
	if err := values.Provider.Validate(); err != nil || !values.SemanticConfig.valid() {
		return false
	}
	scalars := []string{
		values.Origin,
		values.Dataset,
		values.Tenant,
		values.Account,
		values.Proxy,
	}
	for _, scalar := range scalars {
		if scalar == "" || !utf8.ValidString(scalar) {
			return false
		}
	}
	if !validAllowedSlots(values.AllowedSlots) {
		return false
	}
	return true
}

func (m *Manager) fingerprintCredentialSlots(
	providerID string,
	allowedSlots []string,
	credentials []Credential,
) (map[string]string, bool) {
	allowed := make(map[string]struct{}, len(allowedSlots))
	for _, slot := range allowedSlots {
		allowed[slot] = struct{}{}
	}
	slots := make(map[string]string, len(credentials))
	for _, credential := range credentials {
		if !credential.valid() {
			return nil, false
		}
		if _, exists := allowed[credential.slot]; !exists {
			return nil, false
		}
		if _, exists := slots[credential.slot]; exists {
			return nil, false
		}
		digest := m.credentialHMAC(providerID, credential.slot, credential.secret)
		slots[credential.slot] = base64.RawURLEncoding.EncodeToString(digest[:])
	}
	return slots, true
}

func (credential Credential) valid() bool {
	return credential.initialized &&
		credential.slot != "" &&
		utf8.ValidString(credential.slot) &&
		!strings.ContainsRune(credential.slot, '\x00') &&
		len(credential.secret) != 0 &&
		utf8.Valid(credential.secret)
}

func validAllowedSlots(slots []string) bool {
	seen := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		if slot == "" || !utf8.ValidString(slot) || strings.ContainsRune(slot, '\x00') {
			return false
		}
		if _, exists := seen[slot]; exists {
			return false
		}
		seen[slot] = struct{}{}
	}
	return true
}

func (fingerprint ConditionFingerprint) validFor(manager *Manager) bool {
	return fingerprint.initialized && fingerprint.owner == manager
}

func (fingerprint ConfigFingerprint) validFor(manager *Manager) bool {
	return fingerprint.initialized &&
		fingerprint.owner == manager &&
		fingerprint.providerID != ""
}

func (fingerprint ConfigFingerprint) boundToProvider(providerID string) bool {
	return fingerprint.providerID == providerID
}

func (m *Manager) domainHMAC(domain string, value []byte) [fingerprintBytes]byte {
	hash := hmac.New(sha256.New, m.key[:])
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(value)
	return [fingerprintBytes]byte(hash.Sum(nil))
}

func (m *Manager) credentialHMAC(
	providerID string,
	slot string,
	secret []byte,
) [fingerprintBytes]byte {
	hash := hmac.New(sha256.New, m.key[:])
	_, _ = hash.Write([]byte("credential-scope-v1"))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(providerID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(slot))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(secret)
	return [fingerprintBytes]byte(hash.Sum(nil))
}
