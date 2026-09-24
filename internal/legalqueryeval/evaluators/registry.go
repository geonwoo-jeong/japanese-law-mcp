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
	// Version3 は、候補 planning failure を評価失敗へ変換する次期意味版である。
	Version3 = "legal-query-evaluator-v3"
	// Version4 は、corpus schema v3 対応 source 世代の候補 evaluator 版である。
	Version4 = "legal-query-evaluator-v4"
	// CurrentVersion は、新しい request が予約する evaluator version である。
	CurrentVersion = Version4
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
	case Version3:
		evaluator, err := defaultprofile.NewV3()
		if err != nil {
			return nil, fmt.Errorf("標準 evaluator を構築できません: %w", err)
		}
		return evaluator, nil
	case Version4:
		evaluator, err := defaultprofile.NewV4()
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
	case Version1, Version2, Version3, Version4:
		return true
	default:
		return false
	}
}
