package legalquerycandidateprepare

import (
	"fmt"
	"strconv"
	"strings"
)

const minimumCandidateEvaluationCorpusVersion = 13

func validateCandidateEvaluationCorpus(
	schemaVersion int,
	corpusVersion string,
) error {
	if schemaVersion < 2 {
		return fmt.Errorf(
			"candidate-evaluation-corpus-admissibility: 新しい request には corpus schema version 2 以降が必要です",
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
