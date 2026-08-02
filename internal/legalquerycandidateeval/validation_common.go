package legalquerycandidateeval

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

var (
	sha256Pattern              = regexp.MustCompile(`^[0-9a-f]{64}$`)
	candidateContentIDPattern  = regexp.MustCompile(`^candidate-content-sha256-[0-9a-f]{64}$`)
	reviewAttestationIDPattern = regexp.MustCompile(`^review-sha256-[0-9a-f]{64}$`)
	evaluationIDPattern        = regexp.MustCompile(`^evaluation-sha256-[0-9a-f]{64}$`)
	evaluatorVersionPattern    = regexp.MustCompile(`^legal-query-evaluator-v[1-9][0-9]*$`)
	baselineVersionPattern     = regexp.MustCompile(`^default-[1-9][0-9]*$`)
	authorityIDPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	moduleSumPattern           = regexp.MustCompile(`^h1:[A-Za-z0-9+/]{43}=$`)
	goLanguageVersionPattern   = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+(\.[0-9]+)?$`)
	goToolchainVersionPattern  = regexp.MustCompile(`^go[1-9][0-9]*\.[0-9]+(\.[0-9]+)?$`)
	goDebugNamePattern         = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func validatePointer(document PointerDocument) error {
	if document.ArtifactKind != ArtifactKindPointer ||
		!isSupportedSchemaVersion(document.SchemaVersion) ||
		!evaluationIDPattern.MatchString(document.EvaluationID) {
		return fmt.Errorf("candidate evaluation pointer の値が不正です")
	}
	return nil
}

func isSupportedSchemaVersion(schemaVersion int) bool {
	return schemaVersion == SchemaVersionV2 || schemaVersion == SchemaVersionV3
}

func validateSHA256(name, value string) error {
	if !sha256Pattern.MatchString(value) {
		return fmt.Errorf("%s が小文字 SHA-256 ではありません", name)
	}
	return nil
}

func validateMachineString(name, value string, maximumBytes int) error {
	if value == "" || len(value) > maximumBytes {
		return fmt.Errorf("%s の長さが不正です", name)
	}
	for index := range len(value) {
		if value[index] < 0x21 || value[index] > 0x7e {
			return fmt.Errorf("%s は表示可能 ASCII でなければなりません", name)
		}
	}
	return nil
}

func validateRepositoryPath(name, value string) error {
	if value == "" || len(value) > 512 || !fs.ValidPath(value) ||
		path.IsAbs(value) || path.Clean(value) != value ||
		strings.Contains(value, "\\") || hasParentSegment(value) {
		return fmt.Errorf("%s は正規 repository-relative POSIX path ではありません", name)
	}
	return nil
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func validateSortedUniqueStrings(name string, values []string, minimum, maximum int) error {
	if values == nil || len(values) < minimum || len(values) > maximum {
		return fmt.Errorf("%s の件数が不正です", name)
	}
	for index, value := range values {
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("%s は byte 昇順かつ一意でなければなりません", name)
		}
	}
	return nil
}

func validateFileDigests(files []FileDigest, minimum, maximum int) error {
	if files == nil || len(files) < minimum || len(files) > maximum {
		return fmt.Errorf("file digest の件数が不正です")
	}
	previous := ""
	for index, file := range files {
		if err := validateRepositoryPath("path", file.Path); err != nil {
			return err
		}
		if index > 0 && previous >= file.Path {
			return fmt.Errorf("file path は byte 昇順かつ一意でなければなりません")
		}
		if err := validateSHA256("rawSha256", file.RawSHA256); err != nil {
			return err
		}
		previous = file.Path
	}
	return nil
}
