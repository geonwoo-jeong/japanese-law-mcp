package legalqueryadoption

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

var (
	catalogVersionLinePattern = regexp.MustCompile(
		`(?m)^- ` + "`catalogVersion`" + `: ` + "`" +
			`(unified-query-examples-v[1-9][0-9]*)` + "`" + `$`,
	)
	catalogCorpusVersionLinePattern = regexp.MustCompile(
		`(?m)^- ` + "`corpusVersion`" + `: ` + "`" +
			`(corpus-v[1-9][0-9]*)` + "`" + `$`,
	)
	catalogBaselineVersionLinePattern = regexp.MustCompile(
		`(?m)^- ` + "`baselineVersion`" + `: ` + "`" +
			`(default-[1-9][0-9]*)` + "`" + `$`,
	)
	catalogArtifactReferencePattern = regexp.MustCompile(
		"`" + `(corpus-v[1-9][0-9]*):(semantic|execution):` +
			`([a-z0-9][a-z0-9-]{0,63})` + "`",
	)
)

const (
	maximumCatalogEntries       = 64
	maximumCatalogMarkdownBytes = 1 << 20
)

func verifyCatalog(
	ctx context.Context,
	repository *legalqueryartifact.Repository,
	manifest Manifest,
	corpus legalquerycorpus.Corpus,
) error {
	artifacts, err := catalogArtifactCounts(manifest, corpus)
	if err != nil {
		return err
	}
	docs, err := repository.OpenChild("docs")
	if err != nil {
		return fmt.Errorf("検索例カタログの docs を開けません: %w", err)
	}
	defer func() { _ = docs.Close() }()
	catalog, err := docs.OpenChild("unified-query-examples")
	if err != nil {
		return fmt.Errorf("検索例カタログを開けません: %w", err)
	}
	defer func() { _ = catalog.Close() }()
	entries, err := catalog.ReadDirectory(
		maximumCatalogEntries,
		maximumCatalogMarkdownBytes,
	)
	if err != nil {
		return fmt.Errorf("検索例カタログを列挙できません: %w", err)
	}
	aggregate := sha256.New()
	var index []byte
	markdownCount := 0
	referenceCount := 0
	for _, entry := range entries {
		if err := checkAdoptionContext(ctx); err != nil {
			return err
		}
		if entry.Name() == "history" && entry.Info().IsDir() {
			continue
		}
		if entry.Info().Mode()&os.ModeSymlink != 0 ||
			!entry.Info().Mode().IsRegular() ||
			!strings.HasSuffix(entry.Name(), ".md") {
			return fmt.Errorf("検索例カタログに未知の entry があります")
		}
		raw, err := catalog.ReadRegular(entry.Name(), maximumCatalogMarkdownBytes)
		if err != nil {
			return fmt.Errorf("検索例カタログを読めません: %w", err)
		}
		currentReferences, err := verifyCatalogArtifactReferences(
			raw,
			manifest.CorpusVersion(),
			artifacts,
		)
		if err != nil {
			return err
		}
		referenceCount += currentReferences
		fileSum := sha256.Sum256(raw)
		_, _ = fmt.Fprintf(
			aggregate,
			"%s %s\n",
			entry.Name(),
			hex.EncodeToString(fileSum[:]),
		)
		markdownCount++
		if entry.Name() == "00-index.md" {
			index = append([]byte(nil), raw...)
		}
	}
	if markdownCount == 0 || len(index) == 0 {
		return fmt.Errorf("検索例カタログの必須 Markdown がありません")
	}
	if referenceCount == 0 {
		return fmt.Errorf("検索例カタログに verification_artifact がありません")
	}
	if err := verifyCatalogIndexMetadata(index, manifest); err != nil {
		return err
	}
	if hex.EncodeToString(aggregate.Sum(nil)) != manifest.CatalogSHA256() {
		return fmt.Errorf("catalogSha256 が現行 Markdown と一致しません")
	}
	return nil
}

func catalogArtifactCounts(
	manifest Manifest,
	corpus legalquerycorpus.Corpus,
) (map[string]int, error) {
	if err := corpus.Validate(); err != nil {
		return nil, fmt.Errorf("検索例カタログの corpus が不正です: %w", err)
	}
	corpusManifest := corpus.Manifest()
	if corpusManifest.CorpusVersion() != manifest.CorpusVersion() ||
		corpusManifest.HoldoutDigest() != manifest.HoldoutDigest() {
		return nil, fmt.Errorf("検索例カタログと adoption corpus が一致しません")
	}
	counts := make(map[string]int)
	for _, semanticCase := range corpus.Development() {
		counts[catalogArtifactKey("semantic", semanticCase.CaseID())]++
	}
	for _, semanticCase := range corpus.Holdout() {
		counts[catalogArtifactKey("semantic", semanticCase.CaseID())]++
	}
	for _, executionCase := range corpus.Execution() {
		counts[catalogArtifactKey("execution", executionCase.CaseID())]++
	}
	return counts, nil
}

func verifyCatalogArtifactReferences(
	raw []byte,
	corpusVersion string,
	artifacts map[string]int,
) (int, error) {
	matches := catalogArtifactReferencePattern.FindAllSubmatch(raw, -1)
	for _, match := range matches {
		if string(match[1]) != corpusVersion {
			return 0, fmt.Errorf("verification_artifact の corpusVersion が一致しません")
		}
		if artifacts[catalogArtifactKey(string(match[2]), string(match[3]))] != 1 {
			return 0, fmt.Errorf("verification_artifact を corpus で一意に解決できません")
		}
	}
	return len(matches), nil
}

func catalogArtifactKey(kind, caseID string) string {
	return kind + "\x00" + caseID
}

func verifyCatalogIndexMetadata(index []byte, manifest Manifest) error {
	checks := []struct {
		field    string
		pattern  *regexp.Regexp
		expected string
	}{
		{"catalogVersion", catalogVersionLinePattern, manifest.CatalogVersion()},
		{"corpusVersion", catalogCorpusVersionLinePattern, manifest.CorpusVersion()},
		{"baselineVersion", catalogBaselineVersionLinePattern, manifest.BaselineVersion()},
	}
	for _, check := range checks {
		matches := check.pattern.FindAllSubmatch(index, -1)
		if len(matches) != 1 || string(matches[0][1]) != check.expected {
			return fmt.Errorf("%s が index と一致しません", check.field)
		}
	}
	return nil
}
