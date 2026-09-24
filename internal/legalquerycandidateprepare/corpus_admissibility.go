package legalquerycandidateprepare

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

const minimumCandidateEvaluationCorpusVersion = 13

func validateCandidateEvaluationCorpus(
	schemaVersion int,
	corpusVersion string,
) error {
	if schemaVersion != 2 && schemaVersion != 3 {
		return fmt.Errorf(
			"candidate-evaluation-corpus-admissibility: 新しい request には corpus schema version 2 または 3が必要です",
		)
	}
	return validateCandidateEvaluationCorpusVersion(corpusVersion)
}

func validateCandidateEvaluationCorpusVersion(corpusVersion string) error {
	const prefix = "corpus-v"
	versionText := strings.TrimPrefix(corpusVersion, prefix)
	version, err := strconv.Atoi(versionText)
	if err != nil || version < 1 || corpusVersion != fmt.Sprintf("%s%d", prefix, version) {
		return fmt.Errorf(
			"candidate-evaluation-corpus-admissibility: corpus version %q が不正です",
			corpusVersion,
		)
	}
	if version < minimumCandidateEvaluationCorpusVersion {
		return fmt.Errorf(
			"candidate-evaluation-corpus-admissibility: 新しい request には corpus-v%d 以降が必要です",
			minimumCandidateEvaluationCorpusVersion,
		)
	}
	return nil
}

// SOT-ENG-048: 新しい世代を過去の request へ再結合しない。
func validateCandidateEvaluationCorpusForSchema(handoffVersion, schemaVersion int, corpusVersion string) error {
	switch handoffVersion {
	case legalquerycandidateeval.SchemaVersionV3, legalquerycandidateeval.SchemaVersionV4:
		if schemaVersion != 2 {
			return fmt.Errorf("旧世代の request は corpus schema version 2 を必要とします")
		}
	case legalquerycandidateeval.SchemaVersionV5:
		if schemaVersion != 3 {
			return fmt.Errorf("schema version 5 の request は corpus schema version 3 を必要とします")
		}
	default:
		return fmt.Errorf("候補 request の schema version が未対応です")
	}
	return validateCandidateEvaluationCorpus(schemaVersion, corpusVersion)
}
