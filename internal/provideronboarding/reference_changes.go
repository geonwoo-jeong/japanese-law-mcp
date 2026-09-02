package provideronboarding

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	sotIDPattern = regexp.MustCompile(
		`^SOT-(PROD|SCN|MODEL|ARCH|IF|ENG|DEL)-([0-9]{3})$`,
	)
	sotHeadingPattern = regexp.MustCompile(
		`(?m)^# (SOT-[A-Z]+-[0-9]{3}): .+$`,
	)
	sotStatusPattern = regexp.MustCompile(
		`(?m)^- 状態: (草案|有効|廃止)$`,
	)
	sotSuccessorLinePattern = regexp.MustCompile(`(?m)^- 後継:.*$`)
	sotSuccessorPattern     = regexp.MustCompile(
		`^- 後継: \[(SOT-[A-Z]+-[0-9]{3}): [^\]\r\n]+\]\(([^)#\r\n]+)\)$`,
	)
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
)

type sotDocument struct {
	id              string
	path            string
	status          string
	successorID     string
	successorTarget string
}

func classifyTraceOnlyMatrixPaths(
	ctx context.Context,
	client gitClient,
	repository string,
	resolved comparison,
	changes changedPathSet,
	classify interfaceSOTReferenceClassifier,
) (map[string]struct{}, error) {
	if classify == nil {
		return nil, errors.New("matrix の SOT 参照変更分類が指定されていません")
	}
	traceOnly := make(map[string]struct{})
	documents := make(map[string]sotDocument)
	for _, changedPath := range changes.paths {
		providerID, matrixPath := matrixProviderID(changedPath)
		if !matrixPath {
			continue
		}
		previous, current, bothExist, err := client.comparisonPathContents(
			ctx,
			repository,
			resolved,
			changes,
			changedPath,
		)
		if err != nil {
			return nil, err
		}
		if !bothExist {
			continue
		}
		change, referenceOnly, err := classify(
			repository,
			providerID,
			previous,
			current,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"matrix の SOT 参照変更を分類できません: %s: %w",
				changedPath,
				err,
			)
		}
		if !referenceOnly {
			continue
		}
		direct, err := replacementsUseDirectActiveSuccessors(
			repository,
			change.replacements,
			documents,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"matrix の SOT 後継関係を確認できません: %s: %w",
				changedPath,
				err,
			)
		}
		if direct {
			traceOnly[changedPath] = struct{}{}
		}
	}
	return traceOnly, nil
}

func replacementsUseDirectActiveSuccessors(
	repository string,
	replacements []interfaceSOTReferenceReplacement,
	documents map[string]sotDocument,
) (bool, error) {
	if len(replacements) == 0 {
		return false, errors.New("SOT 参照変更に置換対象がありません")
	}
	for _, replacement := range replacements {
		previous, err := cachedSOTDocument(
			repository,
			replacement.previousSOTID,
			documents,
		)
		if err != nil {
			return false, err
		}
		current, err := cachedSOTDocument(
			repository,
			replacement.currentSOTID,
			documents,
		)
		if err != nil {
			return false, err
		}
		if previous.status != "廃止" || current.status != "有効" ||
			previous.successorID != current.id {
			return false, nil
		}
		resolvedSuccessor := filepath.Clean(filepath.Join(
			filepath.Dir(previous.path),
			filepath.FromSlash(previous.successorTarget),
		))
		if resolvedSuccessor != filepath.Clean(current.path) {
			return false, nil
		}
	}
	return true, nil
}

func cachedSOTDocument(
	repository, id string,
	documents map[string]sotDocument,
) (sotDocument, error) {
	if document, exists := documents[id]; exists {
		return document, nil
	}
	document, err := loadSOTDocument(repository, id)
	if err != nil {
		return sotDocument{}, err
	}
	documents[id] = document
	return document, nil
}

func loadSOTDocument(repository, id string) (sotDocument, error) {
	identifier := sotIDPattern.FindStringSubmatch(id)
	if len(identifier) != 3 {
		return sotDocument{}, fmt.Errorf("SOT ID が不正です: %q", id)
	}
	domainDirectory, ok := sotDomainDirectory(identifier[1])
	if !ok {
		return sotDocument{}, fmt.Errorf("SOT domain が不正です: %q", id)
	}
	fileNumber := identifier[2][len(identifier[2])-2:]
	pattern := filepath.Join(
		repository,
		"sot",
		domainDirectory,
		fileNumber+"-*.md",
	)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return sotDocument{}, fmt.Errorf("SOT file を列挙できません: %w", err)
	}
	if len(matches) != 1 {
		return sotDocument{}, fmt.Errorf(
			"SOT ID に対応する file を一つに確定できません: %s",
			id,
		)
	}
	relativePath, err := filepath.Rel(repository, matches[0])
	if err != nil {
		return sotDocument{}, fmt.Errorf("SOT file の path を解決できません: %w", err)
	}
	repositoryPath := filepath.ToSlash(relativePath)
	content, exists, err := readWorkingTreePath(repository, repositoryPath)
	if err != nil || !exists {
		if err == nil {
			err = errors.New("SOT file がありません")
		}
		return sotDocument{}, fmt.Errorf("SOT file を読み取れません: %s: %w", id, err)
	}
	if err := verifySOTIndexEntry(repository, repositoryPath); err != nil {
		return sotDocument{}, err
	}

	headings := sotHeadingPattern.FindAllStringSubmatch(string(content), -1)
	statuses := sotStatusPattern.FindAllStringSubmatch(string(content), -1)
	if len(headings) != 1 || headings[0][1] != id || len(statuses) != 1 {
		return sotDocument{}, fmt.Errorf("SOT metadata が不正です: %s", repositoryPath)
	}
	document := sotDocument{
		id:     id,
		path:   filepath.Clean(matches[0]),
		status: statuses[0][1],
	}
	successorLines := sotSuccessorLinePattern.FindAllString(string(content), -1)
	if len(successorLines) > 1 {
		return sotDocument{}, fmt.Errorf("SOT の後継指定が重複しています: %s", id)
	}
	if len(successorLines) == 1 {
		successor := sotSuccessorPattern.FindStringSubmatch(successorLines[0])
		if len(successor) != 3 {
			return sotDocument{}, fmt.Errorf("SOT の後継指定が不正です: %s", id)
		}
		document.successorID = successor[1]
		document.successorTarget = successor[2]
	}
	return document, nil
}

func verifySOTIndexEntry(repository, repositoryPath string) error {
	indexPath := filepath.ToSlash(filepath.Join(filepath.Dir(repositoryPath), "00-index.md"))
	content, exists, err := readWorkingTreePath(repository, indexPath)
	if err != nil || !exists {
		if err == nil {
			err = errors.New("SOT index がありません")
		}
		return fmt.Errorf("SOT index を読み取れません: %s: %w", indexPath, err)
	}
	wanted := filepath.Base(repositoryPath)
	count := 0
	for _, link := range markdownLinkPattern.FindAllStringSubmatch(string(content), -1) {
		target := strings.SplitN(link[1], "#", 2)[0]
		if target == wanted {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf(
			"SOT index が規則 file を一回だけ参照していません: %s",
			repositoryPath,
		)
	}
	return nil
}

func sotDomainDirectory(domain string) (string, bool) {
	directories := map[string]string{
		"PROD":  "00-product",
		"SCN":   "10-scenarios",
		"MODEL": "20-model",
		"ARCH":  "30-architecture",
		"IF":    "40-interfaces",
		"ENG":   "50-engineering",
		"DEL":   "60-delivery",
	}
	directory, ok := directories[domain]
	return directory, ok
}
