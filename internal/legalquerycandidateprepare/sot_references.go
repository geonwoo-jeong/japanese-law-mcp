package legalquerycandidateprepare

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

const maximumSOTDocumentBytes = 1 << 20

var (
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	ruleFilenamePattern = regexp.MustCompile(`^[0-9]{2}-[a-z0-9][a-z0-9-]*\.md$`)
)

var sotDomains = [...]struct {
	prefix string
	path   string
}{
	{prefix: "PRODUCT", path: "00-product"},
	{prefix: "SCN", path: "10-scenarios"},
	{prefix: "MODEL", path: "20-model"},
	{prefix: "ARCH", path: "30-architecture"},
	{prefix: "IF", path: "40-interfaces"},
	{prefix: "ENG", path: "50-engineering"},
	{prefix: "DELIVERY", path: "60-delivery"},
}

// BuildRequiredSOTReferences は、固定 index から schema 版に対応する review 集合を解決する。
func BuildRequiredSOTReferences(
	ctx context.Context,
	repositoryRoot string,
	schemaVersion int,
) ([]legalquerycandidateeval.SOTReference, error) {
	if err := checkPrepareContext(ctx); err != nil {
		return nil, err
	}
	repository, err := openPrepareRepository(repositoryRoot)
	if err != nil {
		return nil, err
	}
	defer func() { _ = repository.Close() }()
	indices, err := loadFixedSOTIndices(ctx, repository)
	if err != nil {
		return nil, err
	}
	ids, err := legalquerycandidateeval.RequiredReviewSOTIDsForSchema(schemaVersion)
	if err != nil {
		return nil, err
	}
	references := make([]legalquerycandidateeval.SOTReference, 0, len(ids))
	for _, id := range ids {
		if err := checkPrepareContext(ctx); err != nil {
			return nil, err
		}
		reference, err := resolveRequiredSOT(repository, indices, id)
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	return references, nil
}

func loadFixedSOTIndices(
	ctx context.Context,
	repository *prepareRepository,
) (map[string]map[string]string, error) {
	root, err := repository.Read("sot/00-index.md", maximumSOTDocumentBytes)
	if err != nil {
		return nil, err
	}
	indices := make(map[string]map[string]string, len(sotDomains))
	for _, domain := range sotDomains {
		if err := checkPrepareContext(ctx); err != nil {
			return nil, err
		}
		rootTarget := domain.path + "/00-index.md"
		if !strings.Contains(string(root), "("+rootTarget+")") {
			return nil, fmt.Errorf("SOT root index に %s がありません", rootTarget)
		}
		raw, err := repository.Read("sot/"+rootTarget, maximumSOTDocumentBytes)
		if err != nil {
			return nil, err
		}
		files, err := parseDomainIndex(raw)
		if err != nil {
			return nil, fmt.Errorf("SOT index %s が不正です: %w", domain.path, err)
		}
		indices[domain.prefix] = files
	}
	return indices, nil
}

func parseDomainIndex(raw []byte) (map[string]string, error) {
	files := make(map[string]string)
	for _, match := range markdownLinkPattern.FindAllSubmatch(raw, -1) {
		target := string(match[1])
		if !ruleFilenamePattern.MatchString(target) {
			continue
		}
		number := target[:2]
		if _, duplicate := files[number]; duplicate {
			return nil, fmt.Errorf("番号 %s が重複しています", number)
		}
		files[number] = target
	}
	return files, nil
}

func resolveRequiredSOT(
	repository *prepareRepository,
	indices map[string]map[string]string,
	id string,
) (legalquerycandidateeval.SOTReference, error) {
	prefix, number, err := splitSOTID(id)
	if err != nil {
		return legalquerycandidateeval.SOTReference{}, err
	}
	domainPath, exists := sotDomainPath(prefix)
	if !exists {
		return legalquerycandidateeval.SOTReference{}, fmt.Errorf("SOT domain %s が未対応です", prefix)
	}
	filename, exists := indices[prefix][number]
	if !exists {
		return legalquerycandidateeval.SOTReference{}, fmt.Errorf("SOT %s が index にありません", id)
	}
	raw, err := repository.Read("sot/"+domainPath+"/"+filename, maximumSOTDocumentBytes)
	if err != nil {
		return legalquerycandidateeval.SOTReference{}, err
	}
	if err := validateRequiredSOTDocument(id, raw); err != nil {
		return legalquerycandidateeval.SOTReference{}, err
	}
	return legalquerycandidateeval.SOTReference{
		SOTID: id, SOTDocumentSHA256: legalquerycandidateeval.RawSHA256(raw),
	}, nil
}

func splitSOTID(id string) (string, string, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 3 || parts[0] != "SOT" || len(parts[2]) != 3 {
		return "", "", fmt.Errorf("SOT ID %q が不正です", id)
	}
	number, err := strconv.Atoi(parts[2])
	if err != nil || number < 1 || number > 99 {
		return "", "", fmt.Errorf("SOT ID %q の番号が不正です", id)
	}
	return parts[1], fmt.Sprintf("%02d", number), nil
}

func sotDomainPath(prefix string) (string, bool) {
	for _, domain := range sotDomains {
		if domain.prefix == prefix {
			return domain.path, true
		}
	}
	return "", false
}

func validateRequiredSOTDocument(id string, raw []byte) error {
	firstLine, _, _ := strings.Cut(string(raw), "\n")
	if !strings.HasPrefix(firstLine, "# "+id+":") {
		return fmt.Errorf("SOT %s の heading が一致しません", id)
	}
	if !strings.Contains(string(raw), "\n- 状態: 有効\n") {
		return fmt.Errorf("SOT %s は有効ではありません", id)
	}
	return nil
}
