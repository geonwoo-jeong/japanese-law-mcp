package qualitygate

import (
	"fmt"

	"golang.org/x/tools/cover"
)

func verifyTotalCoverage(profilePath string, threshold float64) error {
	profiles, err := cover.ParseProfiles(profilePath)
	if err != nil {
		return fmt.Errorf("coverage profile を解析できませんでした: %w", err)
	}

	var totalStatements int64
	var coveredStatements int64
	for _, profile := range profiles {
		for _, block := range profile.Blocks {
			statements := int64(block.NumStmt)
			totalStatements += statements
			if block.Count > 0 {
				coveredStatements += statements
			}
		}
	}
	if totalStatements == 0 {
		return fmt.Errorf("coverage profile に statement がありません")
	}

	coverage := float64(coveredStatements) * 100 / float64(totalStatements)
	if coverage < threshold {
		return fmt.Errorf("全体カバレッジ %.1f%% が下限 %.1f%% を下回っています", coverage, threshold)
	}
	return nil
}
