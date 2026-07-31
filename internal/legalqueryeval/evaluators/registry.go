// Package evaluators は、採用 manifest の exact evaluator version を閉じて解決する。
package evaluators

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/defaultprofile"
)

// Version1 は、初回 bootstrap が採用する標準 evaluator の意味版である。
const Version1 = "legal-query-evaluator-v1"

// New は exact version に対応する標準 evaluator だけを構築する。
func New(version string) (*defaultprofile.Evaluator, error) {
	switch version {
	case Version1:
		evaluator, err := defaultprofile.New()
		if err != nil {
			return nil, fmt.Errorf("標準 evaluator を構築できません: %w", err)
		}
		return evaluator, nil
	default:
		return nil, fmt.Errorf("未対応の evaluatorVersion です")
	}
}
