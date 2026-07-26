package lawv2

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const continuationLifetime = 15 * time.Minute

type resumePosition struct {
	offset int
	asOf   model.Date
}

func conditionFingerprint(
	manager *continuation.Manager,
	request lawsearch.Request,
) (continuation.ConditionFingerprint, error) {
	condition, err := request.ConditionObject()
	if err != nil {
		return continuation.ConditionFingerprint{}, err
	}
	return manager.FingerprintCondition(condition)
}

func configFingerprint(
	manager *continuation.Manager,
) (continuation.ConfigFingerprint, error) {
	semanticConfig, err := continuation.NewJSONObject([]byte("{}"))
	if err != nil {
		return continuation.ConfigFingerprint{}, err
	}
	return manager.FingerprintConfigScope(continuation.ConfigScopeValues{
		Provider:       Descriptor(),
		Origin:         "https://laws.e-gov.go.jp",
		Dataset:        "n/a",
		Tenant:         "n/a",
		Account:        "n/a",
		Proxy:          "n/a",
		SemanticConfig: semanticConfig,
		AllowedSlots:   []string{},
		Credentials:    []continuation.Credential{},
	})
}

func verifyContinuation(
	manager *continuation.Manager,
	request lawsearch.Request,
	condition continuation.ConditionFingerprint,
	config continuation.ConfigFingerprint,
) (resumePosition, error) {
	token, exists := request.ContinuationToken()
	if !exists {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	cursor, err := manager.Verify(continuation.VerifyInput{
		Token:      token,
		Provider:   Descriptor(),
		Capability: lawSearchCapability(),
		Limit:      request.Limit(),
		Condition:  condition,
		Config:     config,
		MaxTTL:     continuationLifetime,
	})
	if err != nil {
		return resumePosition{}, continuation.ErrInvalidToken
	}

	offset, err := decodePosition(cursor.Position())
	if err != nil {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	snapshot, exists := cursor.Snapshot()
	if !exists {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	asOf, err := decodeSnapshot(snapshot)
	if err != nil {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	sort, exists := cursor.Sort()
	if !exists || !validSort(sort) {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	if requestedAsOf, specified := request.AsOf(); specified &&
		requestedAsOf != asOf {
		return resumePosition{}, continuation.ErrInvalidToken
	}
	return resumePosition{offset: offset, asOf: asOf}, nil
}

func issueContinuation(
	manager *continuation.Manager,
	request lawsearch.Request,
	condition continuation.ConditionFingerprint,
	config continuation.ConfigFingerprint,
	offset int,
	asOf model.Date,
) (string, error) {
	position, err := continuation.NewJSONValue(
		[]byte(`{"offset":` + jsonInteger(offset) + `}`),
	)
	if err != nil {
		return "", err
	}
	snapshot, err := continuation.NewJSONValue(
		[]byte(`{"asOf":` + jsonString(asOf.String()) + `}`),
	)
	if err != nil {
		return "", err
	}
	sort, err := continuation.NewJSONValue(
		[]byte(`{"order":"+law_info.law_id"}`),
	)
	if err != nil {
		return "", err
	}
	return manager.Issue(continuation.IssueInput{
		Provider:   Descriptor(),
		Capability: lawSearchCapability(),
		Limit:      request.Limit(),
		Condition:  condition,
		Config:     config,
		Position:   position,
		Snapshot:   &snapshot,
		Sort:       &sort,
		TTL:        continuationLifetime,
		MaxTTL:     continuationLifetime,
	})
}

func decodePosition(value continuation.JSONValue) (int, error) {
	var decoded struct {
		Offset *int64 `json:"offset"`
	}
	if err := decodeExactObject(value.Bytes(), &decoded); err != nil ||
		decoded.Offset == nil ||
		*decoded.Offset < 0 ||
		*decoded.Offset > math.MaxInt32 {
		return 0, continuation.ErrInvalidToken
	}
	return int(*decoded.Offset), nil
}

func decodeSnapshot(value continuation.JSONValue) (model.Date, error) {
	var decoded struct {
		AsOf *string `json:"asOf"`
	}
	if err := decodeExactObject(value.Bytes(), &decoded); err != nil ||
		decoded.AsOf == nil {
		return model.Date{}, continuation.ErrInvalidToken
	}
	asOf, err := model.NewDate(*decoded.AsOf)
	if err != nil {
		return model.Date{}, continuation.ErrInvalidToken
	}
	return asOf, nil
}

func validSort(value continuation.JSONValue) bool {
	var decoded struct {
		Order *string `json:"order"`
	}
	return decodeExactObject(value.Bytes(), &decoded) == nil &&
		decoded.Order != nil &&
		*decoded.Order == "+law_info.law_id"
}

func decodeExactObject(value []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return continuation.ErrInvalidToken
	}
	return nil
}

func jsonInteger(value int) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
