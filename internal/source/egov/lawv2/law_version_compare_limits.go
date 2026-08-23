package lawv2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawVersionCompareMaximumArticles    = 10000
	lawVersionCompareMaximumTextBytes   = 8 * 1024 * 1024
	lawVersionCompareMaximumChanges     = 10000
	lawVersionCompareProcessingTimeout  = 3 * time.Second
	lawVersionCompareMaximumResultBytes = 12 * 1024 * 1024
)

type lawVersionCompareLimits struct {
	articlesPerVersion int
	combinedTextBytes  int
	changes            int
	processingTimeout  time.Duration
	resultBytes        int
}

func defaultLawVersionCompareLimits() lawVersionCompareLimits {
	return lawVersionCompareLimits{
		articlesPerVersion: lawVersionCompareMaximumArticles,
		combinedTextBytes:  lawVersionCompareMaximumTextBytes,
		changes:            lawVersionCompareMaximumChanges,
		processingTimeout:  lawVersionCompareProcessingTimeout,
		resultBytes:        lawVersionCompareMaximumResultBytes,
	}
}

func (l lawVersionCompareLimits) validate() error {
	if l.articlesPerVersion < 1 || l.combinedTextBytes < 1 || l.changes < 1 ||
		l.processingTimeout <= 0 || l.resultBytes < 1 {
		return fmt.Errorf("e-Gov law.version.compare の資源上限が有効ではありません")
	}
	return nil
}

func validateLawVersionCompareTextBudget(
	before parsedLawVersionDocument,
	after parsedLawVersionDocument,
	limits lawVersionCompareLimits,
) error {
	if int64(before.textBytes)+int64(after.textBytes) > int64(limits.combinedTextBytes) {
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	return nil
}

func validateLawVersionCompareResultBudget(
	comparison model.LawVersionComparison,
	limits lawVersionCompareLimits,
) error {
	payload, err := json.Marshal(comparison)
	if err != nil {
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if len(payload) > limits.resultBytes {
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	return nil
}
