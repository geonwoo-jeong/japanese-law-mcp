// Package evaluators は、採用 manifest の exact evaluator version を閉じて解決する。
package evaluators

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/defaultprofile"
)

const (
	// Version1 は、初回 bootstrap が採用した標準 evaluator の意味版である。
	Version1 = "legal-query-evaluator-v1"
	// Version2 は、現在の標準 evaluator の意味版である。
	Version2 = "legal-query-evaluator-v2"
	// CurrentVersion は、新しい request が予約する evaluator version である。
	CurrentVersion = Version2
)

// New は exact version に対応する標準 evaluator だけを構築する。
func New(version string) (*defaultprofile.Evaluator, error) {
	switch version {
	case Version1:
		evaluator, err := defaultprofile.New()
		if err != nil {
			return nil, fmt.Errorf("標準 evaluator を構築できません: %w", err)
		}
		return evaluator, nil
	case Version2:
		evaluator, err := defaultprofile.NewV2()
		if err != nil {
			return nil, fmt.Errorf("標準 evaluator を構築できません: %w", err)
		}
		return evaluator, nil
	default:
		return nil, fmt.Errorf("未対応の evaluatorVersion です")
	}
}

// IsSupported は、履歴再現を含む実装済みの exact version かを返す。
func IsSupported(version string) bool {
	switch version {
	case Version1, Version2:
		return true
	default:
		return false
	}
}
