package legalquerycorpus

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type rawRequestIdentity struct {
	query           string
	refPresent      bool
	providerID      string
	sourceID        string
	resourceType    string
	resourceID      string
	versionPresent  bool
	versionID       string
	limitPresent    bool
	limitPerAttempt int
}

type semanticSetSeparationIndex struct {
	requests       map[rawRequestIdentity]struct{}
	comparisonKeys map[string]struct{}
	leakageGroups  map[string]struct{}
}

// validateSemanticSetSeparation は、開発用集合と holdout 集合の意味漏出を拒否する。
func validateSemanticSetSeparation(
	development []SemanticCase,
	holdout []SemanticCase,
) error {
	index := newSemanticSetSeparationIndex(development)
	for _, semanticCase := range holdout {
		if _, exists := index.requests[rawRequestIdentityOf(
			semanticCase.Request(),
		)]; exists {
			return fmt.Errorf(
				"development と holdout の完全 request が重複しています",
			)
		}
	}
	for _, semanticCase := range holdout {
		if _, exists := index.comparisonKeys[semanticComparisonKey(
			semanticCase,
		)]; exists {
			return fmt.Errorf(
				"development と holdout の query 比較キーが重複しています",
			)
		}
	}
	for _, semanticCase := range holdout {
		if _, exists := index.leakageGroups[semanticCase.LeakageGroupID()]; exists {
			return fmt.Errorf(
				"development と holdout の leakageGroupId が重複しています",
			)
		}
	}
	return nil
}

func newSemanticSetSeparationIndex(
	development []SemanticCase,
) semanticSetSeparationIndex {
	index := semanticSetSeparationIndex{
		requests:       make(map[rawRequestIdentity]struct{}, len(development)),
		comparisonKeys: make(map[string]struct{}, len(development)),
		leakageGroups:  make(map[string]struct{}, len(development)),
	}
	for _, semanticCase := range development {
		index.requests[rawRequestIdentityOf(semanticCase.Request())] = struct{}{}
		index.comparisonKeys[semanticComparisonKey(semanticCase)] = struct{}{}
		index.leakageGroups[semanticCase.LeakageGroupID()] = struct{}{}
	}
	return index
}

func semanticComparisonKey(semanticCase SemanticCase) string {
	query := strings.TrimFunc(semanticCase.Request().Query(), unicode.IsSpace)
	return querynormalization.ComparisonKey(query)
}

func rawRequestsEqual(left Request, right Request) bool {
	return rawRequestIdentityOf(left) == rawRequestIdentityOf(right)
}

func rawRequestIdentityOf(request Request) rawRequestIdentity {
	identity := rawRequestIdentity{query: request.Query()}
	if limit, exists := request.LimitPerAttempt(); exists {
		identity.limitPresent = true
		identity.limitPerAttempt = limit
	}
	ref, exists := request.Ref()
	if !exists {
		return identity
	}
	identity.refPresent = true
	identity.providerID = ref.ProviderID()
	key := ref.Key()
	identity.sourceID = key.SourceID()
	identity.resourceType = key.ResourceType()
	identity.resourceID = key.ResourceID()
	if versionID, versionExists := key.VersionID(); versionExists {
		identity.versionPresent = true
		identity.versionID = versionID
	}
	return identity
}
