package qualitygate

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const coverageScannerMaximumTokenBytes = 1024 * 1024

type coverageProfile struct {
	fileName string
	mode     string
	blocks   []coverageProfileBlock
}

type coverageProfileBlock struct {
	startLine int
	startCol  int
	endLine   int
	endCol    int
	numStmt   int
	count     int
}

func verifyTotalCoverage(profilePath string, threshold float64) error {
	profiles, err := parseCoverageProfiles(profilePath)
	if err != nil {
		return fmt.Errorf("coverage profile を解析できませんでした: %w", err)
	}

	var totalStatements int64
	var coveredStatements int64
	for _, profile := range profiles {
		for _, block := range profile.blocks {
			statements := int64(block.numStmt)
			totalStatements += statements
			if block.count > 0 {
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

func parseCoverageProfiles(profilePath string) ([]coverageProfile, error) {
	file, err := os.Open(profilePath) //nolint:gosec // SOT-ENG-020: 品質ゲートが生成した coverage profile の明示パスだけを読む。
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), coverageScannerMaximumTokenBytes)

	mode := ""
	files := make(map[string]*coverageProfile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if mode == "" {
			const prefix = "mode: "
			if !strings.HasPrefix(line, prefix) || line == prefix {
				return nil, fmt.Errorf("bad mode line: %v", line)
			}
			mode = line[len(prefix):]
			continue
		}
		fileName, block, err := parseCoverageLine(line)
		if err != nil {
			return nil, fmt.Errorf(
				"line %q doesn't match expected format: %w",
				line,
				err,
			)
		}
		profile := files[fileName]
		if profile == nil {
			profile = &coverageProfile{
				fileName: fileName,
				mode:     mode,
			}
			files[fileName] = profile
		}
		profile.blocks = append(profile.blocks, block)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result := make([]coverageProfile, 0, len(files))
	for _, profile := range files {
		if err := mergeCoverageBlocks(profile); err != nil {
			return nil, err
		}
		result = append(result, *profile)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].fileName < result[right].fileName
	})
	return result, nil
}

func mergeCoverageBlocks(profile *coverageProfile) error {
	sort.Slice(profile.blocks, func(left, right int) bool {
		first := profile.blocks[left]
		second := profile.blocks[right]
		return first.startLine < second.startLine ||
			(first.startLine == second.startLine && first.startCol < second.startCol)
	})
	if len(profile.blocks) == 0 {
		return nil
	}
	writeIndex := 1
	for readIndex := 1; readIndex < len(profile.blocks); readIndex++ {
		block := profile.blocks[readIndex]
		last := profile.blocks[writeIndex-1]
		if sameCoverageBlock(block, last) {
			if block.numStmt != last.numStmt {
				return fmt.Errorf(
					"inconsistent NumStmt: changed from %d to %d",
					last.numStmt,
					block.numStmt,
				)
			}
			if profile.mode == "set" {
				profile.blocks[writeIndex-1].count |= block.count
			} else {
				profile.blocks[writeIndex-1].count += block.count
			}
			continue
		}
		profile.blocks[writeIndex] = block
		writeIndex++
	}
	profile.blocks = profile.blocks[:writeIndex]
	return nil
}

func sameCoverageBlock(
	first coverageProfileBlock,
	second coverageProfileBlock,
) bool {
	return first.startLine == second.startLine &&
		first.startCol == second.startCol &&
		first.endLine == second.endLine &&
		first.endCol == second.endCol
}

func parseCoverageLine(line string) (string, coverageProfileBlock, error) {
	end := len(line)
	block := coverageProfileBlock{}

	var err error
	block.count, end, err = seekCoverageIntegerBack(line, ' ', end, "Count")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	block.numStmt, end, err = seekCoverageIntegerBack(line, ' ', end, "NumStmt")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	block.endCol, end, err = seekCoverageIntegerBack(line, '.', end, "EndCol")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	block.endLine, end, err = seekCoverageIntegerBack(line, ',', end, "EndLine")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	block.startCol, end, err = seekCoverageIntegerBack(line, '.', end, "StartCol")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	block.startLine, end, err = seekCoverageIntegerBack(line, ':', end, "StartLine")
	if err != nil {
		return "", coverageProfileBlock{}, err
	}
	fileName := line[:end]
	if fileName == "" {
		return "", coverageProfileBlock{}, errors.New("a FileName cannot be blank")
	}
	return fileName, block, nil
}

func seekCoverageIntegerBack(
	line string,
	separator byte,
	end int,
	what string,
) (int, int, error) {
	for start := end - 1; start >= 0; start-- {
		if line[start] != separator {
			continue
		}
		value, err := strconv.Atoi(line[start+1 : end])
		if err != nil {
			return 0, 0, fmt.Errorf("couldn't parse %q: %w", what, err)
		}
		if value < 0 {
			return 0, 0, fmt.Errorf(
				"negative values are not allowed for %s, found %d",
				what,
				value,
			)
		}
		return value, start, nil
	}
	return 0, 0, fmt.Errorf("couldn't find a %s before %s", string(separator), what)
}
