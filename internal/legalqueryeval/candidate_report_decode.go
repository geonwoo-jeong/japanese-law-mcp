package legalqueryeval

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

// DecodeStandardReport は current baseline pointer と独立に canonical report を復元する。
func DecodeStandardReport(
	ctx context.Context,
	repositoryRoot string,
	raw []byte,
) (StandardReport, error) {
	if err := checkBaselineContext(ctx); err != nil {
		return StandardReport{}, err
	}
	repository, err := legalqueryartifact.OpenRepository(repositoryRoot)
	if err != nil {
		return StandardReport{}, fmt.Errorf("report schema repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	legalqueryRoot, err := openBaselineLegalQueryRoot(repository)
	if err != nil {
		return StandardReport{}, err
	}
	defer func() { _ = legalqueryRoot.Close() }()
	schema, err := loadBaselineSchema(legalqueryRoot)
	if err != nil {
		return StandardReport{}, err
	}
	report, err := decodeRepositoryBaseline(raw, schema)
	if err != nil {
		return StandardReport{}, fmt.Errorf("標準 report を復元できません: %w", err)
	}
	return report, nil
}
