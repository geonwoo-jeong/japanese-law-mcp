package lawv2

import (
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func lawContentConditionFingerprint(
	manager *continuation.Manager,
	request lawcontentsearch.Request,
) (continuation.ConditionFingerprint, error) {
	condition, err := request.ConditionObject()
	if err != nil {
		return continuation.ConditionFingerprint{}, err
	}
	return manager.FingerprintCondition(condition)
}

func lawContentConfigFingerprint(
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

func verifyLawContentContinuation(
	manager *continuation.Manager,
	request lawcontentsearch.Request,
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
		Capability: lawContentSearchCapability(),
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

func issueLawContentContinuation(
	manager *continuation.Manager,
	request lawcontentsearch.Request,
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
		Capability: lawContentSearchCapability(),
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

func effectiveLawContentAsOf(
	request lawcontentsearch.Request,
	startedAt time.Time,
	manager *continuation.Manager,
	condition continuation.ConditionFingerprint,
	config continuation.ConfigFingerprint,
) (model.Date, int, error) {
	if _, exists := request.ContinuationToken(); exists {
		position, err := verifyLawContentContinuation(
			manager,
			request,
			condition,
			config,
		)
		if err != nil {
			return model.Date{}, 0, err
		}
		return position.asOf, position.offset, nil
	}
	if asOf, exists := request.AsOf(); exists {
		return asOf, 0, nil
	}
	tokyo := time.FixedZone("Asia/Tokyo", 9*60*60)
	asOf, err := model.NewDate(startedAt.In(tokyo).Format(time.DateOnly))
	if err != nil {
		return model.Date{}, 0, err
	}
	return asOf, 0, nil
}
