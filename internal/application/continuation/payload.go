package continuation

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
)

var requiredPayloadKeys = [...]string{
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

var allowedPayloadKeys = map[string]struct{}{
	"providerId":             {},
	"capabilityId":           {},
	"majorVersion":           {},
	"limit":                  {},
	"adapterContractVersion": {},
	"position":               {},
	"issuedAt":               {},
	"expiresAt":              {},
	"conditionFingerprint":   {},
	"configFingerprint":      {},
	"snapshot":               {},
	"sort":                   {},
}

type decodedPayload struct {
	providerID             string
	capabilityID           string
	majorVersion           int64
	limit                  int64
	adapterContractVersion string
	position               JSONValue
	issuedAt               int64
	expiresAt              int64
	conditionFingerprint   [fingerprintBytes]byte
	configFingerprint      [fingerprintBytes]byte
	snapshot               *JSONValue
	sort                   *JSONValue
}

func decodePayload(payload []byte) (decodedPayload, error) {
	if len(payload) == 0 || payload[0] != '{' {
		return decodedPayload{}, ErrInvalidToken
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil {
		return decodedPayload{}, ErrInvalidToken
	}
	if !hasExactPayloadKeys(object) {
		return decodedPayload{}, ErrInvalidToken
	}

	providerID, ok := decodeString(object["providerId"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	capabilityID, ok := decodeString(object["capabilityId"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	majorVersion, ok := decodeInteger(object["majorVersion"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	limit, ok := decodeInteger(object["limit"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	adapterVersion, ok := decodeString(object["adapterContractVersion"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	position, err := NewJSONValue(object["position"])
	if err != nil {
		return decodedPayload{}, ErrInvalidToken
	}
	issuedAt, ok := decodeInteger(object["issuedAt"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	expiresAt, ok := decodeInteger(object["expiresAt"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	conditionFingerprint, ok := decodeFingerprint(object["conditionFingerprint"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	configFingerprint, ok := decodeFingerprint(object["configFingerprint"])
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	snapshot, ok := decodeOptionalJSONValue(object, "snapshot")
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}
	sort, ok := decodeOptionalJSONValue(object, "sort")
	if !ok {
		return decodedPayload{}, ErrInvalidToken
	}

	return decodedPayload{
		providerID:             providerID,
		capabilityID:           capabilityID,
		majorVersion:           majorVersion,
		limit:                  limit,
		adapterContractVersion: adapterVersion,
		position:               position,
		issuedAt:               issuedAt,
		expiresAt:              expiresAt,
		conditionFingerprint:   conditionFingerprint,
		configFingerprint:      configFingerprint,
		snapshot:               snapshot,
		sort:                   sort,
	}, nil
}

func hasExactPayloadKeys(object map[string]json.RawMessage) bool {
	for key := range object {
		if _, allowed := allowedPayloadKeys[key]; !allowed {
			return false
		}
	}
	for _, required := range requiredPayloadKeys {
		if _, exists := object[required]; !exists {
			return false
		}
	}
	return true
}

func decodeString(raw json.RawMessage) (string, bool) {
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return "", false
	}
	return *value, true
}

func decodeInteger(raw json.RawMessage) (int64, bool) {
	var value *int64
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return 0, false
	}
	return *value, true
}

func decodeFingerprint(
	raw json.RawMessage,
) ([fingerprintBytes]byte, bool) {
	value, ok := decodeString(raw)
	if !ok {
		return [fingerprintBytes]byte{}, false
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) != fingerprintBytes {
		return [fingerprintBytes]byte{}, false
	}
	return [fingerprintBytes]byte(decoded), true
}

func decodeOptionalJSONValue(
	object map[string]json.RawMessage,
	key string,
) (*JSONValue, bool) {
	raw, exists := object[key]
	if !exists {
		return nil, true
	}
	value, err := NewJSONValue(raw)
	if err != nil {
		return nil, false
	}
	return &value, true
}

func payloadMatches(input VerifyInput, payload decodedPayload) bool {
	if payload.providerID != input.Provider.ProviderID() ||
		payload.capabilityID != input.Capability.ID() ||
		payload.majorVersion != int64(input.Capability.MajorVersion()) ||
		payload.limit != int64(input.Limit) ||
		payload.adapterContractVersion != input.Provider.AdapterContractVersion() {
		return false
	}
	return hmac.Equal(
		payload.conditionFingerprint[:],
		input.Condition.digest[:],
	) && hmac.Equal(
		payload.configFingerprint[:],
		input.Config.digest[:],
	)
}
