package legalquerycandidateeval

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

func openCandidateEvaluationRoot(
	repository *legalqueryartifact.Repository,
) (*legalqueryartifact.Repository, error) {
	testdata, err := repository.OpenChild("testdata")
	if err != nil {
		return nil, fmt.Errorf("testdata directory を開けません: %w", err)
	}
	defer func() { _ = testdata.Close() }()
	legalquery, err := testdata.OpenChild("legalquery")
	if err != nil {
		return nil, fmt.Errorf("legalquery directory を開けません: %w", err)
	}
	defer func() { _ = legalquery.Close() }()
	root, err := legalquery.OpenChild("candidate-evaluations")
	if err != nil {
		return nil, fmt.Errorf("candidate evaluation directory を開けません: %w", err)
	}
	return root, nil
}

func validateRootEntries(
	root *legalqueryartifact.Repository,
) (preparationRootLayout, error) {
	// SOT-ENG-038: Git が空 directory を保持しないため、未評価時の二履歴 root だけは
	// 不在を論理的な空として扱う。存在する場合は後段で sentinel を含め空以外を拒否する。
	entries, err := root.ReadDirectory(7, maximumSchemaBytes+maximumPointerBytes)
	if err != nil {
		return preparationRootLayout{}, fmt.Errorf("candidate evaluation root を列挙できません: %w", err)
	}
	allowed := map[string]rootEntry{
		"content-manifests":   {directory: true, required: true},
		"current.json":        {required: true},
		"failed-reports":      {directory: true},
		"requests":            {directory: true, required: true},
		"results":             {directory: true},
		"review-attestations": {directory: true, required: true},
		"schema-v2.json":      {required: true},
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		expected, exists := allowed[entry.Name()]
		if !exists || !matchesRootEntry(entry, expected) {
			return preparationRootLayout{}, fmt.Errorf("candidate evaluation root の entry 集合が不正です")
		}
		seen[entry.Name()] = struct{}{}
	}
	for name, expected := range allowed {
		if _, exists := seen[name]; expected.required && !exists {
			return preparationRootLayout{}, fmt.Errorf("candidate evaluation root の必須 entry がありません")
		}
	}
	return preparationRootLayout{
		resultsPresent:       containsRootEntry(seen, "results"),
		failedReportsPresent: containsRootEntry(seen, "failed-reports"),
	}, nil
}

type rootEntry struct {
	directory bool
	required  bool
}

type preparationRootLayout struct {
	resultsPresent       bool
	failedReportsPresent bool
}

func matchesRootEntry(
	entry legalqueryartifact.DirectoryEntry,
	expected rootEntry,
) bool {
	info := entry.Info()
	return info.Mode()&os.ModeSymlink == 0 &&
		((expected.directory && info.IsDir()) ||
			(!expected.directory && info.Mode().IsRegular()))
}

func containsRootEntry(entries map[string]struct{}, name string) bool {
	_, exists := entries[name]
	return exists
}

func loadSchema(root *legalqueryartifact.Repository) (SchemaV2, error) {
	raw, err := root.ReadRegular("schema-v2.json", maximumSchemaBytes)
	if err != nil {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema を読めません: %w", err)
	}
	if !bytes.Equal(raw, CanonicalSchemaV2()) {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema が固定済み schema v2 と一致しません")
	}
	return ParseSchemaV2(raw)
}

func loadPointer(
	ctx context.Context,
	root *legalqueryartifact.Repository,
	schema SchemaV2,
) (PointerDocument, error) {
	raw, err := root.ReadRegular("current.json", maximumPointerBytes)
	if err != nil {
		return PointerDocument{}, fmt.Errorf("candidate evaluation pointer を読めません: %w", err)
	}
	if err := schema.Validate(ctx, raw); err != nil {
		return PointerDocument{}, err
	}
	return DecodePointer(raw)
}

func requireEmptyHistory(
	root *legalqueryartifact.Repository,
	layout preparationRootLayout,
) error {
	if layout.resultsPresent {
		if err := requireEmptyDirectory(
			root, "results", maximumResultFiles, maximumResultTotalBytes,
		); err != nil {
			return err
		}
	}
	if layout.failedReportsPresent {
		return requireEmptyDirectory(
			root, "failed-reports", maximumFailedReportFiles, maximumFailedReportTotal,
		)
	}
	return nil
}

func requireEmptyDirectory(
	root *legalqueryartifact.Repository,
	name string,
	maximumEntries int,
	maximumBytes int64,
) error {
	directory, err := root.OpenChild(name)
	if err != nil {
		return fmt.Errorf("%s directory を開けません: %w", name, err)
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDirectory(maximumEntries, maximumBytes)
	if err != nil {
		return fmt.Errorf("%s directory を列挙できません: %w", name, err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("評価準備では %s directory は空でなければなりません", name)
	}
	return nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("candidate evaluation context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("candidate evaluation 読取りが取り消されました: %w", err)
	}
	return nil
}
